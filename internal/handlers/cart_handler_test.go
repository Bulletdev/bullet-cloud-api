package handlers_test

import (
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/cart"
	"bullet-cloud-api/internal/handlers"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/users"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupCartTest creates mocks, handler, middleware, and router for cart tests.
func setupCartTest(t *testing.T) (*MockCartRepository, *MockUserRepository, *MockProductRepository, *handlers.CartHandler, *auth.Middleware, *mux.Router) {
	mockCartRepo := new(MockCartRepository)
	// Call the base setup - Capture necessary mocks and router, ignore cart repo from base
	_, _, router, mockUserRepo, mockProductRepo, _, _, _, _ := setupBaseTest(t)

	cartHandler := handlers.NewCartHandler(mockCartRepo, mockProductRepo)

	// Need authMiddleware instance for protected routes
	authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)

	apiV1 := router.PathPrefix("/api").Subrouter()
	apiV1.Use(authMiddleware.Authenticate)
	apiV1.HandleFunc("/cart", cartHandler.GetCart).Methods("GET")
	apiV1.HandleFunc("/cart/items", cartHandler.AddItem).Methods("POST")
	apiV1.HandleFunc("/cart/items/{productId:[0-9a-fA-F-]+}", cartHandler.UpdateItem).Methods("PUT")
	apiV1.HandleFunc("/cart/items/{productId:[0-9a-fA-F-]+}", cartHandler.DeleteItem).Methods("DELETE")
	apiV1.HandleFunc("/cart", cartHandler.ClearCart).Methods("DELETE") // Note: DELETE on /api/cart for clearing

	return mockCartRepo, mockUserRepo, mockProductRepo, cartHandler, authMiddleware, router
}

