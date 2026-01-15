package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"pdf-text-reader/internal/domain"
)

const accountDisabledCacheTTL = 30 * time.Second

type accountDisabledCacheEntry struct {
	disabled  bool
	expiresAt time.Time
}

type authService struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger

	accountDisabledCacheMu sync.RWMutex
	accountDisabledCache   map[string]accountDisabledCacheEntry
}

func NewAuthService(
	supabaseClient domain.SupabaseClient,
	logger domain.Logger,
) *authService {
	return &authService{
		supabaseClient:       supabaseClient,
		logger:               logger,
		accountDisabledCache: make(map[string]accountDisabledCacheEntry),
	}

}

// ValidateToken validates a token and returns user info (for frontend validation)
func (s *authService) ValidateToken(token string) (*domain.SupabaseUser, error) {
	user, err := s.supabaseClient.ValidateToken(token)
	if err != nil {
		s.logger.Error("Failed to validate token with Supabase", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return user, nil
}

// IsAccountDisabled checks the persisted flag in `user_preferences.account_disabled`.
// If the user has no preferences row yet, it defaults to false.
func (s *authService) IsAccountDisabled(userID string, token string) (bool, error) {
	now := time.Now()
	s.accountDisabledCacheMu.RLock()
	entry, ok := s.accountDisabledCache[userID]
	s.accountDisabledCacheMu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return entry.disabled, nil
	}

	client, err := s.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return false, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return false, fmt.Errorf("supabase client not initialized")
	}

	data, _, err := client.From("user_preferences").
		Select("account_disabled", "", false).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get account status: %w", err)
	}

	disabled := false
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(rows) > 0 {
		val, ok := rows[0]["account_disabled"]
		if ok && val != nil {
			switch v := val.(type) {
			case bool:
				disabled = v
			case string:
				disabled = v == "true" || v == "1"
			case float64:
				disabled = v != 0
			}
		}
	}

	s.accountDisabledCacheMu.Lock()
	s.accountDisabledCache[userID] = accountDisabledCacheEntry{disabled: disabled, expiresAt: now.Add(accountDisabledCacheTTL)}
	s.accountDisabledCacheMu.Unlock()

	return disabled, nil
}
