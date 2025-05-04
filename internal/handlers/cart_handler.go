package handlers

import (
	// For UserIDContextKey
	"bullet-cloud-api/internal/cart" // Cart Repository
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/products" // Product Repository
	"bullet-cloud-api/internal/webutils" // JSON Helpers
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// CartHandler handles cart-related requests.
type CartHandler struct {
	CartRepo    cart.CartRepository
	ProductRepo products.ProductRepository // Needed to get current price on add
}

// NewCartHandler creates a new CartHandler.
func NewCartHandler(cartRepo cart.CartRepository, productRepo products.ProductRepository) *CartHandler {
	return &CartHandler{
		CartRepo:    cartRepo,
		ProductRepo: productRepo,
	}
}

// --- Request/Response Structs ---

type AddCartItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// CartResponse includes the cart and its items.
type CartResponse struct {
	Cart  models.Cart       `json:"cart"`
	Items []models.CartItem `json:"items"`
	Total float64           `json:"total"` // Calculated total price
}

// --- Handlers ---

// getOrCreateUserCart is a helper to get the cart for the authenticated user.
func (h *CartHandler) getOrCreateUserCart(w http.ResponseWriter, r *http.Request) (*models.Cart, bool) {
	authUserID, err := getAuthenticatedUserID(r) // Use helper from user_handler
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return nil, false
	}

	userCart, err := h.CartRepo.GetOrCreateCartByUserID(r.Context(), authUserID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to get or create cart"), http.StatusInternalServerError)
		return nil, false
	}
	return userCart, true
}

// GetCart handles GET /api/cart
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userCart, ok := h.getOrCreateUserCart(w, r)
	if !ok {
		return
	}

	items, err := h.CartRepo.GetCartItems(r.Context(), userCart.ID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve cart items"), http.StatusInternalServerError)
		return
	}

	// Calculate total
	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}

	resp := CartResponse{
		Cart:  *userCart,
		Items: items,
		Total: total,
	}

	webutils.WriteJSON(w, http.StatusOK, resp)
}

// AddItem handles POST /api/cart/items
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userCart, ok := h.getOrCreateUserCart(w, r)
	if !ok {
		return
	}

	var req AddCartItemRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		webutils.ErrorJSON(w, errors.New("quantity must be positive"), http.StatusBadRequest)
		return
	}

	// Validate product exists and get its current price
	product, err := h.ProductRepo.FindByID(r.Context(), req.ProductID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			webutils.ErrorJSON(w, errors.New("product not found"), http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to validate product"), http.StatusInternalServerError)
		}
		return
	}

	// Add or update the item in the repository
	_, err = h.CartRepo.AddItem(r.Context(), userCart.ID, req.ProductID, req.Quantity, product.Price)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to add item to cart"), http.StatusInternalServerError)
		return
	}

	// Return the updated cart content
	h.GetCart(w, r)
}

// UpdateItem handles PUT /api/cart/items/{productId}
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userCart, ok := h.getOrCreateUserCart(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	productIDStr := vars["productId"]
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid product ID format"), http.StatusBadRequest)
		return
	}

	var req UpdateCartItemRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		// Treat quantity 0 or less as a removal request
		h.DeleteItem(w, r) // Reuse DeleteItem handler logic
		return
	}

	_, err = h.CartRepo.UpdateItemQuantity(r.Context(), userCart.ID, productID, req.Quantity)
	if err != nil {
		if errors.Is(err, cart.ErrProductNotInCart) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to update cart item"), http.StatusInternalServerError)
		}
		return
	}

	// Return the updated cart content
	h.GetCart(w, r)
}

// DeleteItem handles DELETE /api/cart/items/{productId}
func (h *CartHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	userCart, ok := h.getOrCreateUserCart(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	productIDStr := vars["productId"]
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid product ID format"), http.StatusBadRequest)
		return
	}

	err = h.CartRepo.RemoveItem(r.Context(), userCart.ID, productID)
	if err != nil {
		if errors.Is(err, cart.ErrProductNotInCart) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to remove cart item"), http.StatusInternalServerError)
		}
		return
	}

	// Return the updated cart content
	h.GetCart(w, r)
}

// ClearCart handles DELETE /api/cart
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userCart, ok := h.getOrCreateUserCart(w, r)
	if !ok {
		return
	}

	err := h.CartRepo.ClearCart(r.Context(), userCart.ID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to clear cart"), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}
