package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pdf-text-reader/internal/config"
)

func TestNewRouter_Health(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	authHandler := NewAuthHandler(&config.Container{})
	adminHandler := NewAdminHandler()
	documentHandler := NewDocumentHandler(docService, prefService, logger)
	preferenceHandler := NewPreferenceHandler(&config.Container{UserPreferencesService: prefService}, logger)

	router := NewRouter(authHandler, adminHandler, documentHandler, preferenceHandler, func(next http.Handler) http.Handler { return next })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}
