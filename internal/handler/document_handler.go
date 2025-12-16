package handler

import (
	"encoding/json"
	"fmt"
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

// GetDocuments handles getting user documents
func (h *DocumentHandler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	fmt.Println(user)

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

	doc, err := h.documentService.Upload(
		r.Context(),
		user.ID,
		file,
	)
	if err != nil {
		h.writeError(w, 500, err.Error())
		return
	}

	h.writeJSON(w, 201, doc)
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

	fmt.Println(documentID, user)

	h.writeJSON(w, http.StatusOK, documentID)
}

// DeleteDocument handles document deletion
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["id"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	// TODO: Implement delete document logic
	h.writeError(w, http.StatusNotImplemented, "Delete document not implemented yet")
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

// writeJSON writes a JSON response
func (h *DocumentHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
