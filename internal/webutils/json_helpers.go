package webutils

import (
	"encoding/json"
	"net/http"
)

// WriteJSON sends a JSON response with a specific status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// ReadJSON decodes JSON from a request body into a target struct.
func ReadJSON(r *http.Request, target interface{}) error {
	// Limit request body size (e.g., 1MB) to prevent potential DoS
	// r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	// Note: MaxBytesReader needs ResponseWriter, so maybe apply in middleware or handler

	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

// jsonError is used for standard JSON error responses.
type jsonError struct {
	Error string `json:"error"`
}

// ErrorJSON sends a JSON error response.
func ErrorJSON(w http.ResponseWriter, err error, status int) {
	WriteJSON(w, status, jsonError{Error: err.Error()})
}
