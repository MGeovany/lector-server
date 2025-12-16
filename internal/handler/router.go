		package handler

import (
	"net/http"

	"pdf-text-reader/internal/config"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(container *config.Container) http.Handler {
	router := mux.NewRouter()
	
	// API prefix
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// Health check endpoint (no auth required)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"pdf-text-reader"}`))
	}).Methods("GET")
	
	// Initialize handlers
	authHandler := NewAuthHandler(container)
	documentHandler := NewDocumentHandler(container)
	preferenceHandler := NewPreferenceHandler(container)
	
	// Auth middleware for protected routes
	authMiddleware := AuthMiddleware(container)
	
	// Protected routes (require authentication)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware)
	
	// Auth routes (protected)
	protected.HandleFunc("/auth/profile", authHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/auth/profile", authHandler.UpdateProfile).Methods("PUT")
	protected.HandleFunc("/auth/validate", authHandler.ValidateToken).Methods("GET")
	
	// Document routes (protected)
	protected.HandleFunc("/documents", documentHandler.GetDocuments).Methods("GET")
	protected.HandleFunc("/documents", documentHandler.UploadDocument).Methods("POST")
	protected.HandleFunc("/documents/{id}", documentHandler.GetDocument).Methods("GET")
	protected.HandleFunc("/documents/{id}", documentHandler.DeleteDocument).Methods("DELETE")
	protected.HandleFunc("/documents/search", documentHandler.SearchDocuments).Methods("GET")
	
	// Preference routes (protected)
	protected.HandleFunc("/preferences", preferenceHandler.GetPreferences).Methods("GET")
	protected.HandleFunc("/preferences", preferenceHandler.UpdatePreferences).Methods("PUT")
	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.GetReadingPosition).Methods("GET")
	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.UpdateReadingPosition).Methods("PUT")
	
	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173", // SvelteKit dev server
			"http://localhost:4173", // SvelteKit preview
			"http://localhost:3000", // Alternative dev port
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Link",
		},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	
	return c.Handler(router)
}
