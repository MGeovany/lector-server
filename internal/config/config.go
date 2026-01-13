package config

import (
	"os"
	"strconv"

	"pdf-text-reader/internal/domain"
)

// AppConfig implements the domain.Config interface
type AppConfig struct {
	ServerPort  string
	UploadPath  string
	MaxFileSize int64
	LogLevel    string
	SupabaseURL string
	SupabaseKey string
	JWTSecret   string
}

// NewConfig creates a new configuration instance with default values
func NewConfig() domain.Config {
	return &AppConfig{
		// Cloud Run (and many PaaS) provide the listening port via PORT.
		// Keep SERVER_PORT for local/dev compatibility.
		ServerPort:  getEnvOrDefault("PORT", getEnvOrDefault("SERVER_PORT", "8080")),
		UploadPath:  getEnvOrDefault("UPLOAD_PATH", "./uploads"),
		MaxFileSize: getEnvInt64OrDefault("MAX_FILE_SIZE", 50*1024*1024), // 50MB default
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
		SupabaseURL: getEnvOrDefault("SUPABASE_URL", ""),
		SupabaseKey: getEnvOrDefault("SUPABASE_ANON_KEY", ""),
		JWTSecret:   getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
	}
}

// GetServerPort returns the server port
func (c *AppConfig) GetServerPort() string {
	return c.ServerPort
}

// GetUploadPath returns the upload directory path
func (c *AppConfig) GetUploadPath() string {
	return c.UploadPath
}

// GetMaxFileSize returns the maximum allowed file size
func (c *AppConfig) GetMaxFileSize() int64 {
	return c.MaxFileSize
}

// GetLogLevel returns the logging level
func (c *AppConfig) GetLogLevel() string {
	return c.LogLevel
}

// GetSupabaseURL returns the Supabase URL
func (c *AppConfig) GetSupabaseURL() string {
	return c.SupabaseURL
}

// GetSupabaseKey returns the Supabase anon key
func (c *AppConfig) GetSupabaseKey() string {
	return c.SupabaseKey
}

// GetJWTSecret returns the JWT secret key
func (c *AppConfig) GetJWTSecret() string {
	return c.JWTSecret
}

// Helper functions for environment variable handling
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
