package handler

import (
	"net/http"
	"pdf-text-reader/internal/domain"
)

type contextKey string

const (
	userContextKey  contextKey = "user"
	tokenContextKey contextKey = "token"
)

// GetUserFromContext extracts the authenticated user from request context
func GetUserFromContext(r *http.Request) (*domain.SupabaseUser, bool) {
	user, ok := r.Context().Value(userContextKey).(*domain.SupabaseUser)
	return user, ok
}

// GetTokenFromContext extracts the authentication token from request context
func GetTokenFromContext(r *http.Request) (string, bool) {
	token, ok := r.Context().Value(tokenContextKey).(string)
	return token, ok
}

// writeError writes an error response (helper function)
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
