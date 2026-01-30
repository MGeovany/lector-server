package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"pdf-text-reader/internal/domain"
)

// Mock implementations for testing
type MockDocumentRepository struct {
	documents map[string]*domain.Document
	tags      map[string][]string
}

func NewMockDocumentRepository() *MockDocumentRepository {
	return &MockDocumentRepository{
		documents: make(map[string]*domain.Document),
		tags:      make(map[string][]string),
	}
}

func (m *MockDocumentRepository) Create(document *domain.Document, token string) error {
	if document.ID == "" {
		return errors.New("document ID is required")
	}
	m.documents[document.ID] = document
	return nil
}

func (m *MockDocumentRepository) GetByID(id string, token string) (*domain.Document, error) {
	if doc, exists := m.documents[id]; exists {
		return doc, nil
	}
	return nil, errors.New("document not found")
}

func (m *MockDocumentRepository) GetOptimizedByID(id string, token string) (*domain.OptimizedDocument, error) {
	// Minimal mock: return ready with empty pages if doc exists.
	if _, exists := m.documents[id]; !exists {
		return nil, errors.New("document not found")
	}
	return &domain.OptimizedDocument{
		DocumentID:       id,
		ProcessingStatus: "ready",
		OptimizedVersion: 1,
		Pages:            []string{},
	}, nil
}

func (m *MockDocumentRepository) GetOptimizedMetaByID(id string, token string) (*domain.OptimizedDocument, error) {
	return m.GetOptimizedByID(id, token)
}

