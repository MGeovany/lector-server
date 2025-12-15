package config

import (
	"os"
	"strconv"

	"pdf-text-reader/internal/domain"
)

// AppConfig implements the domain.Config interface
type AppConfig struct {
	ServerPort   string
	UploadPath   string
	MaxFileSize  int64
	LogLevel     string
	DatabasePath string
}

// NewConfig creates a new configuration instance with default values
func NewConfig() domain.Config {
	return &AppConfig{
		ServerPort:   getEnvOrDefault("SERVER_PORT", "8080"),
		UploadPath:   getEnvOrDefault("UPLOAD_PATH", "./uploads"),
		MaxFileSize:  getEnvInt64OrDefault("MAX_FILE_SIZE", 50*1024*1024), // 50MB default
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "info"),
		DatabasePath: getEnvOrDefault("DATABASE_PATH", "./data"),
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

// GetDatabasePath returns the database path
func (c *AppConfig) GetDatabasePath() string {
	return c.DatabasePath
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
