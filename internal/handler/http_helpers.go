package handler

import (
	"net/http"
	"pdf-text-reader/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext extracts the authenticated user from request context
func GetUserFromContext(r *http.Request) (*domain.SupabaseUser, bool) {
	user, ok := r.Context().Value(userContextKey).(*domain.SupabaseUser)
	return user, ok
}

// writeError writes an error response (helper function)
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
