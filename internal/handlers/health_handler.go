package handlers

import (
	"bullet-cloud-api/internal/webutils"
	"net/http"
)

// HealthCheck returns a simple status indicating the service is up.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// In the future, this could check DB connection, external services, etc.
	status := map[string]string{"status": "healthy"}
	webutils.WriteJSON(w, http.StatusOK, status)
}
