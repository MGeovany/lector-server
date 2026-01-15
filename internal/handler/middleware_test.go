package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pdf-text-reader/internal/domain"
)

type mockAuthService struct {
	user      *domain.SupabaseUser
	err       error
	lastToken string
	disabled  bool
}

func (m *mockAuthService) ValidateToken(token string) (*domain.SupabaseUser, error) {
	m.lastToken = token
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockAuthService) IsAccountDisabled(userID string, token string) (bool, error) {
	return m.disabled, nil
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	authService := &mockAuthService{}
	logger := NewMockHandlerLogger()

	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Authorization header required") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	authService := &mockAuthService{}
	logger := NewMockHandlerLogger()

	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid authorization header format") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
	authService := &mockAuthService{}
	logger := NewMockHandlerLogger()

	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Token required") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	authService := &mockAuthService{err: errors.New("invalid token")}
	logger := NewMockHandlerLogger()

	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid token") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	authService := &mockAuthService{user: &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}}
	logger := NewMockHandlerLogger()

	called := false
	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, ok := GetUserFromContext(r)
		if !ok || user.ID != "user-1" {
			t.Fatalf("expected user in context")
		}
		token, ok := GetTokenFromContext(r)
		if !ok || token != "good" {
			t.Fatalf("expected token in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !called {
		t.Fatalf("expected next handler to be called")
	}
}

func TestAuthMiddleware_AccountDisabled(t *testing.T) {
	authService := &mockAuthService{
		user:     &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"},
		disabled: true,
	}
	logger := NewMockHandlerLogger()

	middleware := NewAuthMiddleware(authService, logger).Middleware
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Account disabled") {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}
