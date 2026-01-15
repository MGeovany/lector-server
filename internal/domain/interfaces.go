package domain

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
	GetMaxFileSize() int64
	GetLogLevel() string
	GetSupabaseURL() string
	GetSupabaseKey() string
	GetJWTSecret() string
}
