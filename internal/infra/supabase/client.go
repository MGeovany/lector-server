package supabase

import (
	"fmt"

	"pdf-text-reader/internal/domain"

	"github.com/supabase-community/supabase-go"
)

// SupabaseClient implements the domain.SupabaseClient interface
type SupabaseClient struct {
	client *supabase.Client
	config domain.Config
	logger domain.Logger
}

// SupabaseUser represents a user from Supabase Auth
type SupabaseUser struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// NewSupabaseClient creates a new Supabase client instance
func NewSupabaseClient(config domain.Config, logger domain.Logger) domain.SupabaseClient {
	return &SupabaseClient{
		config: config,
		logger: logger,
	}
}

func (s *SupabaseClient) DB() *supabase.Client {
	return s.client
}

// Initialize establishes a connection to Supabase
func (s *SupabaseClient) Initialize() error {
	supabaseURL := s.config.GetSupabaseURL()
	supabaseKey := s.config.GetSupabaseKey()

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("supabase URL and key must be provided")
	}

	client, err := supabase.NewClient(supabaseURL, supabaseKey, &supabase.ClientOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Supabase client: %w", err)
	}

	s.client = client
	s.logger.Info("Supabase client initialized successfully", "url", supabaseURL)
	return nil
}

// GetClient returns the underlying Supabase client
func (s *SupabaseClient) GetClient() interface{} {
	return s.client
}

// GetSupabaseClient returns the typed Supabase client for repository use
func (s *SupabaseClient) GetSupabaseClient() *supabase.Client {
	return s.client
}

// ValidateToken validates a Supabase JWT token and returns user info
func (s *SupabaseClient) ValidateToken(token string) (*domain.SupabaseUser, error) {
	if s.client == nil {
		return nil, fmt.Errorf("Supabase client not initialized")
	}

	// Get user info using an auth client with the access token.
	// Note: passing "Authorization" via Supabase client headers does not affect GoTrue requests.
	user, err := s.client.Auth.WithToken(token).GetUser()
	if err != nil {
		s.logger.Error("Failed to validate token with Supabase", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Convert Supabase user to domain user
	domainUser := &domain.SupabaseUser{
		ID:           user.ID.String(),
		Email:        user.Email,
		UserMetadata: user.UserMetadata,
		CreatedAt:    user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return domainUser, nil
}
