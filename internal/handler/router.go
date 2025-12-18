package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func NewRouter(
	authHandler *AuthHandler,
	documentHandler *DocumentHandler,
	preferenceHandler *PreferenceHandler,
	authMiddleware func(http.Handler) http.Handler,

) http.Handler {

	router := mux.NewRouter()

	// Health check (public)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"pdf-text-reader"}`))
	}).Methods(http.MethodGet)

	// API v1
	api := router.PathPrefix("/api/v1").Subrouter()

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware)

	// Auth
	protected.HandleFunc("/auth/profile", authHandler.GetProfile).Methods(http.MethodGet)
	protected.HandleFunc("/auth/profile", authHandler.UpdateProfile).Methods(http.MethodPut)
	protected.HandleFunc("/auth/validate", authHandler.ValidateToken).Methods(http.MethodGet)

	// Documents
	protected.HandleFunc("/documents", documentHandler.UploadDocument).Methods(http.MethodPost)
	protected.HandleFunc("/documents/{id}", documentHandler.GetDocument).Methods(http.MethodGet)
	protected.HandleFunc("/documents/{id}", documentHandler.DeleteDocument).Methods(http.MethodDelete)
	protected.HandleFunc("/documents/search", documentHandler.SearchDocuments).Methods(http.MethodGet)
	protected.HandleFunc("/documents/user/{id}", documentHandler.GetDocumentsByUserID).Methods(http.MethodGet)

	// Preferences
	protected.HandleFunc("/preferences", preferenceHandler.GetPreferences).Methods(http.MethodGet)
	protected.HandleFunc("/preferences", preferenceHandler.UpdatePreferences).Methods(http.MethodPut)
	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.GetReadingPosition).Methods(http.MethodGet)
	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.UpdateReadingPosition).Methods(http.MethodPut)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			// Local development
			"http://localhost:5173",
			"http://localhost:4173",
			"http://localhost:3000",
			// Production frontend
			"https://lector.thefndrs.com",
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
		},
		AllowCredentials: true,
		MaxAge:           300,
	})

	return c.Handler(router)
}
