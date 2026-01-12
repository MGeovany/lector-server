package domain

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// ReadingPosition represents a user's reading state for a document.
type ReadingPosition struct {
	UserID     string `json:"user_id"`
	DocumentID string `json:"document_id"`

	Progress   float32 `json:"progress"`
	PageNumber int     `json:"page_number"`

	UpdatedAt time.Time `json:"updated_at"`
}

type DocumentMetadata struct {
	OriginalTitle  string `json:"original_title,omitempty"`
	OriginalAuthor string `json:"original_author,omitempty"`
	Language       string `json:"language,omitempty"`
	PageCount      int    `json:"page_count,omitempty"`
	WordCount      int    `json:"word_count,omitempty"`
	FileSize       int64  `json:"file_size,omitempty"`
	Format         string `json:"format,omitempty"`
	Source         string `json:"source,omitempty"`
	HasPassword    bool   `json:"has_password,omitempty"`
}

// Document represents a readable document owned by a user.
type Document struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`

	Title       string  `json:"title"`
	Author      *string `json:"author,omitempty"`
	Description *string `json:"description,omitempty"`

	Content  json.RawMessage  `json:"content"`
	Metadata DocumentMetadata `json:"metadata"`
	Tag      *string          `json:"tag,omitempty"` // Single tag (document can only have one tag)

	IsFavorite bool `json:"is_favorite"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DocumentData is the data transfer representation used by services and handlers.
// Alias to Document so they are interchangeable.
type DocumentData = Document

// DocumentWithPosition represents a document together with the user's current reading state.
type DocumentWithPosition struct {
	DocumentData    *DocumentData    `json:"document"`
	ReadingPosition *ReadingPosition `json:"reading_position,omitempty"`
}

// LibraryResponse is the payload returned by the library endpoint.
type LibraryResponse struct {
	Documents []DocumentWithPosition `json:"documents"`
}

// DocumentRepository defines persistence operations for documents.
type DocumentRepository interface {
	Create(document *Document, token string) error
	GetByID(id string, token string) (*Document, error)
	GetByUserID(userID string, token string) ([]*Document, error)
	Update(document *Document, token string) error
	Delete(id string, token string) error
	Search(userID, query string, token string) ([]*Document, error)
	GetTagsByUserID(userID string, token string) ([]string, error)
	CreateTag(userID string, tagName string, token string) error
	DeleteTag(userID string, tagName string, token string) error

	// Favorites
	SetFavorite(userID string, documentID string, isFavorite bool, token string) error
}

// DocumentService defines the use-case operations for documents.
type DocumentService interface {
	GetDocumentsByUserID(userID string, token string) ([]*DocumentData, error)
	GetDocument(documentID string, token string) (*DocumentData, error)
	DeleteDocument(documentID string, token string) error
	SearchDocuments(userID, query string, token string) ([]*DocumentData, error)
	SetFavorite(userID string, documentID string, isFavorite bool, token string) error
	UpdateDocumentDetails(
		userID string,
		documentID string,
		title *string,
		author *string,
		tag *string,
		token string,
	) (*DocumentData, error)
	GetDocumentTags(userID string, token string) ([]string, error)
	CreateTag(userID string, tagName string, token string) error
	DeleteTag(userID string, tagName string, token string) error
	Upload(
		ctx context.Context,
		userID string,
		file io.Reader,
		token string,
		originalName string,
	) (*DocumentData, error)
}
