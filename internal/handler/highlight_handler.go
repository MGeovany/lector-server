package handler

import (
	"encoding/json"
	"net/http"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

// HighlightHandler handles highlight-related HTTP requests.
type HighlightHandler struct {
	container        *config.Container
	logger           domain.Logger
	highlightService domain.HighlightService
}

func NewHighlightHandler(container *config.Container, logger domain.Logger) *HighlightHandler {
	return &HighlightHandler{
		container:        container,
		logger:           logger,
		highlightService: container.HighlightService,
	}
}

type createHighlightRequest struct {
	DocumentID string   `json:"document_id"`
	Quote      string   `json:"quote"`
	PageNumber *int     `json:"page_number,omitempty"`
	Progress   *float32 `json:"progress,omitempty"`
}

// CreateHighlight handles POST /highlights
func (h *HighlightHandler) CreateHighlight(w http.ResponseWriter, r *http.Request) {
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

	var req createHighlightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.DocumentID == "" {
		h.writeError(w, http.StatusBadRequest, "document_id is required")
		return
	}
	if req.Quote == "" {
		h.writeError(w, http.StatusBadRequest, "quote is required")
		return
	}

	created, err := h.highlightService.CreateHighlight(user.ID, &domain.Highlight{
		DocumentID: req.DocumentID,
		Quote:      req.Quote,
		PageNumber: req.PageNumber,
		Progress:   req.Progress,
	}, token)
	if err != nil {
		h.logger.Error("Failed to create highlight", err, "user_id", user.ID, "document_id", req.DocumentID)
		h.writeError(w, http.StatusInternalServerError, "Failed to create highlight")
		return
	}

	h.writeJSON(w, http.StatusCreated, created)
}

// ListHighlights handles GET /highlights?document_id=...
func (h *HighlightHandler) ListHighlights(w http.ResponseWriter, r *http.Request) {
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

	documentID := r.URL.Query().Get("document_id")
	var docPtr *string
	if documentID != "" {
		docPtr = &documentID
	}

	highlights, err := h.highlightService.ListHighlights(user.ID, docPtr, token)
	if err != nil {
		h.logger.Error("Failed to list highlights", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve highlights")
		return
	}
	if highlights == nil {
		highlights = make([]*domain.Highlight, 0)
	}
	h.writeJSON(w, http.StatusOK, highlights)
}

// DeleteHighlight handles DELETE /highlights/{id}
func (h *HighlightHandler) DeleteHighlight(w http.ResponseWriter, r *http.Request) {
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
	highlightID := vars["id"]
	if highlightID == "" {
		h.writeError(w, http.StatusBadRequest, "Highlight ID is required")
		return
	}

	if err := h.highlightService.DeleteHighlight(user.ID, highlightID, token); err != nil {
		h.logger.Error("Failed to delete highlight", err, "user_id", user.ID, "highlight_id", highlightID)
		h.writeError(w, http.StatusInternalServerError, "Failed to delete highlight")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HighlightHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *HighlightHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

