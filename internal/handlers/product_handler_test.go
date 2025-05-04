package handlers_test

import (
	"bullet-cloud-api/internal/auth" // For middleware and context key
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/products"
	"bullet-cloud-api/internal/users" // For user mock
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

// --- Test Setup Helper ---

// setupProductTest creates mock repositories, handler, middleware, and router for tests.
func setupProductTest(t *testing.T) (*products.MockProductRepository, *users.MockUserRepository, *handlers.ProductHandler, *auth.Middleware, *mux.Router) {
	mockProductRepo := new(products.MockProductRepository)
	mockUserRepo := new(users.MockUserRepository) // Needed for middleware
	productHandler := handlers.NewProductHandler(mockProductRepo)

	// Use a fixed test secret for predictable tokens
	testJwtSecret := "test-secret-for-jwt-please-change"
	authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)

	router := mux.NewRouter()
	apiV1 := router.PathPrefix("/api").Subrouter()

	// Public routes
	apiV1.HandleFunc("/products", productHandler.GetAllProducts).Methods("GET")
	apiV1.HandleFunc("/products/{id:[0-9a-fA-F-]+}", productHandler.GetProduct).Methods("GET")

	// Protected routes
	protectedProductRoutes := apiV1.PathPrefix("/products").Subrouter()
	protectedProductRoutes.Use(authMiddleware.Authenticate) // Apply middleware
	protectedProductRoutes.HandleFunc("", productHandler.CreateProduct).Methods("POST")
	protectedProductRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", productHandler.UpdateProduct).Methods("PUT")
	protectedProductRoutes.HandleFunc("/{id:[0-9a-fA-F-]+}", productHandler.DeleteProduct).Methods("DELETE")

	return mockProductRepo, mockUserRepo, productHandler, authMiddleware, router
}

// --- Tests for Public Routes ---

func TestProductHandler_GetAllProducts(t *testing.T) {
	mockRepo, _, _, _, router := setupProductTest(t)

	tests := []struct {
		name           string
		mockReturn     []models.Product
		mockError      error
		expectedStatus int
		expectedBody   string // Expect JSON string or partial match
	}{
		{
			name: "Success - Multiple Products",
			mockReturn: []models.Product{
				{ID: uuid.New(), Name: "Product A", Price: 10.0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: uuid.New(), Name: "Product B", Price: 25.50, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "Product A", // Just check if product names are present
		},
		{
			name:           "Success - No Products",
			mockReturn:     []models.Product{}, // Empty slice
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name:           "Failure - Repository Error",
			mockReturn:     nil,
			mockError:      errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"failed to retrieve products"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo.On("FindAll", mock.Anything).Return(tc.mockReturn, tc.mockError).Once()

			req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestProductHandler_GetProduct(t *testing.T) {
	mockRepo, _, _, _, router := setupProductTest(t)
	testID := uuid.New()
	foundProduct := models.Product{ID: testID, Name: "Specific Product", Price: 19.99}

	tests := []struct {
		name           string
		productID      string
		mockReturn     *models.Product
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success - Product Found",
			productID:      testID.String(),
			mockReturn:     &foundProduct,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "Specific Product",
		},
		{
			name:           "Failure - Product Not Found",
			productID:      uuid.New().String(), // Different ID
			mockReturn:     nil,
			mockError:      products.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"product not found"}`,
		},
		{
			name:           "Failure - Invalid UUID",
			productID:      "not-a-uuid",
			mockReturn:     nil, // Mock won't be called
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid product ID format"}`,
		},
		{
			name:           "Failure - Repository Error",
			productID:      testID.String(),
			mockReturn:     nil,
			mockError:      errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"failed to retrieve product"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectation only if the UUID is valid and repo expected to be called
			if tc.productID != "not-a-uuid" {
				parsedID, _ := uuid.Parse(tc.productID)
				mockRepo.On("FindByID", mock.Anything, parsedID).Return(tc.mockReturn, tc.mockError).Maybe()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/products/"+tc.productID, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockRepo.AssertExpectations(t) // Maybe() allows call not to happen for invalid UUID test
		})
	}
}

// --- Tests for Protected Routes (Require Authentication) ---

// Helper to generate a test token
func generateTestToken(userID uuid.UUID, secret string) string {
	token, err := auth.GenerateToken(userID, secret, time.Hour)
	if err != nil {
		panic("failed to generate test token: " + err.Error()) // Panic in test setup is ok
	}
	return token
}

func TestProductHandler_CreateProduct(t *testing.T) {
	mockProductRepo, mockUserRepo, _, _, router := setupProductTest(t)
	testUserID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)

	tests := []struct {
		name           string
		body           string
		mockUserReturn *models.User // For middleware check
		mockUserErr    error
		mockCreateErr  error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			body:           `{"name":"New Gadget","description":"Shiny","price":99.90}`,
			mockUserReturn: &models.User{ID: testUserID}, // Simulate user exists
			mockUserErr:    nil,
			mockCreateErr:  nil,
			expectedStatus: http.StatusCreated,
			expectedBody:   "New Gadget",
		},
		{
			name:           "Failure - Invalid JSON",
			body:           `{"name":"Gadget",}`, // Invalid JSON
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockCreateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid request body"}`,
		},
		{
			name:           "Failure - Missing Name",
			body:           `{"description":"Shiny","price":99.90}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockCreateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"product name is required and price must be non-negative"}`,
		},
		{
			name:           "Failure - Negative Price",
			body:           `{"name":"New Gadget","description":"Shiny","price":-10.0}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockCreateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"product name is required and price must be non-negative"}`,
		},
		{
			name:           "Failure - Repo Create Error",
			body:           `{"name":"New Gadget","description":"Shiny","price":99.90}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockCreateErr:  errors.New("db create failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"failed to create product"}`,
		},
		{
			name:           "Failure - Middleware User Check Fails",
			body:           `{"name":"New Gadget","description":"Shiny","price":99.90}`,
			mockUserReturn: nil, // Simulate user not found by middleware
			mockUserErr:    users.ErrUserNotFound,
			mockCreateErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"user associated with token not found"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Mock middleware user check
			mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturn, tc.mockUserErr).Maybe()

			// Mock product repo create (only if middleware check is expected to pass)
			if tc.mockUserErr == nil {
				mockProductRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Product")).
					Return(func(ctx context.Context, p *models.Product) *models.Product {
						// Simulate DB assigning ID and timestamps
						p.ID = uuid.New()
						p.CreatedAt = time.Now()
						p.UpdatedAt = p.CreatedAt
						return p
					}, tc.mockCreateErr).Maybe()
			}

			req := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+testToken)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockProductRepo.AssertExpectations(t)
		})
	}
}

// TODO: Add tests for UpdateProduct
// TODO: Add tests for DeleteProduct
