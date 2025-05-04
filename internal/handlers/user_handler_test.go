package handlers_test

import (
	"bullet-cloud-api/internal/addresses" // Mock needed
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/users" // Mock needed
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

// setupUserHandlerTest creates mocks, handler, middleware, and router for user/address tests.
func setupUserHandlerTest(t *testing.T) (*users.MockUserRepository, *addresses.MockAddressRepository, *handlers.UserHandler, *auth.Middleware, *mux.Router) {
	// Use the base setup for common parts
	mockUserRepo, authMiddleware, router := setupBaseTest(t)

	// Add specific mocks for this handler
	mockAddressRepo := new(addresses.MockAddressRepository)

	// Create the handler with its dependencies
	userHandler := handlers.NewUserHandler(mockUserRepo, mockAddressRepo)

	// Define API routes handled by UserHandler
	apiV1 := router.PathPrefix("/api").Subrouter()
	userRoutes := apiV1.PathPrefix("/users").Subrouter()
	userRoutes.Use(authMiddleware.Authenticate) // All user routes require auth

	// /api/users/me
	userRoutes.HandleFunc("/me", userHandler.GetMe).Methods("GET")

	// /api/users/{userId}/addresses
	addressRoutes := userRoutes.PathPrefix("/{userId:[0-9a-fA-F-]+}/addresses").Subrouter()
	addressRoutes.HandleFunc("", userHandler.ListAddresses).Methods("GET")
	addressRoutes.HandleFunc("", userHandler.AddAddress).Methods("POST")
	addressRoutes.HandleFunc("/{addressId:[0-9a-fA-F-]+}", userHandler.UpdateAddress).Methods("PUT")
	addressRoutes.HandleFunc("/{addressId:[0-9a-fA-F-]+}", userHandler.DeleteAddress).Methods("DELETE")
	addressRoutes.HandleFunc("/{addressId:[0-9a-fA-F-]+}/default", userHandler.SetDefaultAddress).Methods("POST")

	return mockUserRepo, mockAddressRepo, userHandler, authMiddleware, router
}

func TestUserHandler_GetMe(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret) // Uses helper from product test
	foundUser := &models.User{ID: testUserID, Name: "Test User", Email: "test@example.com", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	tests := []struct {
		name              string
		mockUserReturnMid *models.User // Middleware FindByID return
		mockUserErrMid    error        // Middleware FindByID error
		mockUserReturnHnd *models.User // Handler FindByID return
		mockUserErrHnd    error        // Handler FindByID error
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Success",
			mockUserReturnMid: foundUser, // Middleware finds user
			mockUserErrMid:    nil,
			mockUserReturnHnd: foundUser, // Handler finds user
			mockUserErrHnd:    nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      "test@example.com", // Check for email in response
		},
		{
			name:              "Failure - Handler User Not Found",
			mockUserReturnMid: foundUser, // Middleware finds user
			mockUserErrMid:    nil,
			mockUserReturnHnd: nil, // Handler doesn't find user
			mockUserErrHnd:    users.ErrUserNotFound,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      `{"error":"user not found"}`,
		},
		{
			name:              "Failure - Handler Repo Error",
			mockUserReturnMid: foundUser, // Middleware finds user
			mockUserErrMid:    nil,
			mockUserReturnHnd: nil, // Handler repo fails
			mockUserErrHnd:    errors.New("db error"),
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to retrieve user data"}`,
		},
		{
			name:              "Failure - Middleware User Check Fails",
			mockUserReturnMid: nil, // Middleware doesn't find user
			mockUserErrMid:    users.ErrUserNotFound,
			mockUserReturnHnd: nil, // Handler FindByID won't be called
			mockUserErrHnd:    nil,
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"user associated with token not found"}`,
		},
		{
			name:              "Failure - No Auth Token",
			mockUserReturnMid: nil, // Middleware FindByID won't be called
			mockUserErrMid:    nil,
			mockUserReturnHnd: nil, // Handler FindByID won't be called
			mockUserErrHnd:    nil,
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run for isolation
			mockUserRepo, _, _, _, router := setupUserHandlerTest(t)

			// Mock middleware user check (if token is expected)
			if tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock handler user check (only if middleware succeeds)
			if tc.mockUserErrMid == nil && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnHnd, tc.mockUserErrHnd).Once()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			// No need to assert mockAddressRepo expectations here
		})
	}
}

func TestUserHandler_ListAddresses(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	anotherUserID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)
	userForToken := &models.User{ID: testUserID} // User associated with the token

	addresses := []models.Address{
		{ID: uuid.New(), UserID: testUserID, Street: "123 Main St", City: "Anytown", IsDefault: true},
		{ID: uuid.New(), UserID: testUserID, Street: "456 Side St", City: "Anytown", IsDefault: false},
	}

	tests := []struct {
		name              string
		targetUserIDStr   string       // User ID in the URL
		mockUserReturnMid *models.User // Middleware FindByID return
		mockUserErrMid    error        // Middleware FindByID error
		mockAddrReturn    []models.Address
		mockAddrErr       error
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Success - List Own Addresses",
			targetUserIDStr:   testUserID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrReturn:    addresses,
			mockAddrErr:       nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      "123 Main St", // Check for street name
		},
		{
			name:              "Success - List Own Addresses (Empty)",
			targetUserIDStr:   testUserID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrReturn:    []models.Address{}, // Empty list
			mockAddrErr:       nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      `[]`,
		},
		{
			name:              "Failure - Repository Error",
			targetUserIDStr:   testUserID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrReturn:    nil,
			mockAddrErr:       errors.New("db error"),
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to retrieve addresses"}`,
		},
		{
			name:              "Failure - Unauthorized (Different User ID)",
			targetUserIDStr:   anotherUserID.String(), // Requesting addresses for another user
			mockUserReturnMid: userForToken,           // Token belongs to testUserID
			mockUserErrMid:    nil,
			mockAddrReturn:    nil, // Won't be called
			mockAddrErr:       nil,
			expectedStatus:    http.StatusForbidden,
			expectedBody:      `{"error":"forbidden"}`,
		},
		{
			name:              "Failure - Invalid User ID in URL",
			targetUserIDStr:   "not-a-uuid",
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrReturn:    nil, // Won't be called
			mockAddrErr:       nil,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"invalid user ID in URL"}`,
		},
		{
			name:              "Failure - Middleware User Check Fails",
			targetUserIDStr:   testUserID.String(),
			mockUserReturnMid: nil,
			mockUserErrMid:    users.ErrUserNotFound,
			mockAddrReturn:    nil, // Won't be called
			mockAddrErr:       nil,
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"user associated with token not found"}`,
		},
		{
			name:              "Failure - No Auth Token",
			targetUserIDStr:   testUserID.String(),
			mockUserReturnMid: nil, // Won't be called
			mockUserErrMid:    nil,
			mockAddrReturn:    nil, // Won't be called
			mockAddrErr:       nil,
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo, mockAddressRepo, _, _, router := setupUserHandlerTest(t)

			// Mock middleware check (only if token is expected and UserID is valid format)
			if tc.targetUserIDStr != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock address repo call (only if middleware succeeds and user is authorized)
			parsedTargetID, _ := uuid.Parse(tc.targetUserIDStr)
			if tc.mockUserErrMid == nil &&
				tc.expectedBody != `{"error":"authorization header required"}` &&
				tc.expectedStatus != http.StatusForbidden &&
				tc.expectedStatus != http.StatusBadRequest &&
				tc.targetUserIDStr != "not-a-uuid" {
				mockAddressRepo.On("FindByUserID", mock.Anything, parsedTargetID).Return(tc.mockAddrReturn, tc.mockAddrErr).Once()
			}

			reqURL := "/api/users/" + tc.targetUserIDStr + "/addresses"
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockAddressRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_AddAddress(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	anotherUserID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)
	userForToken := &models.User{ID: testUserID}

	tests := []struct {
		name              string
		targetUserIDStr   string       // User ID in the URL
		body              string       // Request body
		mockUserReturnMid *models.User // Middleware FindByID return
		mockUserErrMid    error        // Middleware FindByID error
		mockAddrCreateErr error
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Success - Add Own Address",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA","is_default":true}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil,
			expectedStatus:    http.StatusCreated,
			expectedBody:      "789 New Ave",
		},
		{
			name:              "Failure - Invalid JSON",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"Invalid",}`, // Invalid JSON
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"invalid request body"}`,
		},
		{
			name:              "Failure - Missing Fields",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"Only Street"}`, // Missing other fields
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"all address fields are required"}`,
		},
		{
			name:              "Failure - Unauthorized (Different User ID)",
			targetUserIDStr:   anotherUserID.String(),
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA"}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil, // Won't be called
			expectedStatus:    http.StatusForbidden,
			expectedBody:      `{"error":"forbidden"}`,
		},
		{
			name:              "Failure - Invalid User ID in URL",
			targetUserIDStr:   "not-a-uuid",
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA"}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil, // Won't be called
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"invalid user ID in URL"}`,
		},
		{
			name:              "Failure - Repository Error",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA"}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrCreateErr: errors.New("db create error"),
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to create address"}`,
		},
		{
			name:              "Failure - Middleware User Check Fails",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA"}`,
			mockUserReturnMid: nil,
			mockUserErrMid:    users.ErrUserNotFound,
			mockAddrCreateErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"user associated with token not found"}`,
		},
		{
			name:              "Failure - No Auth Token",
			targetUserIDStr:   testUserID.String(),
			body:              `{"street":"789 New Ave","city":"Newville","state":"NS","postal_code":"N5T 3K1","country":"CA"}`,
			mockUserReturnMid: nil, // Won't be called
			mockUserErrMid:    nil,
			mockAddrCreateErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo, mockAddressRepo, _, _, router := setupUserHandlerTest(t)

			// Mock middleware check (only if token is expected and UserID is valid)
			if tc.targetUserIDStr != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock address repo call (only if middleware/auth/validation succeeds)
			if tc.mockUserErrMid == nil &&
				tc.targetUserIDStr != "not-a-uuid" &&
				tc.expectedBody != `{"error":"authorization header required"}` &&
				tc.expectedStatus != http.StatusForbidden &&
				tc.expectedStatus != http.StatusBadRequest {
				mockAddressRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Address")).
					Return(func(ctx context.Context, adr *models.Address) *models.Address {
						if tc.mockAddrCreateErr != nil {
							return nil
						}
						adr.ID = uuid.New()
						adr.CreatedAt = time.Now()
						adr.UpdatedAt = adr.CreatedAt
						return adr
					}, tc.mockAddrCreateErr).Once()
			}

			reqURL := "/api/users/" + tc.targetUserIDStr + "/addresses"
			req := httptest.NewRequest(http.MethodPost, reqURL, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockAddressRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateAddress(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	anotherUserID := uuid.New()
	testAddressID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)
	userForToken := &models.User{ID: testUserID}

	tests := []struct {
		name              string
		targetUserIDStr   string // User ID in the URL
		targetAddrIDStr   string // Address ID in the URL
		body              string // Request body
		mockUserReturnMid *models.User
		mockUserErrMid    error
		mockAddrUpdateErr error
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Success - Update Own Address",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"999 Updated St","city":"UpdateCity","state":"UP","postal_code":"U9P 0S0","country":"CA","is_default":false}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      "999 Updated St",
		},
		{
			name:              "Failure - Invalid JSON",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street:"}`, // Invalid JSON
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"invalid request body"}`,
		},
		{
			name:              "Failure - Missing Fields",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"city":"Only City"}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil,
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      `{"error":"all address fields are required"}`,
		},
		{
			name:              "Failure - Invalid User ID in URL",
			targetUserIDStr:   "not-a-uuid",
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      "404 page not found",
		},
		{
			name:              "Failure - Invalid Address ID in URL",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   "not-a-uuid",
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      "404 page not found",
		},
		{
			name:              "Failure - Unauthorized (Different User ID)",
			targetUserIDStr:   anotherUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil, // Won't be called
			expectedStatus:    http.StatusForbidden,
			expectedBody:      `{"error":"forbidden"}`,
		},
		{
			name:              "Failure - Address Not Found",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: addresses.ErrAddressNotFound,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      `{"error":"address not found"}`,
		},
		{
			name:              "Failure - Repository Error",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrUpdateErr: errors.New("db update error"),
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to update address"}`,
		},
		{
			name:              "Failure - Middleware User Check Fails",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: nil,
			mockUserErrMid:    users.ErrUserNotFound,
			mockAddrUpdateErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"user associated with token not found"}`,
		},
		{
			name:              "Failure - No Auth Token",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			body:              `{"street":"...","city":"...","state":"...","postal_code":"...","country":"..."}`,
			mockUserReturnMid: nil, // Won't be called
			mockUserErrMid:    nil,
			mockAddrUpdateErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo, mockAddressRepo, _, _, router := setupUserHandlerTest(t)

			// Mock middleware check only if token is expected and BOTH IDs in URL are valid format
			if tc.targetUserIDStr != "not-a-uuid" && tc.targetAddrIDStr != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock address repo call (only if middleware/auth/validation/parsing succeeds)
			parsedTargetID, _ := uuid.Parse(tc.targetUserIDStr)
			parsedAddrID, _ := uuid.Parse(tc.targetAddrIDStr)
			if tc.mockUserErrMid == nil &&
				tc.targetUserIDStr != "not-a-uuid" &&
				tc.targetAddrIDStr != "not-a-uuid" &&
				tc.expectedBody != `{"error":"authorization header required"}` &&
				tc.expectedStatus != http.StatusForbidden &&
				tc.expectedStatus != http.StatusBadRequest {
				mockAddressRepo.On("Update", mock.Anything, parsedTargetID, parsedAddrID, mock.AnythingOfType("*models.Address")).
					Return(func(ctx context.Context, uid, aid uuid.UUID, adr *models.Address) *models.Address {
						if tc.mockAddrUpdateErr != nil {
							return nil
						}
						adr.ID = aid
						adr.UserID = uid
						adr.UpdatedAt = time.Now()
						return adr
					}, tc.mockAddrUpdateErr).Once()
			}

			reqURL := "/api/users/" + tc.targetUserIDStr + "/addresses/" + tc.targetAddrIDStr
			req := httptest.NewRequest(http.MethodPut, reqURL, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockAddressRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_DeleteAddress(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	anotherUserID := uuid.New()
	testAddressID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)
	userForToken := &models.User{ID: testUserID}

	tests := []struct {
		name              string
		targetUserIDStr   string // User ID in the URL
		targetAddrIDStr   string // Address ID in the URL
		mockUserReturnMid *models.User
		mockUserErrMid    error
		mockAddrDeleteErr error
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Success - Delete Own Address",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: nil,
			expectedStatus:    http.StatusNoContent,
			expectedBody:      "",
		},
		{
			name:              "Failure - Invalid User ID in URL",
			targetUserIDStr:   "not-a-uuid",
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: nil,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      "404 page not found",
		},
		{
			name:              "Failure - Invalid Address ID in URL",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   "not-a-uuid",
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: nil,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      "404 page not found",
		},
		{
			name:              "Failure - Unauthorized (Different User ID)",
			targetUserIDStr:   anotherUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: nil, // Won't be called
			expectedStatus:    http.StatusForbidden,
			expectedBody:      `{"error":"forbidden"}`,
		},
		{
			name:              "Failure - Address Not Found",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: addresses.ErrAddressNotFound,
			expectedStatus:    http.StatusNotFound,
			expectedBody:      `{"error":"address not found"}`,
		},
		{
			name:              "Failure - Repository Error",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: userForToken,
			mockUserErrMid:    nil,
			mockAddrDeleteErr: errors.New("db delete error"),
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to delete address"}`,
		},
		{
			name:              "Failure - Middleware User Check Fails",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: nil,
			mockUserErrMid:    users.ErrUserNotFound,
			mockAddrDeleteErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"user associated with token not found"}`,
		},
		{
			name:              "Failure - No Auth Token",
			targetUserIDStr:   testUserID.String(),
			targetAddrIDStr:   testAddressID.String(),
			mockUserReturnMid: nil, // Won't be called
			mockUserErrMid:    nil,
			mockAddrDeleteErr: nil, // Won't be called
			expectedStatus:    http.StatusUnauthorized,
			expectedBody:      `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo, mockAddressRepo, _, _, router := setupUserHandlerTest(t)

			// Mock middleware check only if token is expected and BOTH IDs in URL are valid format
			if tc.targetUserIDStr != "not-a-uuid" && tc.targetAddrIDStr != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock address repo call (only if middleware/auth/parsing succeeds)
			parsedTargetID, _ := uuid.Parse(tc.targetUserIDStr)
			parsedAddrID, _ := uuid.Parse(tc.targetAddrIDStr)
			if tc.mockUserErrMid == nil &&
				tc.targetUserIDStr != "not-a-uuid" &&
				tc.targetAddrIDStr != "not-a-uuid" &&
				tc.expectedBody != `{"error":"authorization header required"}` &&
				tc.expectedStatus != http.StatusForbidden &&
				tc.expectedStatus != http.StatusBadRequest {
				mockAddressRepo.On("Delete", mock.Anything, parsedTargetID, parsedAddrID).Return(tc.mockAddrDeleteErr).Once()
			}

			reqURL := "/api/users/" + tc.targetUserIDStr + "/addresses/" + tc.targetAddrIDStr
			req := httptest.NewRequest(http.MethodDelete, reqURL, nil)
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			if tc.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tc.expectedBody)
			} else {
				assert.Empty(t, rr.Body.String())
			}
			mockUserRepo.AssertExpectations(t)
			mockAddressRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_SetDefaultAddress(t *testing.T) {
	// Test setup vars
	testUserID := uuid.New()
	anotherUserID := uuid.New()
	testAddressID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)
	userForToken := &models.User{ID: testUserID}

	tests := []struct {
		name                  string
		targetUserIDStr       string // User ID in the URL
		targetAddrIDStr       string // Address ID in the URL
		mockUserReturnMid     *models.User
		mockUserErrMid        error
		mockAddrSetDefaultErr error
		expectedStatus        int
		expectedBody          string
	}{
		{
			name:                  "Success - Set Own Address Default",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: nil,
			expectedStatus:        http.StatusOK,
			expectedBody:          "",
		},
		{
			name:                  "Failure - Invalid User ID in URL",
			targetUserIDStr:       "not-a-uuid",
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: nil,
			expectedStatus:        http.StatusNotFound,
			expectedBody:          "404 page not found",
		},
		{
			name:                  "Failure - Invalid Address ID in URL",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       "not-a-uuid",
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: nil,
			expectedStatus:        http.StatusNotFound,
			expectedBody:          "404 page not found",
		},
		{
			name:                  "Failure - Unauthorized (Different User ID)",
			targetUserIDStr:       anotherUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: nil, // Won't be called
			expectedStatus:        http.StatusForbidden,
			expectedBody:          `{"error":"forbidden"}`,
		},
		{
			name:                  "Failure - Address Not Found",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: addresses.ErrAddressNotFound,
			expectedStatus:        http.StatusNotFound,
			expectedBody:          `{"error":"address not found"}`,
		},
		{
			name:                  "Failure - Repository Error",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     userForToken,
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: errors.New("db set default error"),
			expectedStatus:        http.StatusInternalServerError,
			expectedBody:          `{"error":"failed to set default address"}`,
		},
		{
			name:                  "Failure - Middleware User Check Fails",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     nil,
			mockUserErrMid:        users.ErrUserNotFound,
			mockAddrSetDefaultErr: nil, // Won't be called
			expectedStatus:        http.StatusUnauthorized,
			expectedBody:          `{"error":"user associated with token not found"}`,
		},
		{
			name:                  "Failure - No Auth Token",
			targetUserIDStr:       testUserID.String(),
			targetAddrIDStr:       testAddressID.String(),
			mockUserReturnMid:     nil, // Won't be called
			mockUserErrMid:        nil,
			mockAddrSetDefaultErr: nil, // Won't be called
			expectedStatus:        http.StatusUnauthorized,
			expectedBody:          `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo, mockAddressRepo, _, _, router := setupUserHandlerTest(t)

			// Mock middleware check only if token is expected and BOTH IDs in URL are valid format
			if tc.targetUserIDStr != "not-a-uuid" && tc.targetAddrIDStr != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturnMid, tc.mockUserErrMid).Once()
			}

			// Mock address repo call (only if middleware/auth/parsing succeeds)
			parsedTargetID, _ := uuid.Parse(tc.targetUserIDStr)
			parsedAddrID, _ := uuid.Parse(tc.targetAddrIDStr)
			if tc.mockUserErrMid == nil &&
				tc.targetUserIDStr != "not-a-uuid" &&
				tc.targetAddrIDStr != "not-a-uuid" &&
				tc.expectedBody != `{"error":"authorization header required"}` &&
				tc.expectedStatus != http.StatusForbidden &&
				tc.expectedStatus != http.StatusBadRequest {
				mockAddressRepo.On("SetDefault", mock.Anything, parsedTargetID, parsedAddrID).Return(tc.mockAddrSetDefaultErr).Once()
			}

			reqURL := "/api/users/" + tc.targetUserIDStr + "/addresses/" + tc.targetAddrIDStr + "/default"
			req := httptest.NewRequest(http.MethodPost, reqURL, nil) // POST request
			if tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			// Check for specific success message or ignore body for non-error cases if needed
			if tc.expectedStatus >= 400 || (tc.expectedStatus < 300 && tc.expectedBody != "") { // Check body for errors or specific success messages
				assert.Contains(t, rr.Body.String(), tc.expectedBody)
			} else if tc.expectedStatus < 300 && tc.expectedBody == "" { // Check for empty body on success if expected
				assert.Empty(t, rr.Body.String())
			}
			mockUserRepo.AssertExpectations(t)
			mockAddressRepo.AssertExpectations(t)
		})
	}
}

// Tests will go here...
