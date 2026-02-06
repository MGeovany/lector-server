// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	documentService   domain.DocumentService
	preferenceService domain.UserPreferencesService
	logger            domain.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(documentService domain.DocumentService, preferenceService domain.UserPreferencesService, logger domain.Logger) *DocumentHandler {
	return &DocumentHandler{
		documentService:   documentService,
		preferenceService: preferenceService,
		logger:            logger,
	}
}

// Get Documents by User ID

func (h *DocumentHandler) GetDocumentsByUserID(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		h.writeError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	// Fetch documents and reading positions in parallel.
	documentsChan := make(chan []*domain.DocumentData, 1)
	positionsChan := make(chan map[string]*domain.ReadingPosition, 1)
	errChan := make(chan error, 2)

	go func() {
		docs, err := h.documentService.GetDocumentsByUserID(userID, token)
		if err != nil {
			errChan <- err
			return
		}
		documentsChan <- docs
	}()

	go func() {
		positions, err := h.preferenceService.GetAllReadingPositions(userID, token)
		if err != nil {
			// Non-blocking: return empty map if positions fail.
			positionsChan <- make(map[string]*domain.ReadingPosition)
			return
		}
		positionsChan <- positions
	}()

	var documents []*domain.DocumentData
	var positions map[string]*domain.ReadingPosition
	received := 0
	var firstErr error

	for received < 2 {
		select {
		case err := <-errChan:
			if err != nil {
				firstErr = err
			}
			received++
		case docs := <-documentsChan:
			documents = docs
			received++
		case pos := <-positionsChan:
			positions = pos
			received++
		}
	}

	if firstErr != nil {
		h.writeError(w, http.StatusInternalServerError, firstErr.Error())
		return
	}

	// Ensure JSON is [] not null when there are no documents.
	if documents == nil {
		documents = make([]*domain.DocumentData, 0)
	}

	// Attach reading_position onto each document (inline) to avoid extra fetches on clients.
	if positions != nil {
		for _, doc := range documents {
			if doc == nil {
				continue
			}
			if pos, ok := positions[doc.ID]; ok {
				doc.ReadingPosition = pos
			}
		}
	}

	h.writeJSON(w, http.StatusOK, documents)
}

