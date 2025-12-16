package domain

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Document represents a PDF document in the user's library
type Document struct {
	ID           string           `json:"id" db:"id"`
	UserID       string           `json:"user_id" db:"user_id"`
	OriginalName string           `json:"original_name" db:"original_name"`
	Title        string           `json:"title" db:"title"`
	Content      []TextBlock      `json:"content" db:"content"`
	Metadata     DocumentMetadata `json:"metadata" db:"metadata"`
	FilePath     string           `json:"-" db:"file_path"`
	FileSize     int64            `json:"file_size" db:"file_size"`
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at" db:"updated_at"`
}

// UserPreferences represents user's reading preferences
type UserPreferences struct {
	UserID          string    `json:"user_id" db:"user_id"`
	FontSize        int       `json:"font_size" db:"font_size"`
	FontFamily      string    `json:"font_family" db:"font_family"`
	TextColor       string    `json:"text_color" db:"text_color"`
	BackgroundColor string    `json:"background_color" db:"background_color"`
	LineHeight      float64   `json:"line_height" db:"line_height"`
	MaxWidth        int       `json:"max_width" db:"max_width"`
	Theme           string    `json:"theme" db:"theme"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ReadingPosition represents user's reading position in a document
type ReadingPosition struct {
	UserID     string    `json:"user_id" db:"user_id"`
	DocumentID string    `json:"document_id" db:"document_id"`
	Position   int       `json:"position" db:"position"`
	PageNumber int       `json:"page_number" db:"page_number"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuthToken represents an authentication token response
type AuthToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// UserUpdate represents fields that can be updated for a user
type UserUpdate struct {
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
}

// SupabaseUser represents a user from Supabase Auth
type SupabaseUser struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}
