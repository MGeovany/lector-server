package handler

import (
	"encoding/json"
	"net/http"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	container *config.Container
	logger    domain.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(container *config.Container) *DocumentHandler {
	return &DocumentHandler{
		container: container,
		logger:    container.GetLogger(),
	}
}

// GetDocuments handles getting user documents
func (h *DocumentHandler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	documents, err := h.container.GetDocumentRepository().GetByUserID(user.ID)
	if err != nil {
		h.logger.Error("Failed to get documents", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve documents")
		return
	}

	h.writeJSON(w, http.StatusOK, documents)
}

// UploadDocument handles document upload
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement document upload logic
	h.writeError(w, http.StatusNotImplemented, "Upload document not implemented yet")
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

	document, err := h.container.GetDocumentRepository().GetByID(documentID)
	if err != nil {
		h.logger.Error("Failed to get document", err, "document_id", documentID, "user_id", user.ID)
		h.writeError(w, http.StatusNotFound, "Document not found")
		return
	}

	// Verify the document belongs to the user
	if document.UserID != user.ID {
		h.writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	h.writeJSON(w, http.StatusOK, document)
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
