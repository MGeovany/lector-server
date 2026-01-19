package domain

import "time"

// Highlight represents a user's saved excerpt from a document.
type Highlight struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	DocumentID string  `json:"document_id"`
	Quote      string  `json:"quote"`
	PageNumber *int    `json:"page_number,omitempty"`
	Progress   *float32 `json:"progress,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// HighlightRepository defines persistence operations for highlights.
type HighlightRepository interface {
	Create(highlight *Highlight, token string) (*Highlight, error)
	ListByUser(userID string, documentID *string, token string) ([]*Highlight, error)
	Delete(userID string, highlightID string, token string) error
}

// HighlightService defines the use-case operations for highlights.
type HighlightService interface {
	CreateHighlight(userID string, highlight *Highlight, token string) (*Highlight, error)
	ListHighlights(userID string, documentID *string, token string) ([]*Highlight, error)
	DeleteHighlight(userID string, highlightID string, token string) error
}

