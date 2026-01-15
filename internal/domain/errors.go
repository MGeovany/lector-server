package domain

import "errors"

// Domain errors
var (
	ErrDocumentNotFound        = errors.New("document not found")
	ErrAccessDenied            = errors.New("access denied")
	ErrReadingPositionNotFound = errors.New("reading position not found")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidToken            = errors.New("invalid token")
	ErrStorageLimitExceeded    = errors.New("storage limit exceeded")
	ErrInvalidFile             = errors.New("invalid file")
	ErrTagNotFound             = errors.New("tag not found")
	ErrTagAlreadyExists        = errors.New("tag already exists")
)

// ValidationError represents a validation error with field and message information.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}
