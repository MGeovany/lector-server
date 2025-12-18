package repository

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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

	// Serialize complex fields as JSON strings
	// Aggressively clean JSON to remove \u0000 sequences that cause PostgreSQL 22P05 errors
	var contentJSONStr string
	if len(document.Content) > 0 {
		// Validate JSON is valid by unmarshaling and re-marshaling
		var contentData interface{}
		if err := json.Unmarshal(document.Content, &contentData); err != nil {
			r.logger.Warn("Failed to unmarshal content JSON, using empty array", "error", err)
			contentJSONStr = "[]"
		} else {
			// Re-marshal with proper escaping
			cleanedJSON, err := json.Marshal(contentData)
			if err != nil {
				r.logger.Warn("Failed to re-marshal content JSON, using empty array", "error", err)
				contentJSONStr = "[]"
			} else {
				// Remove any \u0000 sequences that cause 22P05 errors
				jsonStr := string(cleanedJSON)
				jsonStr = strings.ReplaceAll(jsonStr, "\\u0000", "")
				jsonStr = strings.ReplaceAll(jsonStr, "\\u000", "")
				// Verify the cleaned JSON is still valid
				var verify interface{}
				if err := json.Unmarshal([]byte(jsonStr), &verify); err != nil {
					r.logger.Warn("Cleaned JSON is invalid, using empty array", "error", err)
					contentJSONStr = "[]"
				} else {
					// Re-marshal the verified data
					finalJSON, err := json.Marshal(verify)
					if err != nil {
						contentJSONStr = "[]"
					} else {
						contentJSONStr = string(finalJSON)
					}
				}
			}
		}
	} else {
		contentJSONStr = "[]"
	}

	// Marshal metadata to JSON string
	metadataJSON, err := json.Marshal(document.Metadata)
	var metadataJSONStr string
	if err != nil {
		r.logger.Warn("Failed to marshal metadata", "error", err)
		metadataJSONStr = "{}"
	} else {
		metadataJSONStr = string(metadataJSON)
	}

	// Use a regex-based approach to remove problematic Unicode sequences
	// Remove \u0000 and other problematic sequences from the JSON string
	// This is critical to avoid PostgreSQL 22P05 errors
	cleanedContentJSON := r.removeProblematicUnicode(contentJSONStr)
	cleanedMetadataJSON := r.removeProblematicUnicode(metadataJSONStr)
	
	// Parse back to interface{} and re-marshal to ensure clean JSON
	// This double-pass ensures that any problematic sequences are removed
	var contentData interface{}
	if err := json.Unmarshal([]byte(cleanedContentJSON), &contentData); err != nil {
		r.logger.Warn("Failed to parse cleaned content JSON for insert", "error", err)
		contentData = []interface{}{}
	} else {
		// Re-marshal to ensure clean JSON encoding
		reMarshaled, err := json.Marshal(contentData)
		if err != nil {
			r.logger.Warn("Failed to re-marshal content JSON", "error", err)
			contentData = []interface{}{}
		} else {
			// Clean again after re-marshaling (client might add escapes)
			reMarshaledStr := r.removeProblematicUnicode(string(reMarshaled))
			if err := json.Unmarshal([]byte(reMarshaledStr), &contentData); err != nil {
				r.logger.Warn("Failed to parse re-marshaled content JSON", "error", err)
				contentData = []interface{}{}
			}
		}
	}
	
	var metadataData interface{}
	if err := json.Unmarshal([]byte(cleanedMetadataJSON), &metadataData); err != nil {
		r.logger.Warn("Failed to parse cleaned metadata JSON for insert", "error", err)
		metadataData = map[string]interface{}{}
	} else {
		// Re-marshal to ensure clean JSON encoding
		reMarshaled, err := json.Marshal(metadataData)
		if err != nil {
			r.logger.Warn("Failed to re-marshal metadata JSON", "error", err)
			metadataData = map[string]interface{}{}
		} else {
			// Clean again after re-marshaling
			reMarshaledStr := r.removeProblematicUnicode(string(reMarshaled))
			if err := json.Unmarshal([]byte(reMarshaledStr), &metadataData); err != nil {
				r.logger.Warn("Failed to parse re-marshaled metadata JSON", "error", err)
				metadataData = map[string]interface{}{}
			}
		}
	}
	
	// Final validation: serialize the entire data structure to JSON, clean it, and re-parse
	// This ensures that the client won't introduce problematic Unicode sequences
	tempData := map[string]interface{}{
		"id":            document.ID,
		"user_id":       document.UserID,
		"original_name": document.OriginalName,
		"title":         document.Title,
		"content":       contentData,
		"metadata":      metadataData,
		"file_path":     document.FilePath,
		"file_size":     document.FileSize,
		"created_at":    document.CreatedAt,
		"updated_at":    document.UpdatedAt,
	}
	
	// Serialize to JSON to check for problematic sequences
	tempJSON, err := json.Marshal(tempData)
	if err != nil {
		r.logger.Error("Failed to marshal data for validation", err, "doc_id", document.ID)
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	// Clean the JSON string
	cleanedTempJSON := r.removeProblematicUnicode(string(tempJSON))
	
	// Parse back to ensure it's valid
	var finalData map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedTempJSON), &finalData); err != nil {
		r.logger.Error("Failed to unmarshal cleaned data", err, "doc_id", document.ID)
		return fmt.Errorf("failed to validate cleaned JSON: %w", err)
	}
	
	// Use the cleaned and validated data
	_, _, err = client.From("documents").Insert(finalData, false, "", "", "").Execute()
	if err != nil {
		// Log the error details for debugging
		r.logger.Error("Failed to insert document in Supabase", err,
			"doc_id", document.ID,
			"content_length", len(contentJSONStr),
			"metadata_length", len(metadataJSONStr),
		)
		return fmt.Errorf("failed to create document: %w", err)
	}

	r.logger.Info(
		"Document created",
		"id", document.ID,
		"user_id", document.UserID,
	)

	return nil
}

