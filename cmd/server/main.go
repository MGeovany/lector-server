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

	preferenceHandler := handler.NewPreferenceHandler(
		container,
		container.Logger,
	)

	authMiddleware := handler.NewAuthMiddleware(
		container.AuthService,
		container.Logger,
	)

	// Router
	router := handler.NewRouter(
		authHandler,
		documentHandler,
		preferenceHandler,
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
