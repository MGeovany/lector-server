package service

import (
	"pdf-text-reader/internal/domain"
)

// PDFService implements the PDF processing business logic
type PDFService struct {
	textExtractor domain.TextExtractor
	fileHandler   domain.FileHandler
	repository    domain.DocumentRepository
	logger        domain.Logger
}

// NewPDFService creates a new PDF service instance
func NewPDFService(
	textExtractor domain.TextExtractor,
	fileHandler domain.FileHandler,
	repository domain.DocumentRepository,
	logger domain.Logger,
) *PDFService {
	return &PDFService{
		textExtractor: textExtractor,
		fileHandler:   fileHandler,
		repository:    repository,
		logger:        logger,
	}
}

// TODO: Implement PDF processing methods in subsequent tasks
// This file will be expanded with actual implementation in later tasks