// TestCartHandler_GetCart tests the GET /api/cart endpoint
func TestCartHandler_GetCart(t *testing.T) {
	testUserID := uuid.New()
	testCart := &models.Cart{ID: uuid.New(), UserID: testUserID}
	testItems := []models.CartItem{
		{CartID: testCart.ID, ProductID: uuid.New(), Quantity: 2, Price: 10.50},
		{CartID: testCart.ID, ProductID: uuid.New(), Quantity: 1, Price: 25.00},
	}

	tests := []struct {
		name                 string
		mocksSetup           func(mockCartRepo *MockCartRepository) // Simplified mock setup
		expectedStatus       int
		expectedBodyContains string
	}{
		{
			name: "Success - Existing Cart with Items",
			mocksSetup: func(mockCartRepo *MockCartRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockGetCartItemsSuccess(mockCartRepo, testCart.ID, testItems)
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart":{"id":"%s","user_id":"%s","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"items":[{"id":"00000000-0000-0000-0000-000000000000","cart_id":"%s","product_id":"%s","quantity":%d,"price":%.2f,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},{"id":"00000000-0000-0000-0000-000000000000","cart_id":"%s","product_id":"%s","quantity":%d,"price":%.2f,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}],"total":%.2f}`, testCart.ID, testUserID, testItems[0].CartID, testItems[0].ProductID, testItems[0].Quantity, testItems[0].Price, testItems[1].CartID, testItems[1].ProductID, testItems[1].Quantity, testItems[1].Price, (testItems[0].Price*float64(testItems[0].Quantity))+(testItems[1].Price*float64(testItems[1].Quantity))),
		},
		{
			name: "Success - New Cart (Empty)",
			mocksSetup: func(mockCartRepo *MockCartRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockGetCartItemsSuccess(mockCartRepo, testCart.ID, []models.CartItem{})
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart": {"id":"%s", "user_id":"%s", "created_at":"0001-01-01T00:00:00Z", "updated_at":"0001-01-01T00:00:00Z"}, "items": [], "total": 0.00}`, testCart.ID, testUserID),
		},
		{
			name: "Error - GetOrCreateCart Fails",
			mocksSetup: func(mockCartRepo *MockCartRepository) {
				mockGetOrCreateCartError(mockCartRepo, testUserID)
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to get or create cart"}`,
		},
		{
			name: "Error - GetCartItems Fails",
			mocksSetup: func(mockCartRepo *MockCartRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockGetCartItemsError(mockCartRepo, testCart.ID)
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to retrieve cart items"}`,
		},
		// Add test case for middleware failure
		{
			name:                 "Error - Middleware User Check Fails",
			mocksSetup:           func(mockCartRepo *MockCartRepository) { /* Cart repo not called */ },
			expectedStatus:       http.StatusUnauthorized,
			expectedBodyContains: `{"error":"user associated with token not found"}`,
		},
		// Add test case for missing token
		{
			name:                 "Error - No Auth Token",
			mocksSetup:           func(mockCartRepo *MockCartRepository) { /* Cart repo not called */ },
			expectedStatus:       http.StatusUnauthorized,
			expectedBodyContains: `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run for isolation
			mockUserRepo := new(MockUserRepository)
			mockCartRepo := new(MockCartRepository)
			mockProductRepo := new(MockProductRepository) // Needed for handler instantiation
			cartHandler := handlers.NewCartHandler(mockCartRepo, mockProductRepo)
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			router := mux.NewRouter()
			router.Handle("/api/cart", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.GetCart))).Methods("GET")

			// Conditionally setup user mock for middleware
			if tc.name == "Error - Middleware User Check Fails" {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(nil, users.ErrUserNotFound).Once()
			} else if tc.name != "Error - No Auth Token" {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe() // Use Maybe for non-error cases
			}

			// Setup CartRepo mocks specific to this test case
			tc.mocksSetup(mockCartRepo)

			// Generate token (or not for the specific test case)
			var currentToken string
			if tc.name != "Error - No Auth Token" {
				var err error
				currentToken, err = generateTestToken(testUserID)
				require.NoError(t, err)
			}

			req, _ := http.NewRequest("GET", "/api/cart", nil)
			if currentToken != "" {
				req.Header.Set("Authorization", "Bearer "+currentToken)
			}

			executeRequestAndAssert(t, router, req, tc.expectedStatus, tc.expectedBodyContains)

			// Assert mocks
			mockCartRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestCartHandler_AddItem tests the POST /api/cart/items endpoint
func TestCartHandler_AddItem(t *testing.T) {
	testUserID := uuid.New()
	testCart := &models.Cart{ID: uuid.New(), UserID: testUserID}
	productID := uuid.New()
	testProduct := &models.Product{ID: productID, Name: "Test Item", Price: 19.99}
	testQuantity := 2
	testCartItem := &models.CartItem{CartID: testCart.ID, ProductID: productID, Quantity: testQuantity, Price: testProduct.Price}

	tests := []struct {
		name                 string
		body                 string
		mocksSetup           func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository)
		expectedStatus       int
		expectedBodyContains string
	}{
		{
			name: "Success - Add New Item",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockFindProductSuccess(mockProductRepo, testProduct)
				mockAddItemSuccess(mockCartRepo, testCart.ID, productID, testQuantity, testProduct.Price, testCartItem)
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: fmt.Sprintf(`{"id":"00000000-0000-0000-0000-000000000000","cart_id":"%s","product_id":"%s","quantity":%d,"price":%.2f,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, testCart.ID, productID, testQuantity, testProduct.Price),
		},
		{
			name: "Error - Invalid Quantity (Zero)",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":0}`, productID),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				// No product or add item mock needed
			},
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"quantity must be positive"}`,
		},
		{
			name: "Error - Invalid Quantity (Negative)",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":-1}`, productID),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
			},
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"quantity must be positive"}`,
		},
		{
			name: "Error - Product Not Found",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockFindProductNotFound(mockProductRepo, productID)
			},
			expectedStatus:       http.StatusNotFound,
			expectedBodyContains: `{"error":"product not found"}`,
		},
		{
			name: "Error - FindProductByID Fails (Internal Error)",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockFindProductError(mockProductRepo, productID)
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to validate product"}`,
		},
		{
			name: "Error - AddItem Fails",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				mockGetOrCreateCartSuccess(mockCartRepo, testUserID, testCart)
				mockFindProductSuccess(mockProductRepo, testProduct)
				mockAddItemError(mockCartRepo, testCart.ID, productID, testQuantity, testProduct.Price)
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to add item to cart"}`,
		},
		{
			name: "Error - Invalid JSON Body",
			body: `{"product_id": invalid}`, // Malformed JSON
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) {
				// May or may not call GetOrCreateCart depending on when body is parsed
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Maybe()
			},
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"invalid request body"}`,
		},
		{
			name: "Error - Middleware User Check Fails",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) { /* No cart/product mocks needed */
			},
			expectedStatus:       http.StatusUnauthorized,
			expectedBodyContains: `{"error":"user associated with token not found"}`,
		},
		{
			name: "Error - No Auth Token",
			body: fmt.Sprintf(`{"product_id":"%s", "quantity":%d}`, productID, testQuantity),
			mocksSetup: func(mockCartRepo *MockCartRepository, mockProductRepo *MockProductRepository) { /* No cart/product mocks needed */
			},
			expectedStatus:       http.StatusUnauthorized,
			expectedBodyContains: `{"error":"authorization header required"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Setup inside t.Run
			mockUserRepo := new(MockUserRepository)
			mockCartRepo := new(MockCartRepository)
			mockProductRepo := new(MockProductRepository)
			cartHandler := handlers.NewCartHandler(mockCartRepo, mockProductRepo)
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			router := mux.NewRouter()
			router.Handle("/api/cart/items", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.AddItem))).Methods("POST")

			// Conditionally setup user mock for middleware
			if tc.name == "Error - Middleware User Check Fails" {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(nil, users.ErrUserNotFound).Once()
			} else if tc.name != "Error - No Auth Token" {
				mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe()
			}

			// Setup CartRepo and ProductRepo mocks
			tc.mocksSetup(mockCartRepo, mockProductRepo)

			// Generate token (or not)
			var currentToken string
			if tc.name != "Error - No Auth Token" {
				var err error
				currentToken, err = generateTestToken(testUserID)
				require.NoError(t, err)
			}

			req, _ := http.NewRequest("POST", "/api/cart/items", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if currentToken != "" {
				req.Header.Set("Authorization", "Bearer "+currentToken)
			}

			rr := executeRequestAndAssert(t, router, req, tc.expectedStatus, "")

			// Assert body based on expected status
			if tc.expectedStatus == http.StatusCreated {
				assert.Contains(t, rr.Body.String(), tc.expectedBodyContains) // Check if created item is in response
			} else {
				require.JSONEq(t, tc.expectedBodyContains, rr.Body.String()) // Exact match for errors
			}

			// Assert mocks
			mockCartRepo.AssertExpectations(t)
			mockProductRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestCartHandler_DeleteItem tests the DELETE /api/cart/items/{productId} endpoint
func TestCartHandler_DeleteItem(t *testing.T) {
	// Explicitly capture all 9 return values
	_, _, router, baseMockUserRepo, baseMockProductRepo, _, _, baseMockCartRepo, token := setupBaseTest(t)

	// Handler created once
	cartHandler := handlers.NewCartHandler(baseMockCartRepo, baseMockProductRepo)

	claims, err := auth.ValidateToken(token, testJwtSecret)
	require.NoError(t, err)
	testUserID := claims.UserID

	productID := uuid.New()
	testCart := &models.Cart{ID: uuid.New(), UserID: testUserID}

	// Route registered once
	router.Handle("/api/cart/items/{productId}", auth.NewMiddleware(testJwtSecret, baseMockUserRepo).Authenticate(http.HandlerFunc(cartHandler.DeleteItem))).Methods("DELETE")

	tests := []struct {
		name                 string
		productIDParam       string
		mockGetOrCreateCart  func(*MockCartRepository)
		mockRemoveItem       func(*MockCartRepository)
		mockGetCartItems     func(*MockCartRepository) // For the final GetCart call
		expectedStatus       int
		expectedBodyContains string
	}{
		{
			name:           "Success",
			productIDParam: productID.String(),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				// Expect two calls because handler calls it, then calls GetCart which calls it again
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Twice()
			},
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(nil).Once()
			},
			mockGetCartItems: func(mockCartRepo *MockCartRepository) {
				// Simulate cart being empty after removal
				mockCartRepo.On("GetCartItems", mock.Anything, testCart.ID).Return([]models.CartItem{}, nil).Once()
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart":{"id":"%s","user_id":"%s","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"items":[],"total":0}`, testCart.ID, testUserID),
		},
		{
			name:           "Product Not Found in Cart",
			productIDParam: productID.String(),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(cart.ErrProductNotInCart).Once()
			},
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusNotFound,
			expectedBodyContains: `{"error":"product not found in cart"}`,
		},
		{
			name:           "Invalid Product ID Format",
			productIDParam: "invalid-uuid",
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"invalid product ID format"}`,
		},
		{
			name:           "GetOrCreateCart Fails",
			productIDParam: productID.String(),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(nil, assert.AnError).Once()
			},
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to get or create cart"}`,
		},
		{
			name:           "RemoveItem Fails (Internal Error)",
			productIDParam: productID.String(),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(assert.AnError).Once()
			},
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to remove cart item"}`,
		},
		{
			name:           "GetCartItems Fails After Successful Remove",
			productIDParam: productID.String(),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				// Expect two calls, same as success case for RemoveItem
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Twice()
			},
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(nil).Once()
			},
			mockGetCartItems: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetCartItems", mock.Anything, testCart.ID).Return(nil, assert.AnError).Once()
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to retrieve cart items"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Fresh mocks for subtest
			mockUserRepo := new(MockUserRepository)
			mockCartRepo := new(MockCartRepository)
			cartHandler.CartRepo = mockCartRepo // Update handler repo

			// New middleware
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe()

			// Setup specific mocks
			tc.mockGetOrCreateCart(mockCartRepo)
			tc.mockRemoveItem(mockCartRepo)
			tc.mockGetCartItems(mockCartRepo)

			url := fmt.Sprintf("/api/cart/items/%s", tc.productIDParam)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			// Re-register route with new middleware
			subRouter := mux.NewRouter()
			subRouter.Handle("/api/cart/items/{productId}", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.DeleteItem))).Methods("DELETE")

			executeRequestAndAssert(t, subRouter, req, tc.expectedStatus, tc.expectedBodyContains)

			mockCartRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestCartHandler_UpdateItem tests the PUT /api/cart/items/{productId} endpoint
