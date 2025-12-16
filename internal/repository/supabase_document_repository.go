package repository

import (
	"encoding/json"
	"fmt"

	"pdf-text-reader/internal/domain"
)

// SupabaseDocumentRepository implements the domain.DocumentRepository interface
type SupabaseDocumentRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

// NewSupabaseDocumentRepository creates a new Supabase document repository
func NewSupabaseDocumentRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) domain.DocumentRepository {
	return &SupabaseDocumentRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

// Create a new document in Supabase
func (r *SupabaseDocumentRepository) Create(
	document *domain.Document,
	token string,
) error {

	// Use client with token for RLS policies
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	// Serializar campos complejos
	// Content is already a JSON string, validate it's valid JSON
	var contentJSON []byte
	if document.Content != "" {
		// Validate it's valid JSON by trying to unmarshal and remarshal
		var contentInterface interface{}
		if err := json.Unmarshal([]byte(document.Content), &contentInterface); err != nil {
			// If not valid JSON, wrap it as a JSON string
			contentJSON = []byte(document.Content)
		} else {
			contentJSON = []byte(document.Content)
		}
	} else {
		contentJSON = []byte("[]")
	}

	metadataJSON, err := json.Marshal(document.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	data := map[string]interface{}{
		"id":            document.ID,
		"user_id":       document.UserID,
		"original_name": document.OriginalName,
		"title":         document.Title,
		"content":       string(contentJSON),
		"metadata":      string(metadataJSON),
		"file_path":     document.FilePath,
		"file_size":     document.FileSize,
		"created_at":    document.CreatedAt,
		"updated_at":    document.UpdatedAt,
	}

	_, _, err = client.
		From("documents").
		Insert(data, false, "", "", "").
		Execute()

	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	r.logger.Info(
		"Document created",
		"id", document.ID,
		"user_id", document.UserID,
	)

	return nil
}

// GetByID retrieves a document by ID
func (r *SupabaseDocumentRepository) GetByID(id string, token string) (*domain.Document, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("documents").
		Select("*", "", false).
		Eq("id", id).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var documents []map[string]interface{}
	if err := json.Unmarshal(data, &documents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	return r.mapToDocument(documents[0])
}

// GetByUserID retrieves all documents for a user
func (r *SupabaseDocumentRepository) GetByUserID(userID string, token string) ([]*domain.Document, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("documents").
		Select("*", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	var documentsData []map[string]interface{}
	if err := json.Unmarshal(data, &documentsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var documents []*domain.Document
	for _, docData := range documentsData {
		doc, err := r.mapToDocument(docData)
		if err != nil {
			r.logger.Error("Failed to map document", err, "doc_id", docData["id"])
			continue
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// Update updates a document in Supabase
func (r *SupabaseDocumentRepository) Update(document *domain.Document, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	contentJSON, err := json.Marshal(document.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	metadataJSON, err := json.Marshal(document.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	data := map[string]interface{}{
		"title":      document.Title,
		"content":    string(contentJSON),
		"metadata":   string(metadataJSON),
		"updated_at": document.UpdatedAt,
	}

	_, _, err = client.From("documents").
		Update(data, "", "").
		Eq("id", document.ID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// Delete deletes a document from Supabase
func (r *SupabaseDocumentRepository) Delete(id string, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	_, _, err = client.From("documents").
		Delete("", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Search searches documents by title or content
func (r *SupabaseDocumentRepository) Search(userID, query string, token string) ([]*domain.Document, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	// Use Supabase's text search capabilities
	data, _, err := client.From("documents").
		Select("*", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	var documentsData []map[string]interface{}
	if err := json.Unmarshal(data, &documentsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var documents []*domain.Document
	for _, docData := range documentsData {
		doc, err := r.mapToDocument(docData)
		if err != nil {
			r.logger.Error("Failed to map document", err, "doc_id", docData["id"])
			continue
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// mapToDocument converts a map to a Document struct
func (r *SupabaseDocumentRepository) mapToDocument(data map[string]interface{}) (*domain.Document, error) {
	document := &domain.Document{
		ID:           getString(data, "id"),
		UserID:       getString(data, "user_id"),
		OriginalName: getString(data, "original_name"),
		Title:        getString(data, "title"),
		FilePath:     getString(data, "file_path"),
		FileSize:     getInt64(data, "file_size"),
	}

	// Parse timestamps
	if createdAt := getString(data, "created_at"); createdAt != "" {
		// Handle timestamp parsing - Supabase returns ISO format
		// For now, store as string and convert when needed
	}

	// Parse JSON fields
	if contentStr := getString(data, "content"); contentStr != "" {
		if err := json.Unmarshal([]byte(contentStr), &document.Content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content: %w", err)
		}
	}

	if metadataStr := getString(data, "metadata"); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &document.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return document, nil
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key]; ok && val != nil {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}
