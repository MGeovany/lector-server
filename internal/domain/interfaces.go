package domain

import (
	"io"
	"mime/multipart"
)

// PDFProcessor defines the main interface for PDF processing operations
type PDFProcessor interface {
	ExtractText(file io.Reader) (*ExtractedDocument, error)
	ValidateFile(file io.Reader) error
	GetMetadata(file io.Reader) (*DocumentMetadata, error)
}

// TextExtractor defines the strategy interface for text extraction
type TextExtractor interface {
	Extract(document *PDFDocument) (*ExtractedText, error)
	SupportsFormat(format PDFFormat) bool
}

// FileHandler defines the interface for file operations
type FileHandler interface {
	SaveUpload(file multipart.File) (*FileInfo, error)
	CleanupTemporary(fileID string) error
	ValidateFileType(file multipart.File) error
}

// DocumentRepository defines the interface for document storage operations
type DocumentRepository interface {
	Store(document *ExtractedDocument) error
	Retrieve(documentID string) (*ExtractedDocument, error)
	Delete(documentID string) error
}

// Logger defines the interface for logging operations
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// Config defines the interface for configuration management
type Config interface {
	GetServerPort() string
	GetUploadPath() string
	GetMaxFileSize() int64
	GetLogLevel() string
	GetDatabasePath() string
}