func TestCartHandler_UpdateItem(t *testing.T) {
	// Explicitly capture all 9 return values
	_, _, router, baseMockUserRepo, baseMockProductRepo, _, _, baseMockCartRepo, token := setupBaseTest(t)

	// Handler created once
	cartHandler := handlers.NewCartHandler(baseMockCartRepo, baseMockProductRepo)

	claims, err := auth.ValidateToken(token, testJwtSecret)
	require.NoError(t, err)
	testUserID := claims.UserID

	productID := uuid.New()
	testCart := &models.Cart{ID: uuid.New(), UserID: testUserID}
	updatedQuantity := 5
	updatedItem := &models.CartItem{CartID: testCart.ID, ProductID: productID, Quantity: updatedQuantity, Price: 15.00}

	// Route registered once
	router.Handle("/api/cart/items/{productId}", auth.NewMiddleware(testJwtSecret, baseMockUserRepo).Authenticate(http.HandlerFunc(cartHandler.UpdateItem))).Methods("PUT")

	tests := []struct {
		name                 string
		productIDParam       string
		body                 string
		mockGetOrCreateCart  func(*MockCartRepository)
		mockUpdateQuantity   func(*MockCartRepository)
		mockRemoveItem       func(*MockCartRepository) // For quantity <= 0 case
		mockGetCartItems     func(*MockCartRepository) // For the final GetCart call
		expectedStatus       int
		expectedBodyContains string
	}{
		{
			name:           "Success",
			productIDParam: productID.String(),
			body:           fmt.Sprintf(`{"quantity": %d}`, updatedQuantity),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				// Expect two calls: one at the start, one inside the final GetCart call
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Twice()
			},
			mockUpdateQuantity: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("UpdateItemQuantity", mock.Anything, testCart.ID, productID, updatedQuantity).Return(updatedItem, nil).Once()
			},
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {}, // Not called
			mockGetCartItems: func(mockCartRepo *MockCartRepository) {
				// Simulate cart having the updated item
				mockCartRepo.On("GetCartItems", mock.Anything, testCart.ID).Return([]models.CartItem{*updatedItem}, nil).Once()
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart":{"id":"%s","user_id":"%s","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"items":[%s],"total":%.2f}`, testCart.ID, testUserID, fmt.Sprintf(`{"id":"00000000-0000-0000-0000-000000000000","cart_id":"%s","product_id":"%s","quantity":%d,"price":%.2f,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`, updatedItem.CartID, updatedItem.ProductID, updatedItem.Quantity, updatedItem.Price), float64(updatedItem.Quantity)*updatedItem.Price),
		},
		{
			name:           "Quantity Zero (Triggers Delete)",
			productIDParam: productID.String(),
			body:           `{"quantity": 0}`,
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				// Expect THREE calls: UpdateItem -> DeleteItem -> GetCart
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Times(3)
			},
			mockUpdateQuantity: func(mockCartRepo *MockCartRepository) {}, // Not called directly
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(nil).Once()
			},
			mockGetCartItems: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetCartItems", mock.Anything, testCart.ID).Return([]models.CartItem{}, nil).Once()
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart":{"id":"%s","user_id":"%s","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"items":[],"total":0}`, testCart.ID, testUserID),
		},
		{
			name:           "Product Not Found in Cart",
			productIDParam: productID.String(),
			body:           fmt.Sprintf(`{"quantity": %d}`, updatedQuantity),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockUpdateQuantity: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("UpdateItemQuantity", mock.Anything, testCart.ID, productID, updatedQuantity).Return(nil, cart.ErrProductNotInCart).Once()
			},
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) {}, // Not called
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) {}, // Not called
			expectedStatus:       http.StatusNotFound,
			expectedBodyContains: `{"error":"product not found in cart"}`,
		},
		{
			name:           "Invalid Quantity (Negative Triggers Delete)",
			productIDParam: productID.String(),
			body:           `{"quantity": -1}`,
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				// Expect THREE calls: UpdateItem -> DeleteItem -> GetCart
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Times(3)
			},
			mockUpdateQuantity: func(mockCartRepo *MockCartRepository) {}, // Not called directly
			mockRemoveItem: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("RemoveItem", mock.Anything, testCart.ID, productID).Return(nil).Once()
			},
			mockGetCartItems: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetCartItems", mock.Anything, testCart.ID).Return([]models.CartItem{}, nil).Once()
			},
			expectedStatus:       http.StatusOK,
			expectedBodyContains: fmt.Sprintf(`{"cart":{"id":"%s","user_id":"%s","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"},"items":[],"total":0}`, testCart.ID, testUserID),
		},
		{
			name:           "Invalid JSON Body",
			productIDParam: productID.String(),
			body:           `{"quantity": "not-a-number"}`,
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Maybe()
			},
			mockUpdateQuantity:   func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"invalid request body"}`,
		},
		{
			name:           "Invalid Product ID Format",
			productIDParam: "invalid-uuid",
			body:           fmt.Sprintf(`{"quantity": %d}`, updatedQuantity),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockUpdateQuantity:   func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: `{"error":"invalid product ID format"}`,
		},
		{
			name:           "GetOrCreateCart Fails",
			productIDParam: productID.String(),
			body:           fmt.Sprintf(`{"quantity": %d}`, updatedQuantity),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(nil, assert.AnError).Once()
			},
			mockUpdateQuantity:   func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) { /* Not called */ },
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to get or create cart"}`,
		},
		{
			name:           "UpdateItemQuantity Fails (Internal Error)",
			productIDParam: productID.String(),
			body:           fmt.Sprintf(`{"quantity": %d}`, updatedQuantity),
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockUpdateQuantity: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("UpdateItemQuantity", mock.Anything, testCart.ID, productID, updatedQuantity).Return(nil, assert.AnError).Once()
			},
			mockRemoveItem:       func(mockCartRepo *MockCartRepository) {}, // Not called
			mockGetCartItems:     func(mockCartRepo *MockCartRepository) {}, // Not called
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to update cart item"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Fresh mocks
			mockUserRepo := new(MockUserRepository)
			mockCartRepo := new(MockCartRepository)
			cartHandler.CartRepo = mockCartRepo // Update handler repo

			// New middleware
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe()

			// Setup specific mocks
			tc.mockGetOrCreateCart(mockCartRepo)
			tc.mockUpdateQuantity(mockCartRepo)
			tc.mockRemoveItem(mockCartRepo)
			tc.mockGetCartItems(mockCartRepo)

			url := fmt.Sprintf("/api/cart/items/%s", tc.productIDParam)
			req, _ := http.NewRequest("PUT", url, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			// Re-register route
			subRouter := mux.NewRouter()
			subRouter.Handle("/api/cart/items/{productId}", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.UpdateItem))).Methods("PUT")

			executeRequestAndAssert(t, subRouter, req, tc.expectedStatus, tc.expectedBodyContains)

			mockCartRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestCartHandler_ClearCart tests the DELETE /api/cart endpoint
