package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"pdf-text-reader/internal/config"
	"pdf-text-reader/internal/handler"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	container := config.NewContainer()
	logger := container.GetLogger()
	cfg := container.GetConfig()
	supabaseClient := container.GetSupabaseClient()

	logger.Info("Starting PDF Text Reader server", "port", cfg.GetServerPort())

	// Initialize Supabase client
	if err := supabaseClient.Initialize(); err != nil {
		logger.Error("Failed to initialize Supabase client", err)
		os.Exit(1)
	}

	// Create router with all routes and CORS configured
	router := handler.NewRouter(container)

	server := &http.Server{
		Addr:    ":" + cfg.GetServerPort(),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	
	if err := server.Close(); err != nil {
		logger.Error("Server forced to shutdown", err)
	}

	logger.Info("Server exited")
}
