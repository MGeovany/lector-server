package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func NewRouter(
	authHandler *AuthHandler,
	adminHandler *AdminHandler,
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

	// Admin routes (NOT behind auth middleware; protected by X-Admin-Secret)
	admin := api.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/users/{id}/account-disabled", adminHandler.SetAccountDisabled).Methods(http.MethodPost)

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware)

	// Auth
	protected.HandleFunc("/auth/profile", authHandler.GetProfile).Methods(http.MethodGet)
	protected.HandleFunc("/auth/profile", authHandler.UpdateProfile).Methods(http.MethodPut)
	protected.HandleFunc("/auth/validate", authHandler.ValidateToken).Methods(http.MethodGet)
	protected.HandleFunc("/auth/account-deletion-request", authHandler.RequestAccountDeletion).Methods(http.MethodPost)

	// Documents
	// Gets all the card information
	protected.HandleFunc("/documents/library", documentHandler.GetLibrary).Methods(http.MethodGet)

	// Get all the doc information
	protected.HandleFunc("/documents", documentHandler.UploadDocument).Methods(http.MethodPost)

	// Get doc data by ID
	protected.HandleFunc("/documents/{id}", documentHandler.GetDocument).Methods(http.MethodGet)

	// Update doc by ID
	protected.HandleFunc("/documents/{id}", documentHandler.UpdateDocument).Methods(http.MethodPut)

	// Favorite/unfavorite doc
	protected.HandleFunc("/documents/{id}/favorite", documentHandler.SetFavorite).Methods(http.MethodPut)

	// Delete doc by ID
	protected.HandleFunc("/documents/{id}", documentHandler.DeleteDocument).Methods(http.MethodDelete)

	// Search docs
	protected.HandleFunc("/documents/search", documentHandler.SearchDocuments).Methods(http.MethodGet)

	// Get all the docs by user ID
	protected.HandleFunc("/documents/user/{id}", documentHandler.GetDocumentsByUserID).Methods(http.MethodGet)

	// Get all document tags for the authenticated user
	protected.HandleFunc("/document-tags", documentHandler.GetDocumentTags).Methods(http.MethodGet)

	// Create a new document tag for the authenticated user
	protected.HandleFunc("/document-tags", documentHandler.CreateTag).Methods(http.MethodPost)

	// Delete a document tag for the authenticated user
	protected.HandleFunc("/document-tags/{name}", documentHandler.DeleteTag).Methods(http.MethodDelete)

	// Preferences
	protected.HandleFunc("/preferences", preferenceHandler.GetPreferences).Methods(http.MethodGet)
	protected.HandleFunc("/preferences", preferenceHandler.UpdatePreferences).Methods(http.MethodPut)

	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.GetReadingPosition).Methods(http.MethodGet)

	protected.HandleFunc("/preferences/reading-position/{documentId}", preferenceHandler.UpdateReadingPosition).Methods(http.MethodPut)

	// Get all reading positions for the authenticated user
	protected.HandleFunc("/preferences/reading-positions", preferenceHandler.GetAllReadingPositions).Methods(http.MethodGet)

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
