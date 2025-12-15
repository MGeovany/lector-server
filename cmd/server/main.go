package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"pdf-text-reader/internal/config"
)

func main() {
	container := config.NewContainer()
	logger := container.GetLogger()
	cfg := container.GetConfig()

	logger.Info("Starting PDF Text Reader server", "port", cfg.GetServerPort())

	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","service":"pdf-text-reader"}`)
	})

	server := &http.Server{
		Addr:    ":" + cfg.GetServerPort(),
		Handler: mux,
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
