package repository

import (
	"pdf-text-reader/internal/domain"
)

// DocumentRepo implements the domain.DocumentRepository interface
type DocumentRepo struct {
	logger domain.Logger
	config domain.Config
}

// NewDocumentRepository creates a new document repository instance
func NewDocumentRepository(logger domain.Logger, config domain.Config) domain.DocumentRepository {
	return &DocumentRepo{
		logger: logger,
		config: config,
	}
}

// Store saves a document to storage
func (r *DocumentRepo) Store(document *domain.ExtractedDocument) error {
	// TODO: Implement document storage in subsequent tasks
	r.logger.Debug("Store method called", "documentID", document.ID)
	return nil
}

// Retrieve gets a document from storage
func (r *DocumentRepo) Retrieve(documentID string) (*domain.ExtractedDocument, error) {
	// TODO: Implement document retrieval in subsequent tasks
	r.logger.Debug("Retrieve method called", "documentID", documentID)
	return nil, nil
}

// Delete removes a document from storage
func (r *DocumentRepo) Delete(documentID string) error {
	// TODO: Implement document deletion in subsequent tasks
	r.logger.Debug("Delete method called", "documentID", documentID)
	return nil
}
