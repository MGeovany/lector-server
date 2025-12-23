package domain

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        string
	Email     string
	Password  string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SupabaseUser represents a user from Supabase Auth
type SupabaseUser struct {
	ID           string
	Email        string
	UserMetadata map[string]interface{}
	CreatedAt    string
	UpdatedAt    string
}

// UserPreferences represents user's reading preferences
type UserPreferences struct {
	UserID     string    `json:"user_id"`
	FontSize   int       `json:"font_size"`
	FontFamily string    `json:"font_family"`
	Theme      string    `json:"theme"`
	Tags       []string  `json:"tags"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UserPreferencesService interface {
	GetPreferences(userID string, token string) (*UserPreferences, error)
	UpdatePreferences(userID string, prefs *UserPreferences, token string) error
	GetReadingPosition(userID, documentID string, token string) (*ReadingPosition, error)
	GetAllReadingPositions(userID string, token string) (map[string]*ReadingPosition, error)
	UpdateReadingPosition(userID, documentID string, position *ReadingPosition, token string) error
}

type UserPreferencesRepository interface {
	GetPreferences(userID string, token string) (*UserPreferences, error)
	UpdatePreferences(prefs *UserPreferences, token string) error
	GetReadingPosition(userID, documentID string, token string) (*ReadingPosition, error)
	GetAllReadingPositions(userID string, token string) (map[string]*ReadingPosition, error)
	UpdateReadingPosition(position *ReadingPosition, token string) error
}
