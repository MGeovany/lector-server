package config

import (
	"pdf-text-reader/internal/domain"
	"pdf-text-reader/internal/infra/supabase"
	"pdf-text-reader/internal/repository"
	"pdf-text-reader/internal/service"
	"pdf-text-reader/pkg/logger"
)

// Container holds all application dependencies
type Container struct {
	Config            domain.Config
	Logger            domain.Logger
	SupabaseClient    domain.SupabaseClient
	DocumentService   domain.DocumentService
	AuthService       domain.AuthService
	StorageService    domain.StorageService
	PreferenceService domain.PreferenceService
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	cfg := NewConfig()
	log := logger.NewLogger(cfg.GetLogLevel())

	// Supabase client
	supabaseClient := supabase.NewSupabaseClient(cfg, log)
	if err := supabaseClient.Initialize(); err != nil {
		log.Error("Failed to initialize Supabase client", err)
		panic(err)
	}

	// Repositories
	documentRepo := repository.NewSupabaseDocumentRepository(
		supabaseClient,
		log,
	)

	preferenceRepo := repository.NewSupabasePreferenceRepository(
		supabaseClient,
		log,
	)

	// Services

	storageService := service.NewStorageService(
		cfg.GetSupabaseURL(),
		cfg.GetSupabaseKey(),
	)

	documentService := service.NewDocumentService(
		documentRepo,
		storageService,
		log,
	)

	authService := service.NewAuthService(
		supabaseClient,
		log,
	)

	preferenceService := service.NewPreferenceService(
		preferenceRepo,
		log,
	)

	return &Container{
		Config:            cfg,
		Logger:            log,
		SupabaseClient:    supabaseClient,
		DocumentService:   documentService,
		AuthService:       authService,
		StorageService:    storageService,
		PreferenceService: preferenceService,
	}
}
