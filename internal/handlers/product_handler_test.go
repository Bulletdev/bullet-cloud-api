package handlers_test

import (
	"bullet-cloud-api/internal/auth" // For middleware and context key
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/products"
	"bullet-cloud-api/internal/users" // For user mock
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupProductTest creates mock repositories, handler, middleware, and router for tests.
func setupProductTest(t *testing.T) (*products.MockProductRepository, *MockUserRepository, *handlers.ProductHandler, *auth.Middleware, *mux.Router) {
	mockProductRepo := new(products.MockProductRepository)
	// Call the base setup - Capture necessary mocks and router, ignore others
	_, _, router, mockUserRepo, _, _, _, _, _ := setupBaseTest(t)

	productHandler := handlers.NewProductHandler(mockProductRepo)

	// Need authMiddleware instance for protected routes
	authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)

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
	// Corrected setupBaseTest call
	_, _, router, _, baseMockProductRepo, _, _, _, _ := setupBaseTest(t)
	productHandler := handlers.NewProductHandler(baseMockProductRepo)

	router.HandleFunc("/api/products", productHandler.GetAllProducts).Methods("GET")

	testProducts := []models.Product{
		{ID: uuid.New(), Name: "Product A", Price: 10.99},
		{ID: uuid.New(), Name: "Product B", Price: 25.50},
	}

	tests := []struct {
		name              string
		mockFindAllReturn []models.Product
		mockFindAllError  error
		expectedStatus    int
		expectedBody      string // Expect JSON list
	}{
		{
			name:              "Success",
			mockFindAllReturn: testProducts,
			mockFindAllError:  nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      fmt.Sprintf(`[{"id":"%s","name":"%s","description":"","price":%.2f,"category_id":null,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},{"id":"%s","name":"%s","description":"","price":%.2f,"category_id":null,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}]`, testProducts[0].ID, testProducts[0].Name, testProducts[0].Price, testProducts[1].ID, testProducts[1].Name, testProducts[1].Price),
		},
		{
			name:              "Success - No Products",
			mockFindAllReturn: []models.Product{},
			mockFindAllError:  nil,
			expectedStatus:    http.StatusOK,
			expectedBody:      `[]`,
		},
		{
			name:              "Error - Repository Failure",
			mockFindAllReturn: nil,
			mockFindAllError:  assert.AnError,
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      `{"error":"failed to retrieve products"}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockProductRepo := new(MockProductRepository)
			productHandler.ProductRepo = mockProductRepo

			mockProductRepo.On("FindAll", mock.Anything).Return(tc.mockFindAllReturn, tc.mockFindAllError).Once()

			req, _ := http.NewRequest("GET", "/api/products", nil)
			executeRequestAndAssert(t, router, req, tc.expectedStatus, tc.expectedBody)
			mockProductRepo.AssertExpectations(t)
		})
	}
}

func TestProductHandler_GetProduct(t *testing.T) {
	// Corrected setupBaseTest call
	_, _, router, _, baseMockProductRepo, _, _, _, _ := setupBaseTest(t)
	productHandler := handlers.NewProductHandler(baseMockProductRepo)

	router.HandleFunc("/api/products/{id}", productHandler.GetProduct).Methods("GET")

	testID := uuid.New()
	testProduct := &models.Product{ID: testID, Name: "Found Product", Price: 50.0}

	tests := []struct {
		name               string
		productID          string
		mockFindByIDReturn *models.Product
		mockFindByIDError  error
		expectedStatus     int
		expectedBody       string
	}{
		{
			name:               "Success",
			productID:          testID.String(),
			mockFindByIDReturn: testProduct,
			mockFindByIDError:  nil,
			expectedStatus:     http.StatusOK,
			expectedBody:       fmt.Sprintf(`{"id":"%s","name":"%s","description":"","price":%.2f,"category_id":null,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, testID, testProduct.Name, testProduct.Price),
		},
		{
			name:               "Not Found",
			productID:          uuid.New().String(),
			mockFindByIDReturn: nil,
			mockFindByIDError:  products.ErrProductNotFound,
			expectedStatus:     http.StatusNotFound,
			expectedBody:       `{"error":"product not found"}`,
		},
		{
			name:               "Invalid ID Format",
			productID:          "not-a-valid-uuid",
			mockFindByIDReturn: nil, // Not called
			mockFindByIDError:  nil, // Not called
			expectedStatus:     http.StatusBadRequest,
			expectedBody:       `{"error":"invalid product ID format"}`,
		},
		{
			name:               "Repository Error",
			productID:          testID.String(),
			mockFindByIDReturn: nil,
			mockFindByIDError:  assert.AnError,
			expectedStatus:     http.StatusInternalServerError,
			expectedBody:       `{"error":"failed to retrieve product"}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockProductRepo := new(MockProductRepository)
			productHandler.ProductRepo = mockProductRepo

			if tc.productID != "not-a-valid-uuid" {
				uuidToFind, _ := uuid.Parse(tc.productID)
				mockProductRepo.On("FindByID", mock.Anything, uuidToFind).Return(tc.mockFindByIDReturn, tc.mockFindByIDError).Once()
			}

			url := fmt.Sprintf("/api/products/%s", tc.productID)
			req, _ := http.NewRequest("GET", url, nil)
			executeRequestAndAssert(t, router, req, tc.expectedStatus, tc.expectedBody)
			mockProductRepo.AssertExpectations(t)
		})
	}
}

// --- Tests for Protected Routes (Require Authentication) ---

func TestProductHandler_CreateProduct(t *testing.T) {
	// Corrected setupBaseTest call
	_, _, router, baseMockUserRepo, baseMockProductRepo, _, _, _, token := setupBaseTest(t)
	productHandler := handlers.NewProductHandler(baseMockProductRepo)
	authMiddleware := auth.NewMiddleware(testJwtSecret, baseMockUserRepo)

	// Extract UserID from token for mock setup
	claims, err := auth.ValidateToken(token, testJwtSecret)
	require.NoError(t, err)
	testUserID := claims.UserID

	router.Handle("/api/products", authMiddleware.Authenticate(http.HandlerFunc(productHandler.CreateProduct))).Methods("POST")

	productName := "New Gadget"
	productPrice := 199.99
	productDesc := "A cool new gadget"
	testProduct := models.Product{Name: productName, Price: productPrice, Description: productDesc}
	createdProduct := models.Product{ID: uuid.New(), Name: productName, Price: productPrice, Description: productDesc}

	tests := []struct {
		name             string
		body             string // JSON request body
		mockCreateReturn *models.Product
		mockCreateError  error
		expectedStatus   int
		expectedBody     string // Expected JSON body
	}{
		{
			name:             "Success",
			body:             fmt.Sprintf(`{"name":"%s", "price":%.2f, "description":"%s"}`, productName, productPrice, productDesc),
			mockCreateReturn: &createdProduct,
			mockCreateError:  nil,
			expectedStatus:   http.StatusCreated,
			expectedBody:     fmt.Sprintf(`{"id":"%s","name":"%s","description":"%s","price":%.2f,"category_id":null,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, createdProduct.ID, createdProduct.Name, createdProduct.Description, createdProduct.Price),
		},
		{
			name:             "Invalid JSON",
			body:             `{"name":"Test",}`, // Malformed
			mockCreateReturn: nil,
			mockCreateError:  nil, // Not called
			expectedStatus:   http.StatusBadRequest,
			expectedBody:     `{"error":"invalid request body"}`,
		},
		{
			name:             "Missing Name",
			body:             fmt.Sprintf(`{"price":%.2f}`, productPrice),
			mockCreateReturn: nil,
			mockCreateError:  nil, // Not called
			expectedStatus:   http.StatusBadRequest,
			expectedBody:     `{"error":"product name is required"}`,
		},
		{
			name: "Missing Price",
			body: `{"name":"New Gadget"}`,
			// Ensure return and error are nil as mock shouldn't be called
			mockCreateReturn: nil,
			mockCreateError:  nil,
			expectedStatus:   http.StatusBadRequest,
			expectedBody:     `{"error":"product price must be positive"}`,
		},
		{
			name:             "Zero Price",
			body:             fmt.Sprintf(`{"name":"%s", "price":0}`, productName),
			mockCreateReturn: nil,
			mockCreateError:  nil,
			expectedStatus:   http.StatusBadRequest,
			expectedBody:     `{"error":"product price must be positive"}`,
		},
		{
			name:             "Repository Error",
			body:             fmt.Sprintf(`{"name":"%s", "price":%.2f}`, productName, productPrice),
			mockCreateReturn: nil,
			mockCreateError:  assert.AnError,
			expectedStatus:   http.StatusInternalServerError,
			expectedBody:     `{"error":"failed to create product"}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockProductRepo := new(MockProductRepository)
			mockUserRepo := new(MockUserRepository)
			productHandler.ProductRepo = mockProductRepo
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe()

			// Setup mock expectation for Create
			if tc.expectedStatus == http.StatusCreated {
				// For success case, match the product details
				mockProductRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
					return p.Name == testProduct.Name && p.Price == testProduct.Price && p.Description == testProduct.Description
				})).Return(tc.mockCreateReturn, tc.mockCreateError).Once()
			} else if tc.expectedStatus == http.StatusInternalServerError && tc.mockCreateError != nil {
				// For repository error case, just match the type for simplicity
				mockProductRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Product")).Return(tc.mockCreateReturn, tc.mockCreateError).Once()
				// Note: We expect the repo method to be called in this specific error case
			}

			req, _ := http.NewRequest("POST", "/api/products", bytes.NewBufferString(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			subRouter := mux.NewRouter()
			subRouter.Handle("/api/products", authMiddleware.Authenticate(http.HandlerFunc(productHandler.CreateProduct))).Methods("POST")

			executeRequestAndAssert(t, subRouter, req, tc.expectedStatus, tc.expectedBody)

			mockProductRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestProductHandler_UpdateProduct(t *testing.T) {
	// Setup inside loop
	testUserID := uuid.New()
	productToUpdateID := uuid.New()
	// Corrected generateTestToken call
	testToken, err := generateTestToken(testUserID)
	require.NoError(t, err, "Failed to generate test token")

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
	// Corrected generateTestToken call
	testToken, err := generateTestToken(testUserID)
	require.NoError(t, err, "Failed to generate test token")

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
