package handlers

import (
	"bullet-cloud-api/internal/addresses"
	"bullet-cloud-api/internal/auth" // For UserIDContextKey
	"bullet-cloud-api/internal/models"
	// For User model
	"bullet-cloud-api/internal/users"    // For UserRepository
	"bullet-cloud-api/internal/webutils" // For JSON helpers
	"errors"
	"log" // Adicionado para log
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux" // Adicionado
)

// UserHandler handles user-related requests, including addresses.
type UserHandler struct {
	UserRepo    users.UserRepository
	AddressRepo addresses.AddressRepository // Adicionado
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userRepo users.UserRepository, addressRepo addresses.AddressRepository) *UserHandler { // Adicionado addressRepo
	return &UserHandler{
		UserRepo:    userRepo,
		AddressRepo: addressRepo, // Adicionado
	}
}

// Helper function to get authenticated user ID from context
func getAuthenticatedUserID(r *http.Request) (uuid.UUID, error) {
	userIDValue := r.Context().Value(auth.UserIDContextKey)
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		// Log this serious issue
		log.Printf("ERROR: User ID not found or not UUID in context for path %s", r.URL.Path)
		return uuid.Nil, errors.New("authentication context error")
	}
	return userID, nil
}

// Helper function to check if the authenticated user matches the user ID in the URL
// Returns the target user ID if authorized, otherwise writes an error and returns Nil UUID.
func checkUserAuthorization(w http.ResponseWriter, r *http.Request, targetUserIDStr string) (uuid.UUID, bool) {
	authUserID, err := getAuthenticatedUserID(r)
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return uuid.Nil, false
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid user ID in URL"), http.StatusBadRequest)
		return uuid.Nil, false
	}

	// For now, only allow users to access their own data
	// TODO: Implement admin role check here if needed for accessing other users' data
	if authUserID != targetUserID {
		webutils.ErrorJSON(w, errors.New("forbidden"), http.StatusForbidden)
		return uuid.Nil, false
	}

	return targetUserID, true
}

// GetMe handles requests for the current user's information.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	authUserID, err := getAuthenticatedUserID(r)
	if err != nil {
		webutils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	user, err := h.UserRepo.FindByID(r.Context(), authUserID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			webutils.ErrorJSON(w, errors.New("user not found"), http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to retrieve user data"), http.StatusInternalServerError)
		}
		return
	}

	user.PasswordHash = ""
	webutils.WriteJSON(w, http.StatusOK, user)
}

// --- Address Handlers ---

// ListAddresses handles GET /api/users/{userId}/addresses
func (h *UserHandler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserIDStr := vars["userId"]

	// Check if the authenticated user is authorized to view these addresses
	targetUserID, authorized := checkUserAuthorization(w, r, targetUserIDStr)
	if !authorized {
		return
	}

	addressList, err := h.AddressRepo.FindByUserID(r.Context(), targetUserID)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to retrieve addresses"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusOK, addressList)
}

// AddAddress handles POST /api/users/{userId}/addresses
type AddAddressRequest struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	IsDefault  bool   `json:"is_default"`
}

func (h *UserHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserIDStr := vars["userId"]

	// Check authorization
	targetUserID, authorized := checkUserAuthorization(w, r, targetUserIDStr)
	if !authorized {
		return
	}

	var req AddAddressRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// Basic Validation
	if req.Street == "" || req.City == "" || req.State == "" || req.PostalCode == "" || req.Country == "" {
		webutils.ErrorJSON(w, errors.New("all address fields are required"), http.StatusBadRequest)
		return
	}

	newAddress := &models.Address{
		UserID:     targetUserID,
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		PostalCode: req.PostalCode,
		Country:    req.Country,
		IsDefault:  req.IsDefault,
	}

	createdAddress, err := h.AddressRepo.Create(r.Context(), newAddress)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to create address"), http.StatusInternalServerError)
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, createdAddress)
}

// UpdateAddress handles PUT /api/users/{userId}/addresses/{addressId}
type UpdateAddressRequest struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	IsDefault  bool   `json:"is_default"`
}

func (h *UserHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserIDStr := vars["userId"]
	addressIDStr := vars["addressId"]

	// Check authorization
	targetUserID, authorized := checkUserAuthorization(w, r, targetUserIDStr)
	if !authorized {
		return
	}

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid address ID format"), http.StatusBadRequest)
		return
	}

	var req UpdateAddressRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// Basic Validation
	if req.Street == "" || req.City == "" || req.State == "" || req.PostalCode == "" || req.Country == "" {
		webutils.ErrorJSON(w, errors.New("all address fields are required"), http.StatusBadRequest)
		return
	}

	addressToUpdate := &models.Address{
		// UserID and ID are used in the repo method query
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		PostalCode: req.PostalCode,
		Country:    req.Country,
		IsDefault:  req.IsDefault,
	}

	updatedAddress, err := h.AddressRepo.Update(r.Context(), targetUserID, addressID, addressToUpdate)
	if err != nil {
		if errors.Is(err, addresses.ErrAddressNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to update address"), http.StatusInternalServerError)
		}
		return
	}

	webutils.WriteJSON(w, http.StatusOK, updatedAddress)
}

// DeleteAddress handles DELETE /api/users/{userId}/addresses/{addressId}
func (h *UserHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserIDStr := vars["userId"]
	addressIDStr := vars["addressId"]

	// Check authorization
	targetUserID, authorized := checkUserAuthorization(w, r, targetUserIDStr)
	if !authorized {
		return
	}

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid address ID format"), http.StatusBadRequest)
		return
	}

	err = h.AddressRepo.Delete(r.Context(), targetUserID, addressID)
	if err != nil {
		if errors.Is(err, addresses.ErrAddressNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to delete address"), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// SetDefaultAddress handles PATCH /api/users/{userId}/addresses/{addressId}/default
func (h *UserHandler) SetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserIDStr := vars["userId"]
	addressIDStr := vars["addressId"]

	// Check authorization
	targetUserID, authorized := checkUserAuthorization(w, r, targetUserIDStr)
	if !authorized {
		return
	}

	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("invalid address ID format"), http.StatusBadRequest)
		return
	}

	err = h.AddressRepo.SetDefault(r.Context(), targetUserID, addressID)
	if err != nil {
		if errors.Is(err, addresses.ErrAddressNotFound) {
			webutils.ErrorJSON(w, err, http.StatusNotFound)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to set default address"), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK) // Return 200 OK on success
}

// TODO: Implement handler for UpdateUser (PUT /api/users/{id})
