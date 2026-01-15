package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"pdf-text-reader/internal/domain"
)

// Mock implementations for handler testing
type MockDocumentService struct {
	documents map[string]*domain.Document
}

func NewMockDocumentService() *MockDocumentService {
	return &MockDocumentService{
		documents: make(map[string]*domain.Document),
	}
}

func (m *MockDocumentService) GetDocumentsByUserID(userID string, token string) ([]*domain.DocumentData, error) {
	var docs []*domain.DocumentData
	for _, doc := range m.documents {
		if doc.UserID == userID {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (m *MockDocumentService) GetDocument(documentID string, token string) (*domain.DocumentData, error) {
	if doc, exists := m.documents[documentID]; exists {
		return doc, nil
	}
	return nil, domain.ErrDocumentNotFound
}

func (m *MockDocumentService) DeleteDocument(documentID string, token string) error {
	if _, exists := m.documents[documentID]; !exists {
		return domain.ErrDocumentNotFound
	}
	delete(m.documents, documentID)
	return nil
}

func (m *MockDocumentService) SearchDocuments(userID, query string, token string) ([]*domain.DocumentData, error) {
	var docs []*domain.DocumentData
	for _, doc := range m.documents {
		if doc.UserID == userID && strings.Contains(strings.ToLower(doc.Title), strings.ToLower(query)) {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (m *MockDocumentService) SetFavorite(userID string, documentID string, isFavorite bool, token string) error {
	if doc, exists := m.documents[documentID]; exists {
		if doc.UserID != userID {
			return domain.ErrAccessDenied
		}
		doc.IsFavorite = isFavorite
		return nil
	}
	return domain.ErrDocumentNotFound
}

func (m *MockDocumentService) UpdateDocumentDetails(userID string, documentID string, title *string, author *string, tag *string, token string) (*domain.DocumentData, error) {
	if doc, exists := m.documents[documentID]; exists {
		if doc.UserID != userID {
			return nil, domain.ErrAccessDenied
		}
		if title != nil {
			doc.Title = *title
		}
		if author != nil {
			doc.Author = author
		}
		if tag != nil {
			doc.Tag = tag
		}
		return doc, nil
	}
	return nil, domain.ErrDocumentNotFound
}

func (m *MockDocumentService) GetDocumentTags(userID string, token string) ([]string, error) {
	// Mock implementation
	return []string{"programming", "tutorial"}, nil
}

func (m *MockDocumentService) CreateTag(userID string, tagName string, token string) error {
	return nil
}

func (m *MockDocumentService) DeleteTag(userID string, tagName string, token string) error {
	return nil
}

func (m *MockDocumentService) Upload(ctx context.Context, userID string, file io.Reader, token string, originalName string) (*domain.DocumentData, error) {
	// Mock implementation
	doc := &domain.DocumentData{
		ID:      "new-doc-id",
		UserID:  userID,
		Title:   originalName,
		Content: json.RawMessage(`[]`),
		Metadata: domain.DocumentMetadata{
			FileSize: 1024,
			Format:   "pdf",
		},
	}
	m.documents[doc.ID] = doc
	return doc, nil
}

type MockUserPreferencesService struct {
	preferences map[string]*domain.UserPreferences
	positions   map[string]map[string]*domain.ReadingPosition
}

func NewMockUserPreferencesService() *MockUserPreferencesService {
	return &MockUserPreferencesService{
		preferences: make(map[string]*domain.UserPreferences),
		positions:   make(map[string]map[string]*domain.ReadingPosition),
	}
}

func (m *MockUserPreferencesService) GetPreferences(userID string, token string) (*domain.UserPreferences, error) {
	if prefs, exists := m.preferences[userID]; exists {
		return prefs, nil
	}
	return &domain.UserPreferences{
		UserID:   userID,
		FontSize: 16,
		Theme:    "light",
	}, nil
}

func (m *MockUserPreferencesService) UpdatePreferences(userID string, prefs *domain.UserPreferences, token string) error {
	m.preferences[userID] = prefs
	return nil
}

func (m *MockUserPreferencesService) GetReadingPosition(userID, documentID string, token string) (*domain.ReadingPosition, error) {
	if userPositions, exists := m.positions[userID]; exists {
		if pos, exists := userPositions[documentID]; exists {
			return pos, nil
		}
	}
	return nil, domain.ErrReadingPositionNotFound
}

func (m *MockUserPreferencesService) GetAllReadingPositions(userID string, token string) (map[string]*domain.ReadingPosition, error) {
	if userPositions, exists := m.positions[userID]; exists {
		return userPositions, nil
	}
	return make(map[string]*domain.ReadingPosition), nil
}

func (m *MockUserPreferencesService) UpdateReadingPosition(userID, documentID string, position *domain.ReadingPosition, token string) error {
	if m.positions[userID] == nil {
		m.positions[userID] = make(map[string]*domain.ReadingPosition)
	}
	m.positions[userID][documentID] = position
	return nil
}

type MockHandlerLogger struct {
	messages []string
}

func NewMockHandlerLogger() *MockHandlerLogger {
	return &MockHandlerLogger{
		messages: []string{},
	}
}

func (m *MockHandlerLogger) Info(msg string, args ...interface{}) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *MockHandlerLogger) Error(msg string, err error, args ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+msg+" - "+err.Error())
}

func (m *MockHandlerLogger) Debug(msg string, args ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *MockHandlerLogger) Warn(msg string, args ...interface{}) {
	m.messages = append(m.messages, "WARN: "+msg)
}

// Test context helpers
func createContextWithUser(r *http.Request, user *domain.SupabaseUser) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func createContextWithToken(r *http.Request, token string) *http.Request {
	ctx := context.WithValue(r.Context(), tokenContextKey, token)
	return r.WithContext(ctx)
}

func TestDocumentHandler_GetDocumentsByUserID(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Test Document",
	}
	docService.documents["doc1"] = doc

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/users/user1/documents", nil)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/users/{id}/documents", handler.GetDocumentsByUserID).Methods("GET")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response body
	var docs []*domain.Document
	err := json.Unmarshal(rr.Body.Bytes(), &docs)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", docs[0].ID)
	}
}

func TestDocumentHandler_GetDocument(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Test Document",
	}
	docService.documents["doc1"] = doc

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/documents/doc1", nil)
	user := &domain.SupabaseUser{ID: "user1", Email: "test@example.com"}
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/{id}", handler.GetDocument).Methods("GET")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response body
	var responseDoc domain.Document
	err := json.Unmarshal(rr.Body.Bytes(), &responseDoc)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if responseDoc.ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", responseDoc.ID)
	}
}

