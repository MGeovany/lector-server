package handler

import (
	"encoding/json"
	"net/http"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/domain"

	"github.com/gorilla/mux"
)

// PreferenceHandler handles preference-related HTTP requests
type PreferenceHandler struct {
	container         *config.Container
	logger            domain.Logger
	preferenceService domain.PreferenceService
}

// NewPreferenceHandler creates a new preference handler
func NewPreferenceHandler(container *config.Container, logger domain.Logger) *PreferenceHandler {
	return &PreferenceHandler{
		container: container,
		logger:    logger,
	}
}

// GetPreferences handles getting user preferences
func (h *PreferenceHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	preferences, err := h.preferenceService.GetPreferences(user.ID)
	if err != nil {
		h.logger.Error("Failed to get preferences", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve preferences")
		return
	}

	h.writeJSON(w, http.StatusOK, preferences)
}

// UpdatePreferences handles updating user preferences
func (h *PreferenceHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update preferences logic
	h.writeError(w, http.StatusNotImplemented, "Update preferences not implemented yet")
}

// GetReadingPosition handles getting reading position for a document
func (h *PreferenceHandler) GetReadingPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	// TODO: Implement get reading position logic
	h.writeError(w, http.StatusNotImplemented, "Get reading position not implemented yet")
}

// UpdateReadingPosition handles updating reading position for a document
func (h *PreferenceHandler) UpdateReadingPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	// TODO: Implement update reading position logic
	h.writeError(w, http.StatusNotImplemented, "Update reading position not implemented yet")
}

// writeError writes an error response
func (h *PreferenceHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeJSON writes a JSON response
func (h *PreferenceHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
