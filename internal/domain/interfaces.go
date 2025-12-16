package domain

import (
	"mime/multipart"
)

// User Management Interfaces

// UserService defines the interface for user management operations
type UserService interface {
	Login(email, password string) (*AuthToken, error)
	GetProfile(userID string) (*User, error)
	UpdateProfile(userID string, updates *UserUpdate) error
	Logout(userID string) error
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
}

// Document Management Interfaces

// DocumentLibraryService defines the interface for document library operations
type DocumentLibraryService interface {
	UploadDocument(userID string, file multipart.File, filename string) (*Document, error)
	GetUserDocuments(userID string) ([]*Document, error)
	GetDocument(userID, documentID string) (*Document, error)
	DeleteDocument(userID, documentID string) error
	SearchDocuments(userID, query string) ([]*Document, error)
}

// DocumentRepository defines the interface for document storage operations
type DocumentRepository interface {
	Create(document *Document) error
	GetByID(id string) (*Document, error)
	GetByUserID(userID string) ([]*Document, error)
	Update(document *Document) error
	Delete(id string) error
	Search(userID, query string) ([]*Document, error)
}

// Preference Management Interfaces

// PreferenceService defines the interface for user preference operations
type PreferenceService interface {
	GetPreferences(userID string) (*UserPreferences, error)
	UpdatePreferences(userID string, prefs *UserPreferences) error
	GetReadingPosition(userID, documentID string) (*ReadingPosition, error)
	UpdateReadingPosition(userID, documentID string, position *ReadingPosition) error
}

// PreferenceRepository defines the interface for preference data operations
type PreferenceRepository interface {
	GetPreferences(userID string) (*UserPreferences, error)
	UpdatePreferences(prefs *UserPreferences) error
	GetReadingPosition(userID, documentID string) (*ReadingPosition, error)
	UpdateReadingPosition(position *ReadingPosition) error
}

// Session Management Interfaces

// SessionRepository defines the interface for session data operations
type SessionRepository interface {
	Create(session *Session) error
	GetByID(id string) (*Session, error)
	GetByUserID(userID string) ([]*Session, error)
	Delete(id string) error
	DeleteExpired() error
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
	GetSupabaseURL() string
	GetSupabaseKey() string
	GetJWTSecret() string
}