func TestDocumentHandler_SearchDocuments(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test documents
	doc1 := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Go Programming Guide",
	}
	doc2 := &domain.Document{
		ID:     "doc2",
		UserID: "user1",
		Title:  "Python Tutorial",
	}
	docService.documents["doc1"] = doc1
	docService.documents["doc2"] = doc2

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/documents/search?q=Go", nil)
	user := &domain.SupabaseUser{ID: "user1", Email: "test@example.com"}
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/search", handler.SearchDocuments).Methods("GET")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response body
	var docs []*domain.Document
	err := json.Unmarshal(rr.Body.Bytes(), &docs)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", docs[0].ID)
	}
}

func TestDocumentHandler_SetFavorite(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test document
	doc := &domain.Document{
		ID:         "doc1",
		UserID:     "user1",
		Title:      "Test Document",
		IsFavorite: false,
	}
	docService.documents["doc1"] = doc

	// Create request body
	reqBody := setFavoriteRequest{IsFavorite: true}
	body, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("PUT", "/api/v1/documents/doc1/favorite", bytes.NewReader(body))
	user := &domain.SupabaseUser{ID: "user1", Email: "test@example.com"}
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/{id}/favorite", handler.SetFavorite).Methods("PUT")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check that document was marked as favorite
	if !doc.IsFavorite {
		t.Error("Expected document to be marked as favorite")
	}
}

func TestDocumentHandler_UpdateDocument(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Original Title",
	}
	docService.documents["doc1"] = doc

	// Create request body
	newTitle := "Updated Title"
	reqBody := updateDocumentRequest{Title: &newTitle}
	body, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("PUT", "/api/v1/documents/doc1", bytes.NewReader(body))
	user := &domain.SupabaseUser{ID: "user1", Email: "test@example.com"}
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/{id}", handler.UpdateDocument).Methods("PUT")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response body
	var responseDoc domain.Document
	err := json.Unmarshal(rr.Body.Bytes(), &responseDoc)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if responseDoc.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, responseDoc.Title)
	}
}

func TestDocumentHandler_DeleteDocument(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Test Document",
	}
	docService.documents["doc1"] = doc

	// Create request
	req := httptest.NewRequest("DELETE", "/api/v1/documents/doc1", nil)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/{id}", handler.DeleteDocument).Methods("DELETE")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check that document was deleted
	if _, exists := docService.documents["doc1"]; exists {
		t.Error("Expected document to be deleted")
	}
}

func TestDocumentHandler_GetDocumentTags(t *testing.T) {
	docService := NewMockDocumentService()
	prefService := NewMockUserPreferencesService()
	logger := NewMockHandlerLogger()

	handler := NewDocumentHandler(docService, prefService, logger)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/documents/tags", nil)
	user := &domain.SupabaseUser{ID: "user1", Email: "test@example.com"}
	req = createContextWithUser(req, user)
	req = createContextWithToken(req, "test-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create router with the handler
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/documents/tags", handler.GetDocumentTags).Methods("GET")

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check response body
	var tags []string
	err := json.Unmarshal(rr.Body.Bytes(), &tags)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}