// removeProblematicUnicode removes problematic Unicode escape sequences from JSON strings
// Specifically targets \u0000 and other sequences that cause PostgreSQL 22P05 errors
// PostgreSQL is very strict about Unicode escape sequences in JSONB
func (r *SupabaseDocumentRepository) removeProblematicUnicode(jsonStr string) string {
	// First, remove all control character Unicode escapes (0000-001F)
	// These are the most common cause of PostgreSQL 22P05 errors
	reControlChars := regexp.MustCompile(`\\u00[0-1][0-9a-fA-F]`)
	jsonStr = reControlChars.ReplaceAllString(jsonStr, "")
	
	// Remove surrogate pairs (0xD800-0xDFFF) which are invalid in JSON
	reSurrogates := regexp.MustCompile(`\\u[dD][89aAbBcCdDeEfF][0-9a-fA-F]{2}`)
	jsonStr = reSurrogates.ReplaceAllString(jsonStr, "")
	
	// Remove any remaining problematic Unicode escapes that might cause issues
	// Specifically target sequences that PostgreSQL might reject
	reProblematic := regexp.MustCompile(`\\u000[0-9a-fA-F]|\\u00[01][0-9a-fA-F]|\\u001[0-9a-fA-F]`)
	jsonStr = reProblematic.ReplaceAllString(jsonStr, "")
	
	// Remove any literal NULL bytes and other control characters
	jsonStr = strings.ReplaceAll(jsonStr, "\x00", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\u0000", "")
	
	// Verify the cleaned JSON is still valid
	var verify interface{}
	if err := json.Unmarshal([]byte(jsonStr), &verify); err != nil {
		// If invalid, try to recover by removing ALL Unicode escape sequences
		// and re-encoding properly
		reAllUnicode := regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
		matches := reAllUnicode.FindAllString(jsonStr, -1)
		for _, match := range matches {
			// Extract the hex value
			hexStr := match[2:] // Remove \u prefix
			// Check if it's a control character or surrogate
			if len(hexStr) == 4 {
				// Remove control characters (0000-001F) and surrogates (D800-DFFF)
				if (hexStr[0] == '0' && hexStr[1] == '0' && hexStr[2] <= '1') ||
					(hexStr[0] == 'd' || hexStr[0] == 'D') && (hexStr[1] >= '8' && hexStr[1] <= 'f' || hexStr[1] >= '8' && hexStr[1] <= 'F') {
				jsonStr = strings.ReplaceAll(jsonStr, match, "")
			}
			}
		}
		
		// Try to unmarshal again
		if err := json.Unmarshal([]byte(jsonStr), &verify); err != nil {
			// Last resort: remove ALL Unicode escapes and let Go re-encode
			jsonStr = reAllUnicode.ReplaceAllStringFunc(jsonStr, func(match string) string {
				return "" // Remove all Unicode escapes
			})
		}
	}
	
	return jsonStr
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

// Update a document in Supabase
func (r *SupabaseDocumentRepository) Update(document *domain.Document, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	// Parse JSON to interface{} so Supabase client can properly serialize it as JSONB
	var contentData interface{}
	if len(document.Content) > 0 {
		// Validate and normalize the JSON
		if err := json.Unmarshal(document.Content, &contentData); err != nil {
			r.logger.Warn("Failed to unmarshal content JSON in update, using empty array", "error", err)
			contentData = []interface{}{}
		}
	} else {
		contentData = []interface{}{}
	}

	// Convert metadata to interface{} for proper JSONB handling
	var metadataData interface{}
	metadataJSON, err := json.Marshal(document.Metadata)
	if err != nil {
		r.logger.Warn("Failed to marshal metadata in update", "error", err)
		metadataData = map[string]interface{}{}
	} else {
		// Unmarshal to interface{} so it's properly handled as JSONB
		if err := json.Unmarshal(metadataJSON, &metadataData); err != nil {
			r.logger.Warn("Failed to unmarshal metadata JSON in update", "error", err)
			metadataData = map[string]interface{}{}
		}
	}

	data := map[string]interface{}{
		"title":      document.Title,
		"content":    contentData,  // Pass as interface{} for JSONB
		"metadata":   metadataData, // Pass as interface{} for JSONB
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

	// Parse JSON fields - handle both string and object/array formats
	if contentVal, ok := data["content"]; ok && contentVal != nil {
		// Content can come as string (JSON) or as array/object (JSONB)
		if contentStr, ok := contentVal.(string); ok {
			// It's a string, use it directly
			document.Content = json.RawMessage(contentStr)
		} else {
			// It's already an object/array, marshal it back to JSON
			contentJSON, err := json.Marshal(contentVal)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal content: %w", err)
			}
			document.Content = json.RawMessage(contentJSON)
		}
	} else {
		// Empty content
		document.Content = json.RawMessage("[]")
	}

	if metadataVal, ok := data["metadata"]; ok && metadataVal != nil {
		// Metadata can come as string (JSON) or as object (JSONB)
		if metadataStr, ok := metadataVal.(string); ok {
			// It's a string, unmarshal it
			if err := json.Unmarshal([]byte(metadataStr), &document.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		} else {
			// It's already an object, marshal and unmarshal to convert
			metadataJSON, err := json.Marshal(metadataVal)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}
			if err := json.Unmarshal(metadataJSON, &document.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
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
