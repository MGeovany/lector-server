package config

import (
	"pdf-text-reader/internal/domain"
	"pdf-text-reader/internal/repository"
	"pdf-text-reader/pkg/logger"
)

// Container holds all application dependencies
type Container struct {
	Config               domain.Config
	Logger               domain.Logger
	SupabaseClient       domain.SupabaseClient
	DocumentRepository   domain.DocumentRepository
	PreferenceRepository domain.PreferenceRepository
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	config := NewConfig()
	appLogger := logger.NewLogger(config.GetLogLevel())
	
	// Initialize Supabase client
	supabaseClient := repository.NewSupabaseClient(config, appLogger)
	
	// Initialize repositories
	documentRepo := repository.NewSupabaseDocumentRepository(supabaseClient, appLogger)
	preferenceRepo := repository.NewSupabasePreferenceRepository(supabaseClient, appLogger)
	
	return &Container{
		Config:               config,
		Logger:               appLogger,
		SupabaseClient:       supabaseClient,
		DocumentRepository:   documentRepo,
		PreferenceRepository: preferenceRepo,
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

// GetSupabaseClient returns the Supabase client instance
func (c *Container) GetSupabaseClient() domain.SupabaseClient {
	return c.SupabaseClient
}

// GetDocumentRepository returns the document repository instance
func (c *Container) GetDocumentRepository() domain.DocumentRepository {
	return c.DocumentRepository
}

// GetPreferenceRepository returns the preference repository instance
func (c *Container) GetPreferenceRepository() domain.PreferenceRepository {
	return c.PreferenceRepository
}
