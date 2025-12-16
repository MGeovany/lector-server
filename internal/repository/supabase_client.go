package repository

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

// Initialize establishes a connection to Supabase
func (s *SupabaseClient) Initialize() error {
	supabaseURL := s.config.GetSupabaseURL()
	supabaseKey := s.config.GetSupabaseKey()

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("Supabase URL and key must be provided")
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

	// For now, we'll implement a basic JWT validation
	// In a production environment, you would validate the JWT signature
	// and extract user information from the token claims
	
	// This is a simplified implementation - in production you should:
	// TODO:
	// 1. Validate JWT signature using the JWT secret
	// 2. Extract user ID from token claims
	// 3. Optionally fetch user details from Supabase if needed
	
	// For development, we'll return a mock user
	// This should be replaced with proper JWT validation
	domainUser := &domain.SupabaseUser{
		ID:           "mock-user-id", // Extract from JWT claims
		Email:        "user@example.com", // Extract from JWT claims
		UserMetadata: make(map[string]interface{}),
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
	}

	return domainUser, nil
}
