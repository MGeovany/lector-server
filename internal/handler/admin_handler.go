package handler

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/supabase-community/supabase-go"
)

// AdminHandler exposes admin-only endpoints protected by X-Admin-Secret.
// These endpoints are intended for internal use (support tooling) and should not be exposed publicly without additional safeguards.
type AdminHandler struct {
	clientOnce sync.Once
	client     *supabase.Client
	clientErr  error
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) serviceRoleClient() (*supabase.Client, error) {
	h.clientOnce.Do(func() {
		supabaseURL := os.Getenv("SUPABASE_URL")
		serviceRoleKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
		if supabaseURL == "" || serviceRoleKey == "" {
			h.clientErr = fmt.Errorf("missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY")
			return
		}

		h.client, h.clientErr = supabase.NewClient(supabaseURL, serviceRoleKey, &supabase.ClientOptions{})
	})

	return h.client, h.clientErr
}

type setAccountDisabledRequest struct {
	AccountDisabled bool `json:"account_disabled"`
}

// SetAccountDisabled toggles the `user_preferences.account_disabled` flag for a given user.
//
// Auth: requires `X-Admin-Secret` header matching env `ADMIN_API_SECRET`.
// DB: uses env `SUPABASE_URL` + `SUPABASE_SERVICE_ROLE_KEY` to bypass RLS.
func (h *AdminHandler) SetAccountDisabled(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Admin-Secret")
	expected := os.Getenv("ADMIN_API_SECRET")
	if expected == "" || secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) != 1 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	vars := mux.Vars(r)
	userID := vars["id"]
	if userID == "" {
		writeError(w, http.StatusBadRequest, "User id is required")
		return
	}

	var req setAccountDisabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	client, err := h.serviceRoleClient()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Server misconfigured")
		return
	}

	data := map[string]interface{}{
		"user_id":          userID,
		"account_disabled": req.AccountDisabled,
	}
	_, _, err = client.From("user_preferences").Upsert(data, "", "", "").Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update account status: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":          userID,
		"account_disabled": req.AccountDisabled,
	})
}
