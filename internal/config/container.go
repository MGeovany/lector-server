package config

import (
	"pdf-text-reader/internal/domain"
	"pdf-text-reader/pkg/logger"
)

// Container holds all application dependencies
type Container struct {
	Config domain.Config
	Logger domain.Logger
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	config := NewConfig()
	appLogger := logger.NewLogger(config.GetLogLevel())
	
	return &Container{
		Config: config,
		Logger: appLogger,
	}
}

// GetConfig returns the configuration instance
func (c *Container) GetConfig() domain.Config {
	return c.Config
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() domain.Logger {
	return c.Logger
}
