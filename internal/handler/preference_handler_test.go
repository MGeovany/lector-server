package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"
)

func TestPreferenceHandler_GetPreferences_OK(t *testing.T) {
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	container := &config.Container{UserPreferencesService: prefService}

	handler := NewPreferenceHandler(container, logger)
	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preferences", nil)
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "token")

	rr := httptest.NewRecorder()
	handler.GetPreferences(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var prefs domain.UserPreferences
	if err := json.Unmarshal(rr.Body.Bytes(), &prefs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if prefs.UserID != "user-1" {
		t.Fatalf("expected user id user-1, got %s", prefs.UserID)
	}
}

func TestPreferenceHandler_UpdatePreferences_OK(t *testing.T) {
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	container := &config.Container{UserPreferencesService: prefService}

	handler := NewPreferenceHandler(container, logger)
	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}

	body := strings.NewReader(`{"font_size":18,"font_family":"serif","theme":"dark","tags":["one","two"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences", body)
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "token")

	rr := httptest.NewRecorder()
	handler.UpdatePreferences(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var prefs domain.UserPreferences
	if err := json.Unmarshal(rr.Body.Bytes(), &prefs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if prefs.FontSize != 18 {
		t.Fatalf("expected font size 18, got %d", prefs.FontSize)
	}
	if prefs.FontFamily != "serif" {
		t.Fatalf("expected font family serif, got %s", prefs.FontFamily)
	}
	if prefs.Theme != "dark" {
		t.Fatalf("expected theme dark, got %s", prefs.Theme)
	}
	if len(prefs.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(prefs.Tags))
	}
}

func TestPreferenceHandler_GetReadingPosition_MissingID(t *testing.T) {
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	container := &config.Container{UserPreferencesService: prefService}

	handler := NewPreferenceHandler(container, logger)
	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preferences/reading-position/", nil)
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "token")

	rr := httptest.NewRecorder()
	handler.GetReadingPosition(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPreferenceHandler_GetReadingPosition_OK(t *testing.T) {
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	container := &config.Container{UserPreferencesService: prefService}

	handler := NewPreferenceHandler(container, logger)
	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}

	prefService.positions["user-1"] = map[string]*domain.ReadingPosition{
		"doc-1": {UserID: "user-1", DocumentID: "doc-1", Progress: 0.6, PageNumber: 3},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preferences/reading-position/doc-1", nil)
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "token")

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/preferences/reading-position/{documentId}", handler.GetReadingPosition).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var position domain.ReadingPosition
	if err := json.Unmarshal(rr.Body.Bytes(), &position); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if position.DocumentID != "doc-1" {
		t.Fatalf("expected document id doc-1, got %s", position.DocumentID)
	}
}

func TestPreferenceHandler_UpdateReadingPosition_OK(t *testing.T) {
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()
	container := &config.Container{UserPreferencesService: prefService}

	handler := NewPreferenceHandler(container, logger)
	user := &domain.SupabaseUser{ID: "user-1", Email: "test@example.com"}

	body := strings.NewReader(`{"progress":0.25,"page_number":2}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/reading-position/doc-1", body)
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "token")

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/preferences/reading-position/{documentId}", handler.UpdateReadingPosition).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var position domain.ReadingPosition
	if err := json.Unmarshal(rr.Body.Bytes(), &position); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if position.Progress != 0.25 {
		t.Fatalf("expected progress 0.25, got %v", position.Progress)
	}
	if position.PageNumber != 2 {
		t.Fatalf("expected page number 2, got %d", position.PageNumber)
	}
}
