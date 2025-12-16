package handler

import (
	"context"
	"net/http"
	"strings"

	"pdf-text-reader/internal/domain"
)

type AuthMiddleware struct {
	authService domain.AuthService
	logger      domain.Logger
}

func NewAuthMiddleware(
	authService domain.AuthService,
	logger domain.Logger,
) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// Middleware returns a mux-compatible middleware
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

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

		user, err := m.authService.ValidateToken(token)
		if err != nil {
			m.logger.Error("Token validation failed", err)
			writeError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
