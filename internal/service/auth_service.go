package service

import (
	"encoding/json"
	"fmt"
	"pdf-text-reader/internal/domain"
)

type authService struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

func NewAuthService(
	supabaseClient domain.SupabaseClient,
	logger domain.Logger,
) *authService {
	return &authService{
		supabaseClient: supabaseClient,
		logger:         logger,
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

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(rows) == 0 {
		return false, nil
	}

	val, ok := rows[0]["account_disabled"]
	if !ok || val == nil {
		return false, nil
	}
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return v == "true" || v == "1", nil
	case float64:
		return v != 0, nil
	default:
		return false, nil
	}
}
