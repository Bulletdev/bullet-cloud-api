package handlers

import (
	"bullet-cloud-api/internal/addresses"
	"bullet-cloud-api/internal/cart"
	"bullet-cloud-api/internal/models"
	"bullet-cloud-api/internal/orders"
	"bullet-cloud-api/internal/webutils"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// OrderHandler handles order-related requests.
type OrderHandler struct {
	OrderRepo   orders.OrderRepository
	CartRepo    cart.CartRepository
	AddressRepo addresses.AddressRepository // To validate shipping address
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(orderRepo orders.OrderRepository, cartRepo cart.CartRepository, addressRepo addresses.AddressRepository) *OrderHandler {
	return &OrderHandler{
		OrderRepo:   orderRepo,
		CartRepo:    cartRepo,
		AddressRepo: addressRepo,
	}
}

// --- Request/Response Structs ---

type CreateOrderRequest struct {
	ShippingAddressID uuid.UUID `json:"shipping_address_id"`
}

type OrderResponse struct {
	Order models.Order       `json:"order"`
	Items []models.OrderItem `json:"items"`
}

// --- Handlers ---

// CreateOrder handles POST /api/orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	authUserID, err := getAuthenticatedUserID(r) // Use helper from user_handler
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	var req CreateOrderRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// Validate Shipping Address belongs to the user
	_, err = h.AddressRepo.FindByUserAndID(r.Context(), authUserID, req.ShippingAddressID)
	if err != nil {
		if errors.Is(err, addresses.ErrAddressNotFound) {
			webutils.ErrorJSON(w, errors.New("shipping address not found or does not belong to user"), http.StatusBadRequest)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to validate shipping address"), http.StatusInternalServerError)
		}
		return
	}

	// Get user's cart
	userCart, err := h.CartRepo.GetOrCreateCartByUserID(r.Context(), authUserID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve user cart"), http.StatusInternalServerError)
		return
	}

	// Get cart items
	cartItems, err := h.CartRepo.GetCartItems(r.Context(), userCart.ID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve cart items"), http.StatusInternalServerError)
		return
	}

	if len(cartItems) == 0 {
		webutils.ErrorJSON(w, errors.New("cannot create order from empty cart"), http.StatusBadRequest)
		return
	}

	// Create order using the repository (which handles transaction and cart clearing)
	newOrder, err := h.OrderRepo.CreateOrderFromCart(r.Context(), authUserID, userCart.ID, req.ShippingAddressID, cartItems)
	if err != nil {
		log.Printf("ERROR creating order for user %s: %v", authUserID, err)
		webutils.ErrorJSON(w, errors.New("failed to create order"), http.StatusInternalServerError)
		return
	}

	// Fetch the newly created order details including items for the response
	createdOrder, createdItems, err := h.OrderRepo.FindOrderByID(r.Context(), newOrder.ID)
	if err != nil {
		log.Printf("ERROR fetching created order %s details: %v", newOrder.ID, err)
		// Return the basic order info even if fetching items fails for the response
		webutils.WriteJSON(w, http.StatusCreated, newOrder)
		return
	}

	resp := OrderResponse{
		Order: *createdOrder,
		Items: createdItems,
	}
	webutils.WriteJSON(w, http.StatusCreated, resp)
}

// ListOrders handles GET /api/orders
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	authUserID, err := getAuthenticatedUserID(r)
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	orderList, err := h.OrderRepo.FindUserOrders(r.Context(), authUserID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve orders"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusOK, orderList)
}

// GetOrder handles GET /api/orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	authUserID, err := getAuthenticatedUserID(r)
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orderIDStr := vars["id"]
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid order ID format"), http.StatusBadRequest)
		return
	}

	order, items, err := h.OrderRepo.FindOrderByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to retrieve order"), http.StatusInternalServerError)
		}
		return
	}

	// Authorization check: Ensure the fetched order belongs to the authenticated user
	if order.UserID != authUserID {
		webutils.ErrorJSON(w, errors.New("forbidden"), http.StatusForbidden)
		return
	}

	resp := OrderResponse{
		Order: *order,
		Items: items,
	}
	webutils.WriteJSON(w, http.StatusOK, resp)
}

// CancelOrder handles PATCH /api/orders/{id}/cancel
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	authUserID, err := getAuthenticatedUserID(r)
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	orderIDStr := vars["id"]
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid order ID format"), http.StatusBadRequest)
		return
	}

	// First, verify the order exists and belongs to the user
	order, _, err := h.OrderRepo.FindOrderByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to retrieve order for cancellation check"), http.StatusInternalServerError)
		}
		return
	}
	if order.UserID != authUserID {
		webutils.ErrorJSON(w, errors.New("forbidden"), http.StatusForbidden)
		return
	}

	// Attempt to update the status to cancelled
	err = h.OrderRepo.UpdateOrderStatus(r.Context(), orderID, models.StatusCancelled)
	if err != nil {
		if errors.Is(err, orders.ErrOrderCannotBeCancelled) {
			webutils.ErrorJSON(w, err, http.StatusConflict) // 409 Conflict - cannot cancel in current state
		} else if errors.Is(err, orders.ErrOrderNotFound) {
			// Should not happen if check above passed, but handle defensively
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to cancel order"), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK) // Return 200 OK or 204 No Content
}

// TODO: Implement public TrackOrder handler (GET /api/orders/tracking/{trackingNumber})
// This would likely need a different repository method FindOrderByTrackingNumber
