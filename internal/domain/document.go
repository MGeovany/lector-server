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

// Validate checks if the reading position has all required fields and valid values.
// Returns an error if validation fails, nil otherwise.
func (r *ReadingPosition) Validate() error {
	if r.UserID == "" {
		return &ValidationError{Field: "user_id", Message: "user ID is required"}
	}
	if r.DocumentID == "" {
		return &ValidationError{Field: "document_id", Message: "document ID is required"}
	}
	if r.Progress < 0 || r.Progress > 1.0 {
		return &ValidationError{Field: "progress", Message: "progress must be between 0 and 1"}
	}
	if r.PageNumber < 0 {
		return &ValidationError{Field: "page_number", Message: "page number cannot be negative"}
	}
	return nil
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

// Validate checks if the metadata has valid values.
// Returns an error if validation fails, nil otherwise.
func (m *DocumentMetadata) Validate() error {
	if m.FileSize < 0 {
		return &ValidationError{Field: "file_size", Message: "file size cannot be negative"}
	}
	if m.PageCount < 0 {
		return &ValidationError{Field: "page_count", Message: "page count cannot be negative"}
	}
	if m.WordCount < 0 {
		return &ValidationError{Field: "word_count", Message: "word count cannot be negative"}
	}
	return nil
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

	// Offline-first fields
	OriginalStoragePath     *string         `json:"original_storage_path,omitempty"`
	OriginalFileName        *string         `json:"original_file_name,omitempty"`
	OriginalMimeType        *string         `json:"original_mime_type,omitempty"`
	OriginalSizeBytes       *int64          `json:"original_size_bytes,omitempty"`
	OriginalChecksumSHA256  *string         `json:"original_checksum_sha256,omitempty"`
	OptimizedContent        json.RawMessage `json:"optimized_content,omitempty"`
	OptimizedVersion        int             `json:"optimized_version"`
	OptimizedSizeBytes      *int64          `json:"optimized_size_bytes,omitempty"`
	OptimizedChecksumSHA256 *string         `json:"optimized_checksum_sha256,omitempty"`
	ProcessingStatus        string          `json:"processing_status"`
	ProcessingError         *string         `json:"processing_error,omitempty"`
	LanguageCode            *string         `json:"language_code,omitempty"`
	ProcessedAt             *time.Time      `json:"processed_at,omitempty"`

	IsFavorite bool `json:"is_favorite"`

	// Optional reading position (when requested by endpoints like documents/user/{id}).
	ReadingPosition *ReadingPosition `json:"reading_position,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks if the document has all required fields and valid values.
// Returns an error if validation fails, nil otherwise.
func (d *Document) Validate() error {
	if d.ID == "" {
		return &ValidationError{Field: "id", Message: "document ID is required"}
	}
	if d.UserID == "" {
		return &ValidationError{Field: "user_id", Message: "user ID is required"}
	}
	if d.Title == "" {
		return &ValidationError{Field: "title", Message: "title is required"}
	}
	if err := d.Metadata.Validate(); err != nil {
		return err
	}
	return nil
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
	GetOptimizedByID(id string, token string) (*OptimizedDocument, error)
	GetOptimizedMetaByID(id string, token string) (*OptimizedDocument, error)
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
	GetOptimizedDocument(documentID string, token string) (*OptimizedDocument, error)
	GetOptimizedDocumentMeta(documentID string, token string) (*OptimizedDocument, error)
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

// OptimizedDocument is a lightweight, offline-friendly representation.
// Pages is typically a JSONB array of strings stored in `documents.optimized_content`.
type OptimizedDocument struct {
	DocumentID           string     `json:"document_id"`
	UserID               string     `json:"-"`
	ProcessingStatus     string     `json:"processing_status"`
	OptimizedVersion     int        `json:"optimized_version"`
	OptimizedChecksumSHA *string    `json:"optimized_checksum_sha256,omitempty"`
	OptimizedSizeBytes   *int64     `json:"optimized_size_bytes,omitempty"`
	LanguageCode         *string    `json:"language_code,omitempty"`
	ProcessedAt          *time.Time `json:"processed_at,omitempty"`
	Pages                []string   `json:"pages,omitempty"`
}
