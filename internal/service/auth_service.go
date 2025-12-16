package service

import (
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
