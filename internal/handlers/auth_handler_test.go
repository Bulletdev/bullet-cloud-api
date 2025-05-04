package handlers_test

import (
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/users"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupAuthTest sets up mocks, handler, and router for auth tests.
func setupAuthTest(t *testing.T) (*users.MockUserRepository, *handlers.AuthHandler, *mux.Router) {
	// Use the base setup for common parts (user mock, middleware, router)
	mockUserRepo, _, router := setupBaseTest(t)

	// Instantiate the real password hasher
	hasher := auth.NewBcryptPasswordHasher()

	// Provide test JWT configuration
	testJwtSecret := "test-secret-for-jwt-please-change"
	testJwtExpiry := time.Hour * 1 // Example expiry for tests

	// Create the AuthHandler, now passing the hasher
	authHandler := handlers.NewAuthHandler(mockUserRepo, hasher, testJwtSecret, testJwtExpiry)

	// Define routes handled by AuthHandler (no middleware needed for these)
	apiV1 := router.PathPrefix("/api/auth").Subrouter()
	apiV1.HandleFunc("/register", authHandler.Register).Methods("POST")
	apiV1.HandleFunc("/login", authHandler.Login).Methods("POST")

	return mockUserRepo, authHandler, router
}

// Placeholder for TestAuthHandler_RegisterHandler
func TestAuthHandler_RegisterHandler(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		mockFindByEmailErr error // Error returned by FindByEmail
		mockCreateErr      error // Error returned by Create
		expectedStatus     int
		expectedBody       string
	}{
		{
			name:               "Success",
			body:               `{"name":"Test User","email":"new@example.com","password":"password123"}`,
			mockFindByEmailErr: users.ErrUserNotFound, // Expect user NOT to be found
			mockCreateErr:      nil,
			expectedStatus:     http.StatusCreated,
			expectedBody:       "new@example.com", // Check for email in response
		},
		{
			name:               "Failure - Invalid JSON",
			body:               `{"name":"Test User",}`, // Malformed JSON
			mockFindByEmailErr: nil,                     // Won't be called
			mockCreateErr:      nil,                     // Won't be called
			expectedStatus:     http.StatusBadRequest,
			expectedBody:       `{"error":"invalid request body"}`,
		},
		{
			name:               "Failure - Missing Name",
			body:               `{"email":"missing@name.com","password":"password123"}`,
			mockFindByEmailErr: nil, // Won't be called
			mockCreateErr:      nil, // Won't be called
			expectedStatus:     http.StatusBadRequest,
			expectedBody:       `{"error":"name, email, and password are required"}`,
		},
		{
			name:               "Failure - Missing Email",
			body:               `{"name":"Missing Email","password":"password123"}`,
			mockFindByEmailErr: nil, // Won't be called
			mockCreateErr:      nil, // Won't be called
			expectedStatus:     http.StatusBadRequest,
			expectedBody:       `{"error":"name, email, and password are required"}`,
		},
		{
			name:               "Failure - Missing Password",
			body:               `{"name":"Missing Password","email":"missing@pass.com"}`,
			mockFindByEmailErr: nil, // Won't be called
			mockCreateErr:      nil, // Won't be called
			expectedStatus:     http.StatusBadRequest,
			expectedBody:       `{"error":"name, email, and password are required"}`,
		},
		{
			name:               "Failure - Email Already Exists",
			body:               `{"name":"Existing User","email":"existing@example.com","password":"password123"}`,
			mockFindByEmailErr: nil, // Simulate user FOUND
			mockCreateErr:      nil, // Won't be called
			expectedStatus:     http.StatusConflict,
			expectedBody:       `{"error":"email already registered"}`,
		},
		{
			name:               "Failure - FindByEmail Repo Error",
			body:               `{"name":"Repo Find Error","email":"finderr@example.com","password":"password123"}`,
			mockFindByEmailErr: errors.New("db find error"),
			mockCreateErr:      nil, // Won't be called
			expectedStatus:     http.StatusInternalServerError,
			expectedBody:       `{"error":"failed to check email existence"}`,
		},
		{
			name:               "Failure - Create Repo Error",
			body:               `{"name":"Repo Create Error","email":"createerr@example.com","password":"password123"}`,
			mockFindByEmailErr: users.ErrUserNotFound, // User not found
			mockCreateErr:      errors.New("db create error"),
			expectedStatus:     http.StatusInternalServerError,
			expectedBody:       `{"error":"failed to register user"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run for isolation
			mockUserRepo, _, router := setupAuthTest(t)

			// --- Mock Setup ---
			shouldMockFindByEmail := tc.expectedStatus != http.StatusBadRequest // Don't mock if validation fails early
			if shouldMockFindByEmail {
				// FindByEmail expects an email string
				var expectedEmail string
				if tc.body != `{"name":"Test User",}` { // Extract email unless JSON is invalid
					// Quick way to get email for mock setup - might need a better method if body varies more
					if strings.Contains(tc.body, "new@example.com") {
						expectedEmail = "new@example.com"
					} else if strings.Contains(tc.body, "existing@example.com") {
						expectedEmail = "existing@example.com"
					} else if strings.Contains(tc.body, "finderr@example.com") {
						expectedEmail = "finderr@example.com"
					} else if strings.Contains(tc.body, "createerr@example.com") {
						expectedEmail = "createerr@example.com"
					}
				}
				if expectedEmail != "" {
					// If FindByEmail should return nil (user found), we provide a dummy user.
					// If it should return ErrUserNotFound, we provide nil user and the error.
					var userReturn *models.User
					if tc.mockFindByEmailErr == nil { // Case: User Found (Email Exists)
						userReturn = &models.User{ID: uuid.New(), Email: expectedEmail}
					}
					mockUserRepo.On("FindByEmail", mock.Anything, expectedEmail).Return(userReturn, tc.mockFindByEmailErr).Once()
				}
			}

			shouldMockCreate := tc.expectedStatus != http.StatusBadRequest &&
				tc.expectedStatus != http.StatusConflict &&
				tc.mockFindByEmailErr == users.ErrUserNotFound // Only mock create if FindByEmail reported 'not found'

			if shouldMockCreate {
				// Expect Create to be called with context, name, email, hash
				mockUserRepo.On(
					"Create",
					mock.Anything,                 // ctx
					mock.AnythingOfType("string"), // name
					mock.AnythingOfType("string"), // email
					mock.AnythingOfType("string"), // passwordHash
				).Return(
					// This function provides the *models.User return value
					func(ctx context.Context, name, email, hash string) *models.User {
						if tc.mockCreateErr != nil {
							return nil // Return nil user if mock error is set
						}
						// Simulate successful creation
						return &models.User{
							ID:    uuid.New(), // Generate dynamic ID
							Name:  name,
							Email: email,
							// PasswordHash is not typically returned
							CreatedAt: time.Now(), // Set dynamic timestamps
							UpdatedAt: time.Now(),
						}
					},
					// This is the error return value
					tc.mockCreateErr,
				).Once()
			}

			// --- Request Execution ---
			req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// --- Assertions ---
			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")
			assert.Contains(t, rr.Body.String(), tc.expectedBody, "Response body mismatch")
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// Placeholder for TestAuthHandler_LoginHandler
func TestAuthHandler_LoginHandler(t *testing.T) {
	// Use the real hasher from setup to create a valid hash for tests
	hasher := auth.NewBcryptPasswordHasher()
	correctPassword := "password123"
	hashedPassword, err := hasher.HashPassword(correctPassword)
	assert.NoError(t, err, "Failed to hash password during test setup")

	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		Name:         "Test User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name                  string
		body                  string
		mockFindByEmailReturn *models.User // User returned by FindByEmail
		mockFindByEmailErr    error        // Error returned by FindByEmail
		expectedStatus        int
		expectedBody          string // Check for token presence or error message
	}{
		{
			name:                  "Success",
			body:                  `{"email":"test@example.com","password":"password123"}`,
			mockFindByEmailReturn: testUser, // User found with correct hash
			mockFindByEmailErr:    nil,
			expectedStatus:        http.StatusOK,
			expectedBody:          `"token":"`, // Just check that a token key exists
		},
		{
			name:                  "Failure - Invalid JSON",
			body:                  `{"email":"test@example.com"`, // Malformed
			mockFindByEmailReturn: nil,
			mockFindByEmailErr:    nil, // Won't be called
			expectedStatus:        http.StatusBadRequest,
			expectedBody:          `{"error":"invalid request body"}`,
		},
		{
			name:                  "Failure - Missing Email",
			body:                  `{"password":"password123"}`,
			mockFindByEmailReturn: nil,
			mockFindByEmailErr:    nil, // Won't be called
			expectedStatus:        http.StatusBadRequest,
			expectedBody:          `{"error":"email and password are required"}`,
		},
		{
			name:                  "Failure - Missing Password",
			body:                  `{"email":"test@example.com"}`,
			mockFindByEmailReturn: nil,
			mockFindByEmailErr:    nil, // Won't be called
			expectedStatus:        http.StatusBadRequest,
			expectedBody:          `{"error":"email and password are required"}`,
		},
		{
			name:                  "Failure - User Not Found",
			body:                  `{"email":"notfound@example.com","password":"password123"}`,
			mockFindByEmailReturn: nil,
			mockFindByEmailErr:    users.ErrUserNotFound,
			expectedStatus:        http.StatusUnauthorized,
			expectedBody:          `{"error":"invalid email or password"}`,
		},
		{
			name:                  "Failure - Incorrect Password",
			body:                  `{"email":"test@example.com","password":"wrongpassword"}`,
			mockFindByEmailReturn: testUser, // User found, but password check will fail
			mockFindByEmailErr:    nil,
			expectedStatus:        http.StatusUnauthorized,
			expectedBody:          `{"error":"invalid email or password"}`,
		},
		{
			name:                  "Failure - FindByEmail Repo Error",
			body:                  `{"email":"dberror@example.com","password":"password123"}`,
			mockFindByEmailReturn: nil,
			mockFindByEmailErr:    errors.New("db find error"),
			expectedStatus:        http.StatusInternalServerError,
			expectedBody:          `{"error":"login failed"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run for isolation
			mockUserRepo, _, router := setupAuthTest(t)

			// --- Mock Setup ---
			shouldMockFindByEmail := tc.expectedStatus != http.StatusBadRequest
			if shouldMockFindByEmail {
				// Extract expected email (simple approach)
				var expectedEmail string
				if strings.Contains(tc.body, "test@example.com") {
					expectedEmail = "test@example.com"
				} else if strings.Contains(tc.body, "notfound@example.com") {
					expectedEmail = "notfound@example.com"
				} else if strings.Contains(tc.body, "dberror@example.com") {
					expectedEmail = "dberror@example.com"
				}
				if expectedEmail != "" {
					mockUserRepo.On("FindByEmail", mock.Anything, expectedEmail).Return(tc.mockFindByEmailReturn, tc.mockFindByEmailErr).Once()
				}
			}

			// --- Request Execution ---
			req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// --- Assertions ---
			assert.Equal(t, tc.expectedStatus, rr.Code, "Status code mismatch")
			assert.Contains(t, rr.Body.String(), tc.expectedBody, "Response body mismatch")
			mockUserRepo.AssertExpectations(t)
		})
	}
}