// GetLibrary handles getting the complete library data (documents + positions)
// DEPRECATED: Use getDocumentsByUserID instead
func (h *DocumentHandler) GetLibrary(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	// Get documents and positions in parallel
	documentsChan := make(chan []*domain.Document, 1)
	positionsChan := make(chan map[string]*domain.ReadingPosition, 1)
	errChan := make(chan error, 2)

	go func() {
		docs, err := h.documentService.GetDocumentsByUserID(user.ID, token)
		if err != nil {
			errChan <- err
			return
		}
		documentsChan <- docs
	}()

	go func() {
		positions, err := h.preferenceService.GetAllReadingPositions(user.ID, token)
		if err != nil {
			// If positions fail, return empty map (not critical)
			positionsChan <- make(map[string]*domain.ReadingPosition)
			return
		}
		positionsChan <- positions
	}()

	// Wait for all results
	var documents []*domain.Document
	var positions map[string]*domain.ReadingPosition
	received := 0
	var firstErr error

	for received < 2 {
		select {
		case err := <-errChan:
			if err != nil {
				firstErr = err
			}
			received++
		case docs := <-documentsChan:
			documents = docs
			received++
		case pos := <-positionsChan:
			positions = pos
			received++
		}
	}

	if firstErr != nil {
		h.logger.Error("Failed to load library data", firstErr, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to load library data")
		return
	}

	// Combine documents with positions
	documentsWithPositions := make([]domain.DocumentWithPosition, 0, len(documents))
	for _, doc := range documents {
		docWithPos := domain.DocumentWithPosition{
			DocumentData: doc,
		}
		if pos, ok := positions[doc.ID]; ok {
			docWithPos.ReadingPosition = pos
		}
		documentsWithPositions = append(documentsWithPositions, docWithPos)
	}

	response := domain.LibraryResponse{
		Documents: documentsWithPositions,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// UploadDocument handles document upload
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {

	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Validate file is present
	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, 400, "File is required")
		return
	}
	defer file.Close()

	// Sanitize filename (strip any path components)
	originalName := strings.TrimSpace(filepath.Base(header.Filename))
	if originalName == "" || originalName == "." || originalName == string(filepath.Separator) {
		originalName = "document"
	}

	// Validate extension (strict allow-list)
	ext := strings.ToLower(filepath.Ext(originalName))
	allowedExt := map[string]bool{
		".pdf":  true,
		".epub": true,
		".txt":  true,
		".md":   true,
	}
	if ext == "" || !allowedExt[ext] {
		h.writeError(w, http.StatusBadRequest, "Unsupported file type. Allowed: PDF (.pdf), EPUB (.epub), TXT (.txt), Markdown (.md).")
		return
	}

	// Validate file size
	if header.Size > 15<<20 { // 15MB single file limit
		h.writeError(w, http.StatusBadRequest, "File too large. Maximum single file size is 15MB.")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	doc, err := h.documentService.Upload(
		r.Context(),
		user.ID,
		file,
		token,
		originalName,
	)
	if err != nil {
		// If the error message mentions storage limit, return 400 with friendly text
		if strings.Contains(err.Error(), "storage limit exceeded") {
			h.writeError(w, http.StatusBadRequest, "Storage limit reached. Please delete some documents or contact support to increase your storage.")
			return
		}
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Clean the document content before returning to avoid JSON serialization errors
	// The document is already saved in the database, we just need to return a safe version
	cleanDoc := h.cleanDocumentForResponse(doc)
	h.writeJSON(w, 201, cleanDoc)
}

// GetStorageUsage returns current storage usage and limit for authenticated user.
func (h *DocumentHandler) GetStorageUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	prefs, err := h.preferenceService.GetPreferences(user.ID, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve preferences")
		return
	}

	limit := prefs.StorageLimitBytes
	if limit <= 0 {
		limit = domain.StorageLimitBytesForPlan(prefs.SubscriptionPlan)
	}

	docs, err := h.documentService.GetDocumentsByUserID(user.ID, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve documents")
		return
	}

	var used int64
	for _, d := range docs {
		if d == nil {
			continue
		}
		used += d.Metadata.FileSize
	}

	type resp struct {
		UsedBytes  int64   `json:"used_bytes"`
		LimitBytes int64   `json:"limit_bytes"`
		Percent    float64 `json:"percent"`
	}

	percent := 0.0
	if limit > 0 {
		percent = float64(used) / float64(limit)
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
	}

	h.writeJSON(w, http.StatusOK, resp{
		UsedBytes:  used,
		LimitBytes: limit,
		Percent:    percent,
	})
}

// GetDocument handles getting a specific document
func (h *DocumentHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	documentID := vars["id"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	document, err := h.documentService.GetDocument(documentID, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify the document belongs to the user
	if document.UserID != user.ID {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Clean the document content before returning
	cleanDoc := h.cleanDocumentForResponse(document)
	h.writeJSON(w, http.StatusOK, cleanDoc)
}

// GetOptimizedDocument returns the lightweight page array used by offline-first clients.
// - 200: processing_status=ready + pages
// - 202: not ready yet
// - 304: If-None-Match matches optimized checksum
func (h *DocumentHandler) GetOptimizedDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	documentID := vars["id"]
	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	includePages := true
	if raw := strings.TrimSpace(r.URL.Query().Get("include_pages")); raw != "" {
		if raw == "0" || strings.EqualFold(raw, "false") {
			includePages = false
		}
	}
	h.logger.Info("GetOptimizedDocument request", "document_id", documentID, "include_pages", includePages)

	var opt *domain.OptimizedDocument
	var err error
	if includePages {
		opt, err = h.documentService.GetOptimizedDocument(documentID, token)
	} else {
		opt, err = h.documentService.GetOptimizedDocumentMeta(documentID, token)
	}
	if err != nil {
		h.logger.Error("GetOptimizedDocument failed", err, "document_id", documentID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	pageCount := 0
	readyCount := 0
	if opt.Pages != nil {
		pageCount = len(opt.Pages)
		for _, p := range opt.Pages {
			if strings.TrimSpace(p) != "" {
				readyCount++
			}
		}
	}
	optSize := int64(0)
	if opt.OptimizedSizeBytes != nil {
		optSize = *opt.OptimizedSizeBytes
	}
	h.logger.Info("GetOptimizedDocument response", "document_id", documentID, "processing_status", opt.ProcessingStatus, "pages_ready", readyCount, "pages_total", pageCount, "optimized_size_bytes", optSize)
	if opt.UserID != "" && opt.UserID != user.ID {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	// ETag support.
	if opt.OptimizedChecksumSHA != nil && *opt.OptimizedChecksumSHA != "" {
		etag := "\"" + *opt.OptimizedChecksumSHA + "\""
		w.Header().Set("ETag", etag)
		if inm := strings.TrimSpace(r.Header.Get("If-None-Match")); inm != "" {
			if inm == etag || strings.Trim(inm, "\"") == *opt.OptimizedChecksumSHA {
				h.logger.Info("GetOptimizedDocument 304 not modified", "document_id", documentID)
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// If we already have some pages, return them even while processing.
	if opt.ProcessingStatus != "ready" {
		hasPages := includePages && len(opt.Pages) > 0
		if hasPages {
			h.logger.Info("GetOptimizedDocument 200 partial", "document_id", documentID, "status", opt.ProcessingStatus)
			h.writeJSON(w, http.StatusOK, opt)
			return
		}
		h.logger.Info("GetOptimizedDocument 202 not ready", "document_id", documentID, "status", opt.ProcessingStatus)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		h.writeJSON(w, http.StatusAccepted, opt)
		return
	}

	// Ensure processed_at is set in response when missing (best-effort).
	if opt.ProcessedAt == nil {
		now := time.Now().UTC()
		opt.ProcessedAt = &now
	}

	h.logger.Info("GetOptimizedDocument 200 ready", "document_id", documentID, "pages_total", pageCount)
	h.writeJSON(w, http.StatusOK, opt)
}

type updateDocumentRequest struct {
	Title  *string `json:"title"`
	Author *string `json:"author"`
	Tag    *string `json:"tag"` // Single tag (document can only have one tag)
}

type setFavoriteRequest struct {
	IsFavorite bool `json:"is_favorite"`
}

// SetFavorite marks/unmarks a document as favorite for the authenticated user.
func (h *DocumentHandler) SetFavorite(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	documentID := vars["id"]
	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	var req setFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.documentService.SetFavorite(user.ID, documentID, req.IsFavorite, token); err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"document_id": documentID,
		"is_favorite": req.IsFavorite,
		"updated":     true,
	})
}

// UpdateDocument updates title/author/tag for a document
func (h *DocumentHandler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	vars := mux.Vars(r)
	documentID := vars["id"]
	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	var req updateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == nil && req.Author == nil && req.Tag == nil {
		h.writeError(w, http.StatusBadRequest, "No updates provided")
		return
	}

	updated, err := h.documentService.UpdateDocumentDetails(user.ID, documentID, req.Title, req.Author, req.Tag, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	cleanDoc := h.cleanDocumentForResponse(updated)
	h.writeJSON(w, http.StatusOK, cleanDoc)
}

// DeleteDocument handles document deletion
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["id"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	err := h.documentService.DeleteDocument(documentID, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, "Document deleted successfully")
}

// SearchDocuments handles document search
func (h *DocumentHandler) SearchDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	documents, err := h.documentService.SearchDocuments(user.ID, query, token)
	if err != nil {
		h.logger.Error("Failed to search documents", err, "user_id", user.ID, "query", query)
		h.writeError(w, http.StatusInternalServerError, "Failed to search documents")
		return
	}

	// Clean documents before returning
	var cleanDocs []*domain.DocumentData
	for _, doc := range documents {
		cleanDocs = append(cleanDocs, h.cleanDocumentForResponse(doc))
	}

	// Ensure JSON is [] not null when empty.
	if cleanDocs == nil {
		cleanDocs = make([]*domain.DocumentData, 0)
	}

	h.writeJSON(w, http.StatusOK, cleanDocs)
}

// GetDocumentTags handles getting all document tags for the authenticated user
func (h *DocumentHandler) GetDocumentTags(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	tags, err := h.documentService.GetDocumentTags(user.ID, token)
	if err != nil {
		h.logger.Error("Failed to get document tags", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to get document tags")
		return
	}

	// Ensure JSON is [] not null when empty.
	if tags == nil {
		tags = make([]string, 0)
	}

	h.writeJSON(w, http.StatusOK, tags)
}

type createTagRequest struct {
	Name string `json:"name"`
}

// CreateTag handles creating a new tag for the authenticated user
func (h *DocumentHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "Tag name is required")
		return
	}

	err := h.documentService.CreateTag(user.ID, req.Name, token)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, "Tag already exists")
			return
		}
		h.logger.Error("Failed to create tag", err, "user_id", user.ID, "tag_name", req.Name)
		h.writeError(w, http.StatusInternalServerError, "Failed to create tag")
		return
	}

	// Return the created tag name
	h.writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name})
}

// DeleteTag handles deleting a tag for the authenticated user
func (h *DocumentHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	vars := mux.Vars(r)
	tagName := vars["name"]
	if tagName == "" {
		h.writeError(w, http.StatusBadRequest, "Tag name is required")
		return
	}

	err := h.documentService.DeleteTag(user.ID, tagName, token)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "Tag not found")
			return
		}
		h.logger.Error("Failed to delete tag", err, "user_id", user.ID, "tag_name", tagName)
		h.writeError(w, http.StatusInternalServerError, "Failed to delete tag")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Tag deleted successfully"})
}

// writeError writes an error response
func (h *DocumentHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// cleanDocumentForResponse ensures the document content is safe for JSON serialization
func (h *DocumentHandler) cleanDocumentForResponse(doc *domain.Document) *domain.Document {
	// Create a copy to avoid modifying the original
	cleanDoc := *doc

	// If content has problematic characters, validate and clean it
	if len(doc.Content) > 0 {
		var contentData interface{}
		if err := json.Unmarshal(doc.Content, &contentData); err != nil {
			// If unmarshaling fails, use empty array
			cleanDoc.Content = json.RawMessage("[]")
		} else {
			// Re-marshal to ensure clean JSON
			cleanedJSON, err := json.Marshal(contentData)
			if err != nil {
				// If marshaling fails, use empty array
				cleanDoc.Content = json.RawMessage("[]")
			} else {
				cleanDoc.Content = json.RawMessage(cleanedJSON)
			}
		}
	} else {
		cleanDoc.Content = json.RawMessage("[]")
	}

	return &cleanDoc
}

// writeJSON writes a JSON response
func (h *DocumentHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
