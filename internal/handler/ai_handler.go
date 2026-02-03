package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

type AIHandler struct {
	aiService domain.AIService
	logger    domain.Logger
}

func NewAIHandler(aiService domain.AIService, logger domain.Logger) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		logger:    logger,
	}
}

// Ingest handles document ingestion (embedding generation)
func (h *AIHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	if h.aiService == nil || (interface{})(h.aiService) == nil {
		writeError(w, http.StatusServiceUnavailable, "AI Service not configured (missing GCP_GCP_PROJECT_ID or credentials)")
		return
	}

	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	vars := mux.Vars(r)
	documentID := vars["id"]
	if documentID == "" {
		writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	// Async potential? For now synchronous or return 202
	if err := h.aiService.IngestDocument(r.Context(), user.ID, documentID, token); err != nil {
		h.logger.Error("Ingestion failed", err, "doc_id", documentID)
		writeError(w, http.StatusInternalServerError, "Failed to ingest document")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ingested"})
}

// Ask handles chat queries
func (h *AIHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if h.aiService == nil || (interface{})(h.aiService) == nil {
		writeError(w, http.StatusServiceUnavailable, "AI Service not configured (missing GCP_GCP_PROJECT_ID or credentials)")
		return
	}

	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.DocumentID == "" {
		writeError(w, http.StatusBadRequest, "document_id is required")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt cannot be empty")
		return
	}
	const maxPromptLen = 2000
	if len(req.Prompt) > maxPromptLen {
		writeError(w, http.StatusBadRequest, "prompt too long")
		return
	}

	resp, err := h.aiService.Ask(r.Context(), user.ID, req, token)
	if err != nil {
		h.logger.Error("Ask AI failed", err, "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "Failed to process query")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetChatHistory handles retrieving chat history
func (h *AIHandler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	if h.aiService == nil || (interface{})(h.aiService) == nil {
		writeError(w, http.StatusServiceUnavailable, "AI Service not configured (missing GCP_GCP_PROJECT_ID or credentials)")
		return
	}

	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	token, ok := GetTokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Token not found in context")
		return
	}

	vars := mux.Vars(r)
	sessionID := vars["id"]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	history, err := h.aiService.GetChatHistory(r.Context(), user.ID, sessionID, token)
	if err != nil {
		h.logger.Error("GetChatHistory failed", err, "session_id", sessionID)
		writeError(w, http.StatusInternalServerError, "Failed to retrieve history")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(history)
}
