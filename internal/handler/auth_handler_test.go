package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"
)

func TestAuthHandler_GetProfile_Unauthorized(t *testing.T) {
	handler := NewAuthHandler(&config.Container{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	rr := httptest.NewRecorder()

	handler.GetProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "User not found in context") {
		t.Fatalf("expected error message in response, got %s", rr.Body.String())
	}
}

func TestAuthHandler_GetProfile_OK(t *testing.T) {
	handler := NewAuthHandler(&config.Container{})

	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com", UserMetadata: map[string]interface{}{"name": "test"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	req = createContextWithUser(req, user)

	rr := httptest.NewRecorder()
	handler.GetProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["ID"] != "user-1" {
		t.Fatalf("expected ID user-1, got %v", payload["ID"])
	}
}

func TestAuthHandler_UpdateProfile_BadRequest(t *testing.T) {
	handler := NewAuthHandler(&config.Container{})

	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com", UserMetadata: map[string]interface{}{}}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/profile", strings.NewReader("{bad"))
	req = createContextWithUser(req, user)

	rr := httptest.NewRecorder()
	handler.UpdateProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestAuthHandler_UpdateProfile_OK(t *testing.T) {
	handler := NewAuthHandler(&config.Container{})

	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com", UserMetadata: map[string]interface{}{}}
	body := strings.NewReader(`{"name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/profile", body)
	req = createContextWithUser(req, user)

	rr := httptest.NewRecorder()
	handler.UpdateProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if user.UserMetadata["name"] != "New Name" {
		t.Fatalf("expected user metadata to be updated")
	}
}

func TestAuthHandler_ValidateToken_OK(t *testing.T) {
	handler := NewAuthHandler(&config.Container{})

	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com", UserMetadata: map[string]interface{}{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/validate", nil)
	req = createContextWithUser(req, user)

	rr := httptest.NewRecorder()
	handler.ValidateToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
