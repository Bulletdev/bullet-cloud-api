package auth

import (
	"bullet-cloud-api/internal/users"    // For UserRepository
	"bullet-cloud-api/internal/webutils" // Changed from handlers
	"context"
	"errors"
	"net/http"
	"strings"
)

// ContextKey is a type used for context keys to avoid collisions.
type ContextKey string

const UserIDContextKey ContextKey = "userID"

// Middleware provides authentication middleware.
type Middleware struct {
	jwtSecret string
	userRepo  users.UserRepository
}

// NewMiddleware creates a new instance of Middleware.
func NewMiddleware(jwtSecret string, userRepo users.UserRepository) *Middleware {
	return &Middleware{
		jwtSecret: jwtSecret,
		userRepo:  userRepo,
	}
}

// Authenticate verifies the JWT token from the Authorization header.
// If valid, it adds the UserID to the request context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			webutils.ErrorJSON(w, errors.New("authorization header required"), http.StatusUnauthorized) // Use webutils
			return
		}

		// Check if the header is in the format "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			webutils.ErrorJSON(w, errors.New("invalid authorization header format"), http.StatusUnauthorized) // Use webutils
			return
		}

		tokenString := headerParts[1]

		// Validate the token
		claims, err := ValidateToken(tokenString, m.jwtSecret)
		if err != nil {
			webutils.ErrorJSON(w, ErrInvalidToken, http.StatusUnauthorized) // Use webutils
			return
		}

		// Optional: Check if user still exists in the database
		_, err = m.userRepo.FindByID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, users.ErrUserNotFound) {
				webutils.ErrorJSON(w, errors.New("user associated with token not found"), http.StatusUnauthorized) // Use webutils
			} else {
				webutils.ErrorJSON(w, errors.New("error verifying user"), http.StatusInternalServerError) // Use webutils
			}
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