func (m *MockDocumentRepository) GetByUserID(userID string, token string) ([]*domain.Document, error) {
	var docs []*domain.Document
	for _, doc := range m.documents {
		if doc.UserID == userID {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (m *MockDocumentRepository) Update(document *domain.Document, token string) error {
	if _, exists := m.documents[document.ID]; !exists {
		return errors.New("document not found")
	}
	m.documents[document.ID] = document
	return nil
}

func (m *MockDocumentRepository) Delete(id string, token string) error {
	if _, exists := m.documents[id]; !exists {
		return errors.New("document not found")
	}
	delete(m.documents, id)
	return nil
}

func (m *MockDocumentRepository) Search(userID, query string, token string) ([]*domain.Document, error) {
	var docs []*domain.Document
	for _, doc := range m.documents {
		if doc.UserID == userID && strings.Contains(strings.ToLower(doc.Title), strings.ToLower(query)) {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (m *MockDocumentRepository) GetTagsByUserID(userID string, token string) ([]string, error) {
	return m.tags[userID], nil
}

func (m *MockDocumentRepository) CreateTag(userID string, tagName string, token string) error {
	if m.tags[userID] == nil {
		m.tags[userID] = []string{}
	}
	m.tags[userID] = append(m.tags[userID], tagName)
	return nil
}

func (m *MockDocumentRepository) DeleteTag(userID string, tagName string, token string) error {
	tags := m.tags[userID]
	for i, tag := range tags {
		if tag == tagName {
			m.tags[userID] = append(tags[:i], tags[i+1:]...)
			return nil
		}
	}
	return errors.New("tag not found")
}

func (m *MockDocumentRepository) SetFavorite(userID string, documentID string, isFavorite bool, token string) error {
	if doc, exists := m.documents[documentID]; exists {
		doc.IsFavorite = isFavorite
		return nil
	}
	return errors.New("document not found")
}

type MockStorageService struct {
	files map[string][]byte
}

func NewMockStorageService() *MockStorageService {
	return &MockStorageService{
		files: make(map[string][]byte),
	}
}

func (m *MockStorageService) Upload(ctx context.Context, path string, file io.Reader, token string) error {
	// Simplified mock - just record that upload was called
	return nil
}

type MockLogger struct {
	messages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		messages: []string{},
	}
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *MockLogger) Error(msg string, err error, args ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+msg+" - "+err.Error())
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func TestDocumentService_GetDocumentsByUserID(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create test documents
	doc1 := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Document 1",
	}
	doc2 := &domain.Document{
		ID:     "doc2",
		UserID: "user2",
		Title:  "Document 2",
	}

	_ = repo.Create(doc1, "token")
	_ = repo.Create(doc2, "token")

	// Test getting documents for user1
	docs, err := service.GetDocumentsByUserID("user1", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", docs[0].ID)
	}
}

func TestDocumentService_GetDocument(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Document 1",
	}

	_ = repo.Create(doc, "token")

	// Test getting existing document
	retrievedDoc, err := service.GetDocument("doc1", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if retrievedDoc.ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", retrievedDoc.ID)
	}

	// Test getting non-existent document
	_, err = service.GetDocument("nonexistent", "token")
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

func TestDocumentService_DeleteDocument(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Document 1",
	}

	_ = repo.Create(doc, "token")

	// Verify document exists
	_, err := repo.GetByID("doc1", "token")
	if err != nil {
		t.Error("Expected document to exist before deletion")
	}

	// Delete document
	err = service.DeleteDocument("doc1", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify document is deleted
	_, err = repo.GetByID("doc1", "token")
	if err == nil {
		t.Error("Expected document to be deleted")
	}
}

func TestDocumentService_SearchDocuments(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

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
	doc3 := &domain.Document{
		ID:     "doc3",
		UserID: "user2",
		Title:  "Go Web Development",
	}

	_ = repo.Create(doc1, "token")
	_ = repo.Create(doc2, "token")
	_ = repo.Create(doc3, "token")

	// Test searching for "Go"
	docs, err := service.SearchDocuments("user1", "Go", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].ID != "doc1" {
		t.Errorf("Expected document ID 'doc1', got '%s'", docs[0].ID)
	}

	// Test searching for "Python"
	docs, err = service.SearchDocuments("user1", "Python", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].ID != "doc2" {
		t.Errorf("Expected document ID 'doc2', got '%s'", docs[0].ID)
	}
}

func TestDocumentService_SetFavorite(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create test document
	doc := &domain.Document{
		ID:         "doc1",
		UserID:     "user1",
		Title:      "Document 1",
		IsFavorite: false,
	}

	_ = repo.Create(doc, "token")

	// Test setting favorite to true
	err := service.SetFavorite("user1", "doc1", true, "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedDoc, _ := repo.GetByID("doc1", "token")
	if !updatedDoc.IsFavorite {
		t.Error("Expected document to be marked as favorite")
	}

	// Test setting favorite to false
	err = service.SetFavorite("user1", "doc1", false, "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedDoc, _ = repo.GetByID("doc1", "token")
	if updatedDoc.IsFavorite {
		t.Error("Expected document to not be marked as favorite")
	}

	// Test setting favorite for different user (should fail)
	err = service.SetFavorite("user2", "doc1", true, "token")
	if err == nil {
		t.Error("Expected error when setting favorite for different user")
	}
}

func TestDocumentService_UpdateDocumentDetails(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create test document
	doc := &domain.Document{
		ID:     "doc1",
		UserID: "user1",
		Title:  "Document 1",
	}

	_ = repo.Create(doc, "token")

	// Test updating title
	newTitle := "Updated Title"
	updatedDoc, err := service.UpdateDocumentDetails("user1", "doc1", &newTitle, nil, nil, "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if updatedDoc.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updatedDoc.Title)
	}

	// Test updating author
	newAuthor := "Updated Author"
	updatedDoc, err = service.UpdateDocumentDetails("user1", "doc1", nil, &newAuthor, nil, "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if *updatedDoc.Author != newAuthor {
		t.Errorf("Expected author '%s', got '%s'", newAuthor, *updatedDoc.Author)
	}

	// Test updating for different user (should fail)
	_, err = service.UpdateDocumentDetails("user2", "doc1", &newTitle, nil, nil, "token")
	if err == nil {
		t.Error("Expected error when updating for different user")
	}
}

func TestDocumentService_GetDocumentTags(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Add some tags for user1
	_ = repo.CreateTag("user1", "programming", "token")
	_ = repo.CreateTag("user1", "tutorial", "token")
	_ = repo.CreateTag("user2", "design", "token")

	// Test getting tags for user1
	tags, err := service.GetDocumentTags("user1", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// Test getting tags for user2
	tags, err = service.GetDocumentTags("user2", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}
}

func TestDocumentService_CreateTag(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Test creating valid tag
	err := service.CreateTag("user1", "programming", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test creating tag with empty name (should fail)
	err = service.CreateTag("user1", "", "token")
	if err == nil {
		t.Error("Expected error for empty tag name")
	}

	// Test creating tag with only whitespace (should fail)
	err = service.CreateTag("user1", "   ", "token")
	if err == nil {
		t.Error("Expected error for whitespace-only tag name")
	}
}

func TestDocumentService_DeleteTag(t *testing.T) {
	repo := NewMockDocumentRepository()
	storage := NewMockStorageService()
	logger := NewMockLogger()

	service := NewDocumentService(repo, nil, storage, logger)

	// Create a tag first
	_ = repo.CreateTag("user1", "programming", "token")

	// Test deleting existing tag
	err := service.DeleteTag("user1", "programming", "token")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test deleting non-existent tag (should fail)
	err = service.DeleteTag("user1", "nonexistent", "token")
	if err == nil {
		t.Error("Expected error for non-existent tag")
	}

	// Test deleting tag with empty name (should fail)
	err = service.DeleteTag("user1", "", "token")
	if err == nil {
		t.Error("Expected error for empty tag name")
	}
}
