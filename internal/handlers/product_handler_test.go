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

// setupProductTest creates mock repositories, handler, middleware, and router for tests.
func setupProductTest(t *testing.T) (*products.MockProductRepository, *users.MockUserRepository, *handlers.ProductHandler, *auth.Middleware, *mux.Router) {
	mockProductRepo := new(products.MockProductRepository)
	// Call the base setup
	mockUserRepo, authMiddleware, router := setupBaseTest(t)

	productHandler := handlers.NewProductHandler(mockProductRepo)

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
	// Keep setup outside for GetAll as it's simpler
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
		tc := tc // Capture range variable
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
	// Removed setup from here
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
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found",
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
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Moved setup inside t.Run for isolation
			mockRepo, _, _, _, router := setupProductTest(t)

			// Setup mock expectation only if the UUID is valid
			if tc.productID != "not-a-uuid" {
				parsedID, _ := uuid.Parse(tc.productID)
				mockRepo.On("FindByID", mock.Anything, parsedID).Return(tc.mockReturn, tc.mockError).Once()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/products/"+tc.productID, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockRepo.AssertExpectations(t)
		})
	}
}

// --- Tests for Protected Routes (Require Authentication) ---

func TestProductHandler_CreateProduct(t *testing.T) {
	// Removed setup from here
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
		{
			name:           "Failure - No Auth Token",
			body:           `{"name":"No Token Product","price":50.0}`,
			mockUserReturn: nil,
			mockUserErr:    nil,
			mockCreateErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Moved setup inside t.Run for isolation
			mockProductRepo, mockUserRepo, _, _, router := setupProductTest(t)

			// Mock middleware user check
			if tc.expectedStatus != http.StatusUnauthorized || tc.expectedBody == `{"error":"user associated with token not found"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturn, tc.mockUserErr).Once()
			}

			// Mock product repo create (only if middleware check is expected to pass and validation is ok)
			if tc.mockUserErr == nil && tc.expectedStatus != http.StatusBadRequest && tc.expectedStatus != http.StatusUnauthorized {
				mockProductRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Product")).
					Return(func(ctx context.Context, p *models.Product) *models.Product {
						if tc.mockCreateErr != nil {
							return nil
						}
						p.ID = uuid.New()
						p.CreatedAt = time.Now()
						p.UpdatedAt = p.CreatedAt
						return p
					}, tc.mockCreateErr).Once()
			}

			req := httptest.NewRequest(http.MethodPost, "/api/products", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.expectedStatus != http.StatusUnauthorized || tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockProductRepo.AssertExpectations(t)
		})
	}
}

func TestProductHandler_UpdateProduct(t *testing.T) {
	// Setup inside loop
	testUserID := uuid.New()
	productToUpdateID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)

	tests := []struct {
		name           string
		productID      string
		body           string
		mockUserReturn *models.User
		mockUserErr    error
		mockUpdateErr  error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Updated Gadget","description":"Better","price":129.99}`, // Include all fields
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "Updated Gadget",
		},
		{
			name:           "Failure - Invalid UUID",
			productID:      "not-a-uuid",
			body:           `{"name":"Update Attempt"}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusNotFound, // Expect 404 from router
			expectedBody:   "404 page not found",
		},
		{
			name:           "Failure - Invalid JSON",
			productID:      productToUpdateID.String(),
			body:           `{"name":}`, // Invalid JSON
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid request body"}`,
		},
		{
			name:           "Failure - Missing Name",
			productID:      productToUpdateID.String(),
			body:           `{"price":50.0}`, // Missing name
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"product name is required and price must be non-negative"}`,
		},
		{
			name:           "Failure - Negative Price",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Bad Price Product","price":-1.0}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"product name is required and price must be non-negative"}`,
		},
		{
			name:           "Failure - Product Not Found",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Update Attempt","price":10.0}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  products.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"product not found"}`,
		},
		{
			name:           "Failure - Repo Update Error",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Update Attempt","price":10.0}`,
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockUpdateErr:  errors.New("db update failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"failed to update product"}`,
		},
		{
			name:           "Failure - Middleware User Check Fails",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Update Attempt","price":10.0}`,
			mockUserReturn: nil,
			mockUserErr:    users.ErrUserNotFound,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"user associated with token not found"}`,
		},
		{
			name:           "Failure - No Auth Token",
			productID:      productToUpdateID.String(),
			body:           `{"name":"Update Attempt","price":10.0}`,
			mockUserReturn: nil,
			mockUserErr:    nil,
			mockUpdateErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Moved setup inside t.Run for isolation
			mockProductRepo, mockUserRepo, _, _, router := setupProductTest(t)

			// Mock middleware user check only if the UUID is valid AND we are not testing the "No Auth Token" case directly
			if tc.productID != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturn, tc.mockUserErr).Once()
			}

			// Mock product repo update (only if middleware/validation/parsing passes)
			if tc.productID != "not-a-uuid" && tc.mockUserErr == nil && tc.expectedStatus != http.StatusBadRequest && tc.expectedStatus != http.StatusUnauthorized {
				parsedID, _ := uuid.Parse(tc.productID)
				mockProductRepo.On("Update", mock.Anything, parsedID, mock.AnythingOfType("*models.Product")).
					Return(func(ctx context.Context, id uuid.UUID, p *models.Product) *models.Product {
						if tc.mockUpdateErr != nil {
							return nil
						}
						p.ID = id
						p.UpdatedAt = time.Now() // Simulate update
						return p
					}, tc.mockUpdateErr).Once()
			}

			req := httptest.NewRequest(http.MethodPut, "/api/products/"+tc.productID, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.expectedStatus != http.StatusUnauthorized || tc.expectedBody != `{"error":"authorization header required"}` {
				req.Header.Set("Authorization", "Bearer "+testToken)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
			mockUserRepo.AssertExpectations(t)
			mockProductRepo.AssertExpectations(t)
		})
	}
}

func TestProductHandler_DeleteProduct(t *testing.T) {
	// Setup inside loop
	testUserID := uuid.New()
	productToDeleteID := uuid.New()
	testJwtSecret := "test-secret-for-jwt-please-change"
	testToken := generateTestToken(testUserID, testJwtSecret)

	tests := []struct {
		name           string
		productID      string
		mockUserReturn *models.User
		mockUserErr    error
		mockDeleteErr  error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			productID:      productToDeleteID.String(),
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockDeleteErr:  nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "", // No body on success
		},
		{
			name:           "Failure - Invalid UUID",
			productID:      "not-a-uuid",
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockDeleteErr:  nil,
			expectedStatus: http.StatusNotFound, // Expect 404 from router
			expectedBody:   "404 page not found",
		},
		{
			name:           "Failure - Product Not Found",
			productID:      productToDeleteID.String(),
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockDeleteErr:  products.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"product not found"}`,
		},
		{
			name:           "Failure - Repo Delete Error",
			productID:      productToDeleteID.String(),
			mockUserReturn: &models.User{ID: testUserID},
			mockUserErr:    nil,
			mockDeleteErr:  errors.New("db delete failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"failed to delete product"}`,
		},
		{
			name:           "Failure - Middleware User Check Fails",
			productID:      productToDeleteID.String(),
			mockUserReturn: nil,
			mockUserErr:    users.ErrUserNotFound,
			mockDeleteErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"user associated with token not found"}`,
		},
		{
			name:           "Failure - No Auth Token",
			productID:      productToDeleteID.String(),
			mockUserReturn: nil,
			mockUserErr:    nil,
			mockDeleteErr:  nil,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Moved setup inside t.Run for isolation
			mockProductRepo, mockUserRepo, _, _, router := setupProductTest(t)

			// Mock middleware user check only if the UUID is valid AND we are not testing the "No Auth Token" case directly
			if tc.productID != "not-a-uuid" && tc.expectedBody != `{"error":"authorization header required"}` {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(tc.mockUserReturn, tc.mockUserErr).Once()
			}

			// Mock product repo delete (only if middleware/parsing passes)
			if tc.productID != "not-a-uuid" && tc.mockUserErr == nil && tc.expectedStatus != http.StatusBadRequest && tc.expectedStatus != http.StatusUnauthorized {
				parsedID, _ := uuid.Parse(tc.productID)
				mockProductRepo.On("Delete", mock.Anything, parsedID).Return(tc.mockDeleteErr).Once()
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/products/"+tc.productID, nil)
			if tc.expectedStatus != http.StatusUnauthorized || tc.expectedBody != `{"error":"authorization header required"}` {
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
			mockProductRepo.AssertExpectations(t)
		})
	}
}
