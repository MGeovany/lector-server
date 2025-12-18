package handler

import (
	"encoding/json"
	"net/http"

	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	documentService domain.DocumentService
	logger          domain.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(documentService domain.DocumentService, logger domain.Logger) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
		logger:          logger,
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

	documents, err := h.documentService.GetDocumentsByUserID(userID, token)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, documents)
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

	// Validate file size
	if header.Size > 10<<20 { // 10MB
		h.writeError(w, 400, "File too large")
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
		header.Filename,
	)
	if err != nil {
		h.writeError(w, 500, err.Error())
		return
	}

	// Clean the document content before returning to avoid JSON serialization errors
	// The document is already saved in the database, we just need to return a safe version
	cleanDoc := h.cleanDocumentForResponse(doc)
	h.writeJSON(w, 201, cleanDoc)
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
	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	// TODO: Implement search documents logic
	h.writeError(w, http.StatusNotImplemented, "Search documents not implemented yet")
}

// writeError writes an error response
func (h *DocumentHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
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
	json.NewEncoder(w).Encode(data)
}