func TestCartHandler_ClearCart(t *testing.T) {
	// Explicitly capture all 9 return values
	_, _, router, baseMockUserRepo, baseMockProductRepo, _, _, baseMockCartRepo, token := setupBaseTest(t)

	// Handler created once
	cartHandler := handlers.NewCartHandler(baseMockCartRepo, baseMockProductRepo)

	claims, err := auth.ValidateToken(token, testJwtSecret)
	require.NoError(t, err)
	testUserID := claims.UserID
	testCart := &models.Cart{ID: uuid.New(), UserID: testUserID}

	// Route registered once
	router.Handle("/api/cart", auth.NewMiddleware(testJwtSecret, baseMockUserRepo).Authenticate(http.HandlerFunc(cartHandler.ClearCart))).Methods("DELETE")

	tests := []struct {
		name                 string
		mockGetOrCreateCart  func(*MockCartRepository)
		mockClearCart        func(*MockCartRepository)
		expectedStatus       int
		expectedBodyContains string // Should be empty for 204
	}{
		{
			name: "Success",
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockClearCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("ClearCart", mock.Anything, testCart.ID).Return(nil).Once()
			},
			expectedStatus:       http.StatusNoContent,
			expectedBodyContains: "",
		},
		{
			name: "GetOrCreateCart Fails",
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(nil, assert.AnError).Once()
			},
			mockClearCart:        func(mockCartRepo *MockCartRepository) { /* Not called */ },
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to get or create cart"}`,
		},
		{
			name: "ClearCart Fails",
			mockGetOrCreateCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("GetOrCreateCartByUserID", mock.Anything, testUserID).Return(testCart, nil).Once()
			},
			mockClearCart: func(mockCartRepo *MockCartRepository) {
				mockCartRepo.On("ClearCart", mock.Anything, testCart.ID).Return(assert.AnError).Once()
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: `{"error":"failed to clear cart"}`,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Fresh mocks
			mockUserRepo := new(MockUserRepository)
			mockCartRepo := new(MockCartRepository)
			cartHandler.CartRepo = mockCartRepo // Update handler repo

			// New middleware
			authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
			mockUserRepo.On("FindByID", mock.Anything, testUserID).Return(&models.User{ID: testUserID}, nil).Maybe()

			// Setup specific mocks
			tc.mockGetOrCreateCart(mockCartRepo)
			tc.mockClearCart(mockCartRepo)

			req, _ := http.NewRequest("DELETE", "/api/cart", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			// Re-register route
			subRouter := mux.NewRouter()
			subRouter.Handle("/api/cart", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.ClearCart))).Methods("DELETE")

			executeRequestAndAssert(t, subRouter, req, tc.expectedStatus, tc.expectedBodyContains)

			mockCartRepo.AssertExpectations(t)
			mockUserRepo.AssertExpectations(t)
		})
	}
}
