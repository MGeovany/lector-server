package repository

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pdf-text-reader/internal/domain"
)

// DocumentRepository implements the domain.DocumentRepository interface
type DocumentRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

// NewSupabaseDocumentRepository creates a new Supabase document repository

func NewDocumentRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) domain.DocumentRepository {
	return &DocumentRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

// Create a new document in Supabase
func (r *DocumentRepository) Create(
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

	// Optional optimized_content (JSONB)
	var optimizedData interface{}
	if len(document.OptimizedContent) > 0 {
		if err := json.Unmarshal(document.OptimizedContent, &optimizedData); err != nil {
			r.logger.Warn("Failed to unmarshal optimized_content JSON for insert", "error", err)
			optimizedData = nil
		}
	}

	// Final validation: serialize the entire data structure to JSON, clean it, and re-parse
	// This ensures that the client won't introduce problematic Unicode sequences
	tempData := map[string]interface{}{
		"id":         document.ID,
		"user_id":    document.UserID,
		"title":      document.Title,
		"content":    contentData,
		"metadata":   metadataData,
		"created_at": document.CreatedAt,
		"updated_at": document.UpdatedAt,
	}

	// Add optional fields if they exist
	if document.Author != nil {
		tempData["author"] = *document.Author
	}
	if document.Description != nil {
		tempData["description"] = *document.Description
	}

	// Offline-first columns
	if document.OriginalStoragePath != nil {
		tempData["original_storage_path"] = *document.OriginalStoragePath
	}
	if document.OriginalFileName != nil {
		tempData["original_file_name"] = *document.OriginalFileName
	}
	if document.OriginalMimeType != nil {
		tempData["original_mime_type"] = *document.OriginalMimeType
	}
	if document.OriginalSizeBytes != nil {
		tempData["original_size_bytes"] = *document.OriginalSizeBytes
	}
	if document.OriginalChecksumSHA256 != nil {
		tempData["original_checksum_sha256"] = *document.OriginalChecksumSHA256
	}
	if optimizedData != nil {
		tempData["optimized_content"] = optimizedData
	}
	if document.OptimizedVersion > 0 {
		tempData["optimized_version"] = document.OptimizedVersion
	}
	if document.OptimizedSizeBytes != nil {
		tempData["optimized_size_bytes"] = *document.OptimizedSizeBytes
	}
	if document.OptimizedChecksumSHA256 != nil {
		tempData["optimized_checksum_sha256"] = *document.OptimizedChecksumSHA256
	}
	if document.ProcessingStatus != "" {
		tempData["processing_status"] = document.ProcessingStatus
	}
	if document.ProcessingError != nil {
		tempData["processing_error"] = *document.ProcessingError
	}
	if document.LanguageCode != nil {
		tempData["language_code"] = *document.LanguageCode
	}
	if document.ProcessedAt != nil {
		tempData["processed_at"] = *document.ProcessedAt
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
func (r *DocumentRepository) removeProblematicUnicode(jsonStr string) string {
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
func (r *DocumentRepository) GetByID(id string, token string) (*domain.Document, error) {
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

	docData := documents[0]

	// Best-effort: populate favorite flag.
	// We only have document_id here; we can read user_id from the document row and check favorites.
	if userID := getString(docData, "user_id"); userID != "" {
		isFav, favErr := r.isFavorite(userID, id, token)
		if favErr == nil {
			docData["is_favorite"] = isFav
		}
	}

	// Fetch tag for this document - get tag_id first, then get tag name
	docTagsData, _, err := client.From("document_tags").
		Select("tag_id", "", false).
		Eq("document_id", id).
		Limit(1, "").
		Execute()
	if err == nil {
		var docTagsList []map[string]interface{}
		if err := json.Unmarshal(docTagsData, &docTagsList); err == nil && len(docTagsList) > 0 {
			tagID := getString(docTagsList[0], "tag_id")
			if tagID != "" {
				// Get tag name from user_tags
				tagData, _, err := client.From("user_tags").
					Select("name", "", false).
					Eq("id", tagID).
					Execute()
				if err == nil {
					var tags []map[string]interface{}
					if err := json.Unmarshal(tagData, &tags); err == nil && len(tags) > 0 {
						tagName := getString(tags[0], "name")
						if tagName != "" {
							docData["tag"] = tagName
						}
					}
				}
			}
		}
	}

	return r.mapToDocument(docData)
}

// GetOptimizedByID retrieves the lightweight optimized pages payload for a document.
func (r *DocumentRepository) GetOptimizedByID(id string, token string) (*domain.OptimizedDocument, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("documents").
		Select("id,processing_status,optimized_content,optimized_version,optimized_checksum_sha256,optimized_size_bytes,language_code,processed_at,user_id", "", false).
		Eq("id", id).
		Limit(1, "").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get optimized document: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	row := rows[0]

	out := &domain.OptimizedDocument{
		DocumentID:       getString(row, "id"),
		UserID:           getString(row, "user_id"),
		ProcessingStatus: getString(row, "processing_status"),
		OptimizedVersion: getInt(row, "optimized_version"),
	}
	if out.ProcessingStatus == "" {
		out.ProcessingStatus = "ready"
	}
	if out.OptimizedVersion <= 0 {
		out.OptimizedVersion = 1
	}
	if v := getString(row, "optimized_checksum_sha256"); v != "" {
		out.OptimizedChecksumSHA = &v
	}
	if v := getInt64(row, "optimized_size_bytes"); v > 0 {
		out.OptimizedSizeBytes = &v
	}
	if v := getString(row, "language_code"); v != "" {
		out.LanguageCode = &v
	}
	if processedAt := getString(row, "processed_at"); processedAt != "" {
		if t, err := time.Parse(time.RFC3339, processedAt); err == nil {
			out.ProcessedAt = &t
		} else if t, err := time.Parse(time.RFC3339Nano, processedAt); err == nil {
			out.ProcessedAt = &t
		}
	}

	// Decode optimized_content into []string (it may arrive as string or []any).
	if val, ok := row["optimized_content"]; ok && val != nil {
		var pages []string
		switch v := val.(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &pages)
		default:
			b, err := json.Marshal(v)
			if err == nil {
				_ = json.Unmarshal(b, &pages)
			}
		}
		if pages != nil {
			out.Pages = pages
		}
	}

	return out, nil
}

// GetOptimizedMetaByID retrieves optimized metadata without the pages payload.
func (r *DocumentRepository) GetOptimizedMetaByID(id string, token string) (*domain.OptimizedDocument, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("documents").
		Select("id,processing_status,optimized_version,optimized_checksum_sha256,optimized_size_bytes,language_code,processed_at,user_id", "", false).
		Eq("id", id).
		Limit(1, "").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get optimized document meta: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	row := rows[0]
	out := &domain.OptimizedDocument{
		DocumentID:       getString(row, "id"),
		UserID:           getString(row, "user_id"),
		ProcessingStatus: getString(row, "processing_status"),
		OptimizedVersion: getInt(row, "optimized_version"),
	}
	if out.ProcessingStatus == "" {
		out.ProcessingStatus = "ready"
	}
	if out.OptimizedVersion <= 0 {
		out.OptimizedVersion = 1
	}
	if v := getString(row, "optimized_checksum_sha256"); v != "" {
		out.OptimizedChecksumSHA = &v
	}
	if v := getInt64(row, "optimized_size_bytes"); v > 0 {
		out.OptimizedSizeBytes = &v
	}
	if v := getString(row, "language_code"); v != "" {
		out.LanguageCode = &v
	}
	if processedAt := getString(row, "processed_at"); processedAt != "" {
		if t, err := time.Parse(time.RFC3339, processedAt); err == nil {
			out.ProcessedAt = &t
		} else if t, err := time.Parse(time.RFC3339Nano, processedAt); err == nil {
			out.ProcessedAt = &t
		}
	}
	return out, nil
}

// GetByUserID retrieves all documents for a user
func (r *DocumentRepository) GetByUserID(userID string, token string) ([]*domain.Document, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	// Select all fields except content to reduce payload size when listing documents
	// Content is only needed when opening a specific document for reading
	data, _, err := client.From("documents").
		Select("id,user_id,title,author,description,metadata,created_at,updated_at,processing_status,processed_at,optimized_version,optimized_size_bytes,optimized_checksum_sha256,language_code,original_size_bytes", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	var documentsData []map[string]interface{}
	if err := json.Unmarshal(data, &documentsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Fetch favorites for user once and mark docs.
	favIDs, favErr := r.favoriteIDsByUser(userID, token)
	if favErr != nil {
		r.logger.Warn("Failed to fetch favorites for user", "error", favErr, "user_id", userID)
	}

	// Get all document IDs to fetch tags
	documentIDs := make([]string, 0, len(documentsData))
	for _, docData := range documentsData {
		if docID, ok := docData["id"].(string); ok && docID != "" {
			documentIDs = append(documentIDs, docID)
		}
	}

	// Fetch tags for all documents (only first tag per document)
	tagsMap := make(map[string]string)
	if len(documentIDs) > 0 {
		// First, get all document_tag relationships
		docTagsData, _, err := client.From("document_tags").
			Select("document_id,tag_id", "", false).
			In("document_id", documentIDs).
			Execute()
		if err == nil {
			var docTagsList []map[string]interface{}
			if err := json.Unmarshal(docTagsData, &docTagsList); err == nil {
				// Group by document_id to get only first tag per document
				docTagMap := make(map[string]string) // document_id -> tag_id (only first)
				for _, docTagData := range docTagsList {
					docID := getString(docTagData, "document_id")
					tagID := getString(docTagData, "tag_id")
					if docID != "" && tagID != "" {
						// Only store first tag for each document
						if _, exists := docTagMap[docID]; !exists {
							docTagMap[docID] = tagID
						}
					}
				}

				// Get all unique tag IDs
				tagIDs := make([]string, 0, len(docTagMap))
				tagIDSet := make(map[string]bool)
				for _, tagID := range docTagMap {
					if !tagIDSet[tagID] {
						tagIDs = append(tagIDs, tagID)
						tagIDSet[tagID] = true
					}
				}

				// Get tag names from user_tags
				if len(tagIDs) > 0 {
					tagsData, _, err := client.From("user_tags").
						Select("id,name", "", false).
						In("id", tagIDs).
						Execute()
					if err == nil {
						var tagsList []map[string]interface{}
						if err := json.Unmarshal(tagsData, &tagsList); err == nil {
							// Create map of tag_id -> tag_name
							tagNameMap := make(map[string]string)
							for _, tagData := range tagsList {
								tagID := getString(tagData, "id")
								tagName := getString(tagData, "name")
								if tagID != "" && tagName != "" {
									tagNameMap[tagID] = tagName
								}
							}

							// Map document_id -> tag_name
							for docID, tagID := range docTagMap {
								if tagName, exists := tagNameMap[tagID]; exists {
									tagsMap[docID] = tagName
								}
							}
						}
					}
				}
			}
		} else {
			r.logger.Warn("Failed to fetch tags for documents", "error", err)
		}
	}

	var documents []*domain.Document
	for _, docData := range documentsData {
		// Set content to empty JSON array since we didn't fetch it
		docData["content"] = json.RawMessage("[]")

		// Add favorite flag.
		if favErr == nil {
			if docID, ok := docData["id"].(string); ok && docID != "" {
				if favIDs[docID] {
					docData["is_favorite"] = true
				}
			}
		}

		// Add tag to document data (single tag only)
		if docID, ok := docData["id"].(string); ok {
			if tag, exists := tagsMap[docID]; exists {
				docData["tag"] = tag
			}
		}

		doc, err := r.mapToDocument(docData)
		if err != nil {
			r.logger.Error("Failed to map document", err, "doc_id", docData["id"])
			continue
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// SetFavorite inserts/deletes the favorite relationship for a (user, document).
func (r *DocumentRepository) SetFavorite(userID string, documentID string, isFavorite bool, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	if isFavorite {
		row := map[string]interface{}{
			"user_id":     userID,
			"document_id": documentID,
		}
		// Insert is idempotent due to PK (user_id, document_id). If it already exists, Supabase may return 409.
		// We treat 409 as success; supabase-go doesn't expose status cleanly here, so we just attempt insert and
		// ignore "duplicate key" style errors.
		_, _, err = client.From("document_favorites").Insert(row, false, "", "", "").Execute()
		if err != nil {
			// Best-effort duplicate detection.
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
				strings.Contains(strings.ToLower(err.Error()), "already exists") {
				return nil
			}
			return fmt.Errorf("failed to set favorite: %w", err)
		}
		return nil
	}

	_, _, err = client.From("document_favorites").
		Delete("", "").
		Eq("user_id", userID).
		Eq("document_id", documentID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to unset favorite: %w", err)
	}
	return nil
}

// Update a document in Supabase
func (r *DocumentRepository) Update(document *domain.Document, token string) error {
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

	// Optional optimized_content (JSONB)
	if len(document.OptimizedContent) > 0 {
		var optimizedData interface{}
		if err := json.Unmarshal(document.OptimizedContent, &optimizedData); err != nil {
			r.logger.Warn("Failed to unmarshal optimized_content JSON in update", "error", err)
		} else {
			data["optimized_content"] = optimizedData
		}
	}
	if document.OptimizedVersion > 0 {
		data["optimized_version"] = document.OptimizedVersion
	}
	if document.OptimizedSizeBytes != nil {
		data["optimized_size_bytes"] = *document.OptimizedSizeBytes
	}
	if document.OptimizedChecksumSHA256 != nil {
		data["optimized_checksum_sha256"] = *document.OptimizedChecksumSHA256
	}
	if document.ProcessingStatus != "" {
		data["processing_status"] = document.ProcessingStatus
	}
	if document.ProcessingError != nil {
		data["processing_error"] = *document.ProcessingError
	}
	if document.LanguageCode != nil {
		data["language_code"] = *document.LanguageCode
	}
	if document.ProcessedAt != nil {
		data["processed_at"] = *document.ProcessedAt
	}

	// Original file columns (set when provided)
	if document.OriginalStoragePath != nil {
		data["original_storage_path"] = *document.OriginalStoragePath
	}
	if document.OriginalFileName != nil {
		data["original_file_name"] = *document.OriginalFileName
	}
	if document.OriginalMimeType != nil {
		data["original_mime_type"] = *document.OriginalMimeType
	}
	if document.OriginalSizeBytes != nil {
		data["original_size_bytes"] = *document.OriginalSizeBytes
	}
	if document.OriginalChecksumSHA256 != nil {
		data["original_checksum_sha256"] = *document.OriginalChecksumSHA256
	}

	// Add optional fields if they exist
	if document.Author != nil {
		data["author"] = *document.Author
	} else {
		data["author"] = nil
	}
	if document.Description != nil {
		data["description"] = *document.Description
	} else {
		data["description"] = nil
	}

	_, _, err = client.From("documents").
		Update(data, "", "").
		Eq("id", document.ID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	// Update tag relationship in document_tags table
	// First, get user_id to find the tag
	userID := document.UserID
	if userID == "" {
		// Try to get user_id from the document if not set
		docData, _, err := client.From("documents").
			Select("user_id", "", false).
			Eq("id", document.ID).
			Execute()
		if err == nil {
			var docs []map[string]interface{}
			if err := json.Unmarshal(docData, &docs); err == nil && len(docs) > 0 {
				userID = getString(docs[0], "user_id")
			}
		}
	}

	if userID != "" {
		// Delete existing tag relationships for this document
		_, _, err = client.From("document_tags").
			Delete("", "").
			Eq("document_id", document.ID).
			Execute()
		if err != nil {
			r.logger.Warn("Failed to delete existing document tags", "error", err, "document_id", document.ID)
		}

		// If tag is provided, create new relationship
		if document.Tag != nil && *document.Tag != "" {
			// Find the tag_id from user_tags table
			tagData, _, err := client.From("user_tags").
				Select("id", "", false).
				Eq("user_id", userID).
				Eq("name", *document.Tag).
				Execute()
			if err == nil {
				var tags []map[string]interface{}
				if err := json.Unmarshal(tagData, &tags); err == nil && len(tags) > 0 {
					tagID := getString(tags[0], "id")
					if tagID != "" {
						// Create new relationship
						docTagData := map[string]interface{}{
							"document_id": document.ID,
							"tag_id":      tagID,
						}
						_, _, err = client.From("document_tags").
							Insert(docTagData, false, "", "", "").
							Execute()
						if err != nil {
							r.logger.Warn("Failed to create document tag relationship", "error", err, "document_id", document.ID, "tag", *document.Tag)
						}
					}
				}
			} else {
				r.logger.Warn("Failed to find tag", "error", err, "tag_name", *document.Tag, "user_id", userID)
			}
		}
	}

	return nil
}

// Delete deletes a document from Supabase
func (r *DocumentRepository) Delete(id string, token string) error {
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
func (r *DocumentRepository) Search(userID, query string, token string) ([]*domain.Document, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	// Get all user documents first (Supabase doesn't have full-text search by default)
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

	// Filter documents by query (case-insensitive search in title and content)
	queryLower := strings.ToLower(query)
	var documents []*domain.Document
	for _, docData := range documentsData {
		doc, err := r.mapToDocument(docData)
		if err != nil {
			r.logger.Error("Failed to map document", err, "doc_id", docData["id"])
			continue
		}

		// Search in title
		titleMatch := strings.Contains(strings.ToLower(doc.Title), queryLower)

		// Search in content (first 1000 chars for performance)
		contentMatch := false
		if len(doc.Content) > 0 {
			contentStr := strings.ToLower(string(doc.Content))
			if len(contentStr) > 1000 {
				contentStr = contentStr[:1000]
			}
			contentMatch = strings.Contains(contentStr, queryLower)
		}

		// Search in author
		authorMatch := false
		if doc.Author != nil {
			authorMatch = strings.Contains(strings.ToLower(*doc.Author), queryLower)
		}

		if titleMatch || contentMatch || authorMatch {
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// mapToDocument converts a map to a Document struct
func (r *DocumentRepository) mapToDocument(data map[string]interface{}) (*domain.Document, error) {
	document := &domain.Document{
		ID:          getString(data, "id"),
		UserID:      getString(data, "user_id"),
		Title:       getString(data, "title"),
		Author:      getStringPointer(data, "author"),
		Description: getStringPointer(data, "description"),
	}

	// Offline-first columns (optional)
	document.OriginalStoragePath = getStringPointer(data, "original_storage_path")
	document.OriginalFileName = getStringPointer(data, "original_file_name")
	document.OriginalMimeType = getStringPointer(data, "original_mime_type")
	if v := getInt64(data, "original_size_bytes"); v > 0 {
		document.OriginalSizeBytes = &v
	}
	document.OriginalChecksumSHA256 = getStringPointer(data, "original_checksum_sha256")

	if val, ok := data["optimized_content"]; ok && val != nil {
		if s, ok := val.(string); ok {
			document.OptimizedContent = json.RawMessage(s)
		} else {
			b, err := json.Marshal(val)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal optimized_content: %w", err)
			}
			document.OptimizedContent = json.RawMessage(b)
		}
	}

	// Defaults for existing rows
	document.OptimizedVersion = getInt(data, "optimized_version")
	if document.OptimizedVersion <= 0 {
		document.OptimizedVersion = 1
	}
	if v := getInt64(data, "optimized_size_bytes"); v > 0 {
		document.OptimizedSizeBytes = &v
	}
	document.OptimizedChecksumSHA256 = getStringPointer(data, "optimized_checksum_sha256")

	document.ProcessingStatus = getString(data, "processing_status")
	if document.ProcessingStatus == "" {
		document.ProcessingStatus = "ready"
	}
	document.ProcessingError = getStringPointer(data, "processing_error")
	document.LanguageCode = getStringPointer(data, "language_code")
	if processedAt := getString(data, "processed_at"); processedAt != "" {
		if t, err := time.Parse(time.RFC3339, processedAt); err == nil {
			document.ProcessedAt = &t
		} else if t, err := time.Parse(time.RFC3339Nano, processedAt); err == nil {
			document.ProcessedAt = &t
		}
	}

	// Favorites (optional field in list endpoints)
	if val, ok := data["is_favorite"]; ok && val != nil {
		switch v := val.(type) {
		case bool:
			document.IsFavorite = v
		case string:
			document.IsFavorite = strings.EqualFold(v, "true")
		case float64:
			document.IsFavorite = v != 0
		default:
			document.IsFavorite = false
		}
	}

	// Parse timestamps
	if createdAt := getString(data, "created_at"); createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			document.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
			document.CreatedAt = t
		}
	}
	if updatedAt := getString(data, "updated_at"); updatedAt != "" {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			document.UpdatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
			document.UpdatedAt = t
		}
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

	// Parse tag (single tag only)
	if tagVal, ok := data["tag"]; ok && tagVal != nil {
		if tagStr, ok := tagVal.(string); ok && tagStr != "" {
			document.Tag = &tagStr
		}
	}

	return document, nil
}

// favoriteIDsByUser returns a set of document_id that are favorited by user.
func (r *DocumentRepository) favoriteIDsByUser(userID string, token string) (map[string]bool, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("document_favorites").
		Select("document_id", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}

	set := make(map[string]bool, len(rows))
	for _, row := range rows {
		if id := getString(row, "document_id"); id != "" {
			set[id] = true
		}
	}
	return set, nil
}

func (r *DocumentRepository) isFavorite(userID string, documentID string, token string) (bool, error) {
	set, err := r.favoriteIDsByUser(userID, token)
	if err != nil {
		return false, err
	}
	return set[documentID], nil
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

func getStringPointer(data map[string]interface{}, key string) *string {
	if val, ok := data[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			if str == "" {
				return nil
			}
			return &str
		}
	}
	return nil
}

// getStringArray is defined in user_preference_repository.go and shared across the package

// GetTagsByUserID retrieves all tags for a user from the user_tags table
func (r *DocumentRepository) GetTagsByUserID(userID string, token string) ([]string, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	// Fetch tags from user_tags table
	tagsData, _, err := client.From("user_tags").
		Select("name", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	var tagsList []map[string]interface{}
	if err := json.Unmarshal(tagsData, &tagsList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	tags := make([]string, 0, len(tagsList))
	for _, tagData := range tagsList {
		if tagName := getString(tagData, "name"); tagName != "" {
			tags = append(tags, tagName)
		}
	}

	return tags, nil
}

// CreateTag creates a new tag for a user in the user_tags table
func (r *DocumentRepository) CreateTag(userID string, tagName string, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	// Check if tag already exists for this user
	existingTagsData, _, err := client.From("user_tags").
		Select("id", "", false).
		Eq("user_id", userID).
		Eq("name", tagName).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to check existing tag: %w", err)
	}

	var existingTags []map[string]interface{}
	if err := json.Unmarshal(existingTagsData, &existingTags); err != nil {
		return fmt.Errorf("failed to unmarshal existing tags: %w", err)
	}

	if len(existingTags) > 0 {
		return fmt.Errorf("tag already exists")
	}

	// Create new tag
	tagData := map[string]interface{}{
		"user_id": userID,
		"name":    tagName,
	}

	_, _, err = client.From("user_tags").
		Insert(tagData, false, "", "", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	r.logger.Info("Tag created successfully", "user_id", userID, "tag_name", tagName)
	return nil
}

// DeleteTag deletes a tag for a user from the user_tags table
func (r *DocumentRepository) DeleteTag(userID string, tagName string, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	// First, find the tag ID
	tagData, _, err := client.From("user_tags").
		Select("id", "", false).
		Eq("user_id", userID).
		Eq("name", tagName).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to find tag: %w", err)
	}

	var tags []map[string]interface{}
	if err := json.Unmarshal(tagData, &tags); err != nil {
		return fmt.Errorf("failed to unmarshal tag data: %w", err)
	}

	if len(tags) == 0 {
		return fmt.Errorf("tag not found")
	}

	tagID := getString(tags[0], "id")
	if tagID == "" {
		return fmt.Errorf("tag ID not found")
	}

	// Delete all document_tag relationships first (CASCADE should handle this, but being explicit)
	_, _, err = client.From("document_tags").
		Delete("", "").
		Eq("tag_id", tagID).
		Execute()
	if err != nil {
		r.logger.Warn("Failed to delete document_tag relationships", "error", err, "tag_id", tagID)
		// Continue anyway, CASCADE should handle it
	}

	// Delete the tag
	_, _, err = client.From("user_tags").
		Delete("", "").
		Eq("id", tagID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	r.logger.Info("Tag deleted successfully", "user_id", userID, "tag_name", tagName)
	return nil
}
