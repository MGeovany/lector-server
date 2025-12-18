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
		container:         container,
		logger:            logger,
		preferenceService: container.PreferenceService,
	}
}

// GetPreferences handles getting user preferences
func (h *PreferenceHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
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

	preferences, err := h.preferenceService.GetPreferences(user.ID, token)
	if err != nil {
		h.logger.Error("Failed to get preferences", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve preferences")
		return
	}

	h.writeJSON(w, http.StatusOK, preferences)
}

// UpdatePreferences handles updating user preferences
func (h *PreferenceHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
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

	// Decode partial preferences (only fields that are sent)
	var prefsUpdate map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&prefsUpdate); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get current preferences first
	currentPrefs, err := h.preferenceService.GetPreferences(user.ID, token)
	if err != nil {
		h.logger.Error("Failed to get current preferences", err, "user_id", user.ID)
		// If no preferences exist, create defaults
		currentPrefs = &domain.UserPreferences{
			UserID:          user.ID,
			FontSize:        16,
			FontFamily:      "system-ui",
			TextColor:       "#000000",
			BackgroundColor: "#ffffff",
			LineHeight:      1.5,
			MaxWidth:        800,
			Theme:           "light",
		}
	}

	// Update only the fields that were sent
	// Handle font_size (can be int or float64 from JSON)
	if fontSizeVal, ok := prefsUpdate["font_size"]; ok {
		switch v := fontSizeVal.(type) {
		case float64:
			currentPrefs.FontSize = int(v)
		case int:
			currentPrefs.FontSize = v
		case int64:
			currentPrefs.FontSize = int(v)
		}
	}
	
	// Handle font_family
	if fontFamily, ok := prefsUpdate["font_family"].(string); ok {
		currentPrefs.FontFamily = fontFamily
	}
	
	// Handle theme
	if theme, ok := prefsUpdate["theme"].(string); ok {
		currentPrefs.Theme = theme
	}
	
	// Handle line_height (can be int or float64 from JSON)
	if lineHeightVal, ok := prefsUpdate["line_height"]; ok {
		switch v := lineHeightVal.(type) {
		case float64:
			currentPrefs.LineHeight = v
		case int:
			currentPrefs.LineHeight = float64(v)
		case int64:
			currentPrefs.LineHeight = float64(v)
		}
	}
	
	// Handle text_color
	if textColor, ok := prefsUpdate["text_color"].(string); ok {
		currentPrefs.TextColor = textColor
	}
	
	// Handle background_color
	if backgroundColor, ok := prefsUpdate["background_color"].(string); ok {
		currentPrefs.BackgroundColor = backgroundColor
	}
	
	// Handle max_width (can be int or float64 from JSON)
	if maxWidthVal, ok := prefsUpdate["max_width"]; ok {
		switch v := maxWidthVal.(type) {
		case float64:
			currentPrefs.MaxWidth = int(v)
		case int:
			currentPrefs.MaxWidth = v
		case int64:
			currentPrefs.MaxWidth = int(v)
		}
	}

	if err := h.preferenceService.UpdatePreferences(user.ID, currentPrefs, token); err != nil {
		h.logger.Error("Failed to update preferences", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get updated preferences to return
	updatedPrefs, err := h.preferenceService.GetPreferences(user.ID, token)
	if err != nil {
		h.logger.Error("Failed to get updated preferences", err, "user_id", user.ID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve updated preferences")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedPrefs)
}

// GetReadingPosition handles getting reading position for a document
func (h *PreferenceHandler) GetReadingPosition(w http.ResponseWriter, r *http.Request) {
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
	documentID := vars["documentId"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	position, err := h.preferenceService.GetReadingPosition(user.ID, documentID, token)
	if err != nil {
		h.logger.Error("Failed to get reading position", err, "user_id", user.ID, "document_id", documentID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve reading position")
		return
	}

	h.writeJSON(w, http.StatusOK, position)
}

// UpdateReadingPosition handles updating reading position for a document
func (h *PreferenceHandler) UpdateReadingPosition(w http.ResponseWriter, r *http.Request) {
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
	documentID := vars["documentId"]

	if documentID == "" {
		h.writeError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	var position domain.ReadingPosition
	if err := json.NewDecoder(r.Body).Decode(&position); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.preferenceService.UpdateReadingPosition(user.ID, documentID, &position, token); err != nil {
		h.logger.Error("Failed to update reading position", err, "user_id", user.ID, "document_id", documentID)
		h.writeError(w, http.StatusInternalServerError, "Failed to update reading position")
		return
	}

	// Get updated position to return
	updatedPosition, err := h.preferenceService.GetReadingPosition(user.ID, documentID, token)
	if err != nil {
		h.logger.Error("Failed to get updated reading position", err, "user_id", user.ID, "document_id", documentID)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve updated reading position")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedPosition)
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
