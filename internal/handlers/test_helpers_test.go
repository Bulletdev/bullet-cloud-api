package handlers_test

import (
	"bullet-cloud-api/internal/auth"
	"bullet-cloud-api/internal/users"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
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

// executeRequestAndAssert performs the common steps of:
// 1. Creating an HTTP request.
// 2. Setting JSON Content-Type and optional Authorization headers.
// 3. Executing the request against the provided router.
// 4. Asserting the response status code.
// 5. Asserting that the response body contains the expected substring.
func executeRequestAndAssert(t *testing.T, router *mux.Router, method, url, token string, body io.Reader, expectedStatus int, expectedBodyContains string) {
	req, err := http.NewRequest(method, url, body)
	assert.NoError(t, err, "Failed to create request")

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, expectedStatus, rr.Code, "Status code mismatch for %s %s", method, url)

	if expectedBodyContains != "" {
		assert.Contains(t, rr.Body.String(), expectedBodyContains, "Response body mismatch for %s %s", method, url)
	} else if expectedStatus < 300 { // Only assert empty body for non-redirect/non-error success cases if expectedBodyContains is empty
		// Special case for 204 No Content or similar where empty body is expected
		if rr.Code == http.StatusNoContent {
			assert.Empty(t, rr.Body.String(), "Body should be empty for 204 No Content on %s %s", method, url)
		} // Other success cases might have bodies (like 200 OK returning an object)
		// We rely on expectedBodyContains for those. If it's empty, we don't assert emptiness for 200/201.
	}
}
