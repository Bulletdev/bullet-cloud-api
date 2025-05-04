package handlers

import (
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/users"
	"bullet-cloud-api/internal/webutils"
	"errors"
	"net/http"
	"time"
)

// AuthHandler handles authentication requests.
type AuthHandler struct {
	UserRepo            users.UserRepository
	JwtSecret           string
	TokenExpiryDuration time.Duration
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userRepo users.UserRepository, jwtSecret string, tokenExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		UserRepo:            userRepo,
		JwtSecret:           jwtSecret,
		TokenExpiryDuration: tokenExpiry,
	}
}

// --- Request/Response Structs ---

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// --- Handlers ---

// Register handles new user registration.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.Name == "" || req.Email == "" || req.Password == "" {
		webutils.ErrorJSON(w, errors.New("name, email, and password are required"), http.StatusBadRequest)
		return
	}
	// TODO: Add more robust validation (email format, password strength)

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to hash password"), http.StatusInternalServerError)
		return
	}

	// Create user
	newUser, err := h.UserRepo.Create(r.Context(), req.Name, req.Email, hashedPassword)
	if err != nil {
		if errors.Is(err, users.ErrEmailAlreadyExists) {
			webutils.ErrorJSON(w, err, http.StatusConflict)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to create user"), http.StatusInternalServerError)
		}
		return
	}

	// Return created user (ensure password hash is not included)
	newUser.PasswordHash = "" // Already done in repository, but good practice to double-check
	webutils.WriteJSON(w, http.StatusCreated, newUser)
}

// Login handles user login and JWT generation.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := webutils.ReadJSON(r, &req); err != nil {
		webutils.ErrorJSON(w, errors.New("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		webutils.ErrorJSON(w, errors.New("email and password are required"), http.StatusBadRequest)
		return
	}

	// Find user by email
	user, err := h.UserRepo.FindByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			webutils.ErrorJSON(w, errors.New("invalid email or password"), http.StatusUnauthorized)
		} else {
			webutils.ErrorJSON(w, errors.New("failed to find user"), http.StatusInternalServerError)
		}
		return
	}

	// Check password
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		webutils.ErrorJSON(w, errors.New("invalid email or password"), http.StatusUnauthorized)
		return
	}

	// Generate JWT
	tokenString, err := auth.GenerateToken(user.ID, h.JwtSecret, h.TokenExpiryDuration)
	if err != nil {
		webutils.ErrorJSON(w, errors.New("failed to generate token"), http.StatusInternalServerError)
		return
	}

	// Return token
	webutils.WriteJSON(w, http.StatusOK, LoginResponse{Token: tokenString})
}
