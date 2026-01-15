package service

import (
	"errors"
	"testing"

	"github.com/supabase-community/supabase-go"
	"pdf-text-reader/internal/domain"
)

// MockSupabaseClient for testing
type MockSupabaseClient struct {
	users map[string]*domain.SupabaseUser
}

func NewMockSupabaseClient() *MockSupabaseClient {
	return &MockSupabaseClient{
		users: make(map[string]*domain.SupabaseUser),
	}
}

func (m *MockSupabaseClient) Initialize() error {
	return nil
}

func (m *MockSupabaseClient) ValidateToken(token string) (*domain.SupabaseUser, error) {
	// Simple mock: if token is "valid-token", return a user
	if token == "valid-token" {
		return &domain.SupabaseUser{
			ID:    "user-123",
			Email: "test@example.com",
		}, nil
	}

	// If token is "invalid-token", return an error
	if token == "invalid-token" {
		return nil, errors.New("invalid token")
	}

	// For any other token, return error
	return nil, errors.New("token validation failed")
}

func (m *MockSupabaseClient) DB() *supabase.Client {
	return nil // Mock implementation
}

func (m *MockSupabaseClient) GetClientWithToken(token string) (*supabase.Client, error) {
	return nil, nil // Mock implementation
}

func TestAuthService_ValidateToken(t *testing.T) {
	client := NewMockSupabaseClient()
	logger := NewMockLogger()

	service := NewAuthService(client, logger)

	// Test valid token
	user, err := service.ValidateToken("valid-token")
	if err != nil {
		t.Errorf("Expected no error for valid token, got %v", err)
	}

	if user.ID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", user.ID)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected user email 'test@example.com', got '%s'", user.Email)
	}

	// Test invalid token
	_, err = service.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	expectedError := "invalid token: invalid token"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}

	// Test empty token
	_, err = service.ValidateToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}
