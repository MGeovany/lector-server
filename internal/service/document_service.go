package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"pdf-text-reader/internal/domain"

	"github.com/google/uuid"
)

type DocumentService struct {
	storage StorageService
	repo    domain.DocumentRepository
	logger  domain.Logger
}

func NewDocumentService(
	repo domain.DocumentRepository,
	storage StorageService,
	logger domain.Logger,
) *DocumentService {
	return &DocumentService{
		storage: storage,
		repo:    repo,
		logger:  logger,
	}
}

func (s *DocumentService) Upload(
	ctx context.Context,
	userID string,
	file io.Reader,
) (*domain.Document, error) {

	docID := uuid.New().String()
	path := fmt.Sprintf("documents/%s/%s.pdf", userID, docID)

	if err := s.storage.Upload(ctx, path, file); err != nil {
		return nil, err
	}

	doc := &domain.Document{
		ID:        docID,
		UserID:    userID,
		FilePath:  path,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}

	return doc, nil
}
