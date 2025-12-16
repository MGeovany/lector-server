package handler

import (
	"context"
	"net/http"
	"strings"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"
)

// AuthMiddleware validates Supabase JWT tokens
func AuthMiddleware(container *config.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Extract token from "Bearer <token>" format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			token := parts[1]
			if token == "" {
				writeError(w, http.StatusUnauthorized, "Token required")
				return
			}

			// Validate token with Supabase
			supabaseClient := container.GetSupabaseClient()
			user, err := supabaseClient.ValidateToken(token)
			if err != nil {
				container.GetLogger().Error("Token validation failed", err, "token", token[:10]+"...")
				writeError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			// Add user to request context
			ctx := context.WithValue(r.Context(), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext extracts the authenticated user from request context
func GetUserFromContext(r *http.Request) (*domain.SupabaseUser, bool) {
	user, ok := r.Context().Value("user").(*domain.SupabaseUser)
	return user, ok
}

// writeError writes an error response (helper function)
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
