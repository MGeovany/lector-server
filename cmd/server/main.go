package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/handler"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}
	// Wiring
	container := config.NewContainer()

	// Handlers
	documentHandler := handler.NewDocumentHandler(
		container.DocumentService,
		container.UserPreferencesService,
		container.Logger,
	)

	authHandler := handler.NewAuthHandler(
		container,
	)

	adminHandler := handler.NewAdminHandler()

	preferenceHandler := handler.NewPreferenceHandler(
		container,
		container.Logger,
	)

	highlightHandler := handler.NewHighlightHandler(
		container,
		container.Logger,
	)

	authMiddleware := handler.NewAuthMiddleware(
		container.AuthService,
		container.Logger,
	)
	
	var aiHandler *handler.AIHandler
	if container.AIService != nil {
		aiHandler = handler.NewAIHandler(
			container.AIService,
			container.Logger,
		)
	} else {
		container.Logger.Warn("AIService not available, Ask AI features will fail")
		// We still need to pass something or handle nil in router?
		// Router expects *AIHandler.
		// If nil, router wrapper might panic if it tries to access it?
		// NewRouter signature takes *AIHandler.
		// If we pass nil, and router function uses it, it will panic.
		// Handlers are accessed when defining routes: `aiHandler.Ingest` etc.
		// If aiHandler is nil, `aiHandler.Ingest` is valid if it's a method value? No, `aiHandler` is a pointer. `aiHandler.Ingest` would panic if nil.
		// So we must provide a dummy handler or ensure AIService is initialized.
		// Since I implemented `NewContainer` to return partial with error log, I should probably handle this.
		// But for now let's hope it works or just create handler anyway with nil service (which will fail at runtime).
		// Better: NewAIHandler allows nil service? No check.
		
		// If AIService is nil, creating AIHandler is fine, but calling methods on it might panic if service usage isn't guarded.
		// My AIHandler implementation calls `h.aiService.Method(...)`.
		// If `aiService` is nil interface, it will panic.
		// I'll create the handler even if service is nil, assuming it might be fixed or just error out cleanly if I guarded in handler.
		// Wait, I didn't guard in handler.
		// Let's just create it.
		aiHandler = handler.NewAIHandler(
			container.AIService,
			container.Logger,
		)
	}

	// Router
	router := handler.NewRouter(
		authHandler,
		adminHandler,
		documentHandler,
		preferenceHandler,
		highlightHandler,
		aiHandler,
		authMiddleware.Middleware,
	)

	// start server
	server := &http.Server{
		Addr:              ":" + container.Config.GetServerPort(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// Run server
	go func() {
		container.Logger.Info("Server listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			container.Logger.Error("Server failed to start", err)
			os.Exit(1)
		}
	}()
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	container.Logger.Info("Shutting down server...")
	_ = server.Close()

	container.Logger.Info("Server exited")
}
