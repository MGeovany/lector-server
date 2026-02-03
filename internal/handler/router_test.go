package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"
)

type MockHighlightService struct{}

func (m *MockHighlightService) CreateHighlight(userID string, highlight *domain.Highlight, token string) (*domain.Highlight, error) {
	return &domain.Highlight{ID: "h1", UserID: userID, DocumentID: highlight.DocumentID, Quote: highlight.Quote}, nil
}
func (m *MockHighlightService) ListHighlights(userID string, documentID *string, token string) ([]*domain.Highlight, error) {
	return []*domain.Highlight{}, nil
}
func (m *MockHighlightService) DeleteHighlight(userID string, highlightID string, token string) error { return nil }

func TestNewRouter_Health(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	highlightService := &MockHighlightService{}

	authHandler := NewAuthHandler(&config.Container{})
	adminHandler := NewAdminHandler()
	documentHandler := NewDocumentHandler(docService, prefService, logger)
	preferenceHandler := NewPreferenceHandler(&config.Container{UserPreferencesService: prefService}, logger)
	highlightHandler := NewHighlightHandler(&config.Container{HighlightService: highlightService}, logger)

	router := NewRouter(authHandler, adminHandler, documentHandler, preferenceHandler, highlightHandler, nil, func(next http.Handler) http.Handler { return next })

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
