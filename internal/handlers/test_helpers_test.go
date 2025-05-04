package handlers_test

import (
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/users"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// --- Common Test Setup --- //

const testJwtSecret = "test-secret-for-jwt-please-change" // Consistent secret for tests

// setupBaseTest initializes common components for handler tests:
// - MockUserRepository
// - AuthMiddleware (configured with test secret and user repo)
// - Mux Router
func setupBaseTest(t *testing.T) (*users.MockUserRepository, *auth.Middleware, *mux.Router) {
	mockUserRepo := new(users.MockUserRepository)
	authMiddleware := auth.NewMiddleware(testJwtSecret, mockUserRepo)
	router := mux.NewRouter()

	return mockUserRepo, authMiddleware, router
}

// generateTestToken creates a JWT for testing purposes.
func generateTestToken(userID uuid.UUID, secret string) string {
	// Use a fixed expiry for consistency unless specific test requires different
	tokenExpiry := time.Hour * 1
	token, err := auth.GenerateToken(userID, secret, tokenExpiry)
	if err != nil {
		// Panic is acceptable in test setup if token generation fails fundamentally
		panic("Failed to generate test token: " + err.Error())
	}
	return token
}
