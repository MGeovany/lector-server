package handler

import (
	"encoding/json"
	"net/http"

	"pdf-text-reader/internal/config"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	container *config.Container
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(container *config.Container) *AuthHandler {
	return &AuthHandler{
		container: container,
	}
}

// GetProfile returns the current user's profile information
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

// UpdateProfile updates the current user's profile information
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var updateData struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update user metadata in Supabase
	// Note: This would typically be done through Supabase Admin API
	// For now, we'll return the updated user data
	user.UserMetadata["name"] = updateData.Name

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}
