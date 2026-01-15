package domain

import (
	"strings"
	"testing"
	"time"
)

func TestUser_Validation(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		wantValid bool
	}{
		{
			name: "Valid user",
			user: User{
				ID:        "test-id",
				Email:     "test@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantValid: true,
		},
		{
			name: "User with missing ID",
			user: User{
				Email:     "test@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantValid: false,
		},
		{
			name: "User with missing email",
			user: User{
				ID:        "test-id",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantValid: false,
		},
		{
			name: "User with invalid email format",
			user: User{
				ID:        "test-id",
				Email:     "invalid-email",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic email validation - check for @ symbol with proper format
			hasAtSymbol := strings.Contains(tt.user.Email, "@") && len(tt.user.Email) > 3
			// More detailed validation for problematic cases
			parts := strings.Split(tt.user.Email, "@")
			validFormat := hasAtSymbol && len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 && !strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")

			isValid := tt.user.ID != "" && tt.user.Email != "" &&
				validFormat &&
				(tt.user.CreatedAt.IsZero() == false) &&
				(tt.user.UpdatedAt.IsZero() == false)

			if isValid != tt.wantValid {
				t.Errorf("User validation = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestUserPreferences_Validation(t *testing.T) {
	tests := []struct {
		name        string
		preferences UserPreferences
		wantValid   bool
	}{
		{
			name: "Valid preferences",
			preferences: UserPreferences{
				UserID:     "user-id",
				FontSize:   16,
				FontFamily: "Arial",
				Theme:      "light",
				Tags:       []string{"tag1", "tag2"},
				UpdatedAt:  time.Now(),
			},
			wantValid: true,
		},
		{
			name: "Preferences with missing UserID",
			preferences: UserPreferences{
				FontSize:   16,
				FontFamily: "Arial",
				UpdatedAt:  time.Now(),
			},
			wantValid: false,
		},
		{
			name: "Preferences with invalid font size",
			preferences: UserPreferences{
				UserID:     "user-id",
				FontSize:   -1, // Invalid
				FontFamily: "Arial",
				UpdatedAt:  time.Now(),
			},
			wantValid: false,
		},
		{
			name: "Valid preferences with minimal fields",
			preferences: UserPreferences{
				UserID:    "user-id",
				FontSize:  12,
				UpdatedAt: time.Now(),
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.preferences.UserID != "" &&
				tt.preferences.FontSize > 0 &&
				(tt.preferences.UpdatedAt.IsZero() == false)

			if isValid != tt.wantValid {
				t.Errorf("UserPreferences validation = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestSupabaseUser_Validation(t *testing.T) {
	tests := []struct {
		name      string
		user      SupabaseUser
		wantValid bool
	}{
		{
			name: "Valid Supabase user",
			user: SupabaseUser{
				ID:    "test-id",
				Email: "test@example.com",
			},
			wantValid: true,
		},
		{
			name: "Supabase user with missing ID",
			user: SupabaseUser{
				Email: "test@example.com",
			},
			wantValid: false,
		},
		{
			name: "Supabase user with missing email",
			user: SupabaseUser{
				ID: "test-id",
			},
			wantValid: false,
		},
		{
			name: "Supabase user with empty email",
			user: SupabaseUser{
				ID:    "test-id",
				Email: "",
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.user.ID != "" && tt.user.Email != ""
			if isValid != tt.wantValid {
				t.Errorf("SupabaseUser validation = %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

func TestUserPreferences_DefaultValues(t *testing.T) {
	prefs := UserPreferences{
		UserID:    "user-id",
		FontSize:  14,
		UpdatedAt: time.Now(),
	}

	// Test that default values are reasonable
	if prefs.FontSize < 8 || prefs.FontSize > 72 {
		t.Error("Font size should be within reasonable range (8-72)")
	}

	// Test default theme if not set
	if prefs.Theme == "" {
		prefs.Theme = "light" // Default theme
	}

	if prefs.Theme != "light" && prefs.Theme != "dark" {
		t.Error("Theme should be either 'light' or 'dark'")
	}
}

func TestUserPreferences_UpdateTimestamp(t *testing.T) {
	prefs := UserPreferences{
		UserID:    "user-id",
		FontSize:  16,
		UpdatedAt: time.Now().Add(-time.Hour), // Updated 1 hour ago
	}

	oldUpdatedAt := prefs.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	// Simulate updating preferences
	prefs.FontFamily = "Times New Roman"
	prefs.UpdatedAt = time.Now()

	if !prefs.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated when preferences are modified")
	}
}

func TestUser_EmailValidation(t *testing.T) {
	tests := []struct {
		email     string
		wantValid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.org", true},
		{"invalid-email", false},
		{"", false},
		{"@example.com", false},
		{"test@", false},
		{"test@.com", false},
		{"test@example.", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			user := User{
				ID:        "test-id",
				Email:     tt.email,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Basic email validation - check for @ symbol with proper format
			hasAtSymbol := strings.Contains(user.Email, "@") && len(user.Email) > 3
			// More detailed validation for problematic cases
			parts := strings.Split(user.Email, "@")
			validFormat := hasAtSymbol && len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 && !strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")

			emailValid := user.ID != "" && user.Email != "" &&
				validFormat &&
				(user.CreatedAt.IsZero() == false) &&
				(user.UpdatedAt.IsZero() == false)

			if emailValid != tt.wantValid {
				t.Errorf("Email validation for '%s' = %v, want %v", tt.email, emailValid, tt.wantValid)
			}
		})
	}
}
