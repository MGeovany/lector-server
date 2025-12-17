package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"pdf-text-reader/internal/domain"

	"github.com/google/uuid"
	"encoding/json"
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

func (s *DocumentService) GetDocumentsByUserID(userID string, token string) ([]*domain.Document, error) {
	documents, err := s.repo.GetByUserID(userID, token)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (s *DocumentService) Upload(
	ctx context.Context,
	userID string,
	file io.Reader,
	token string,
	originalName string,
) (*domain.Document, error) {

	docID := uuid.New().String()
	// Path should be relative to bucket, not include bucket name
	path := fmt.Sprintf("%s/%s.pdf", userID, docID)

	// Read file to get size
	fileBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	var totalSize int64
	for {
		n, err := file.Read(buf)
		if n > 0 {
			fileBytes = append(fileBytes, buf[:n]...)
			totalSize += int64(n)
		}
		if err != nil {
			break
		}
	}

	// Upload file (need to create new reader from bytes)
	fileReader := bytes.NewReader(fileBytes)
	if err := s.storage.Upload(ctx, path, fileReader, token); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Use original filename or generate one
	if originalName == "" {
		originalName = docID + ".pdf"
	}

	doc := &domain.Document{
		ID:           docID,
		UserID:       userID,
		OriginalName: originalName,
		Title:        originalName,              // Will be updated when we extract title from PDF
		// Empty JSON array, will be populated when PDF is processed
		Content:      json.RawMessage("[]"),
		Metadata:     domain.DocumentMetadata{}, // Empty metadata
		FilePath:     path,
		FileSize:     totalSize,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(doc, token); err != nil {
		return nil, err
	}

	return doc, nil
}
