package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"pdf-text-reader/internal/domain"

	"encoding/json"
	"github.com/google/uuid"
)

type DocumentService struct {
	storage      StorageService
	repo         domain.DocumentRepository
	logger       domain.Logger
	pdfProcessor *PDFProcessor
}

func NewDocumentService(
	repo domain.DocumentRepository,
	storage StorageService,
	logger domain.Logger,
) *DocumentService {
	return &DocumentService{
		storage:      storage,
		repo:         repo,
		logger:       logger,
		pdfProcessor: NewPDFProcessor(logger),
	}
}

func (s *DocumentService) GetDocumentsByUserID(userID string, token string) ([]*domain.Document, error) {
	documents, err := s.repo.GetByUserID(userID, token)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (s *DocumentService) GetDocument(documentID string, token string) (*domain.Document, error) {
	document, err := s.repo.GetByID(documentID, token)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func (s *DocumentService) DeleteDocument(documentID string, token string) error {
	err := s.repo.Delete(documentID, token)
	if err != nil {
		return err
	}
	return nil
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

	// Read file to get size and content
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

	// Process PDF to extract text and metadata
	// For small files, process immediately; for larger files, process in background
	// Threshold: 2MB - files larger than this will be processed asynchronously
	const asyncThreshold = 2 * 1024 * 1024 // 2MB

	var contentJSON json.RawMessage
	var metadata domain.DocumentMetadata
	title := originalName

	if totalSize < asyncThreshold {
		// Process synchronously for small files
		blocks, pdfMetadata, err := s.pdfProcessor.ProcessPDF(fileBytes)
		if err != nil {
			s.logger.Error("Failed to process PDF", err, "doc_id", docID)
			contentJSON = json.RawMessage("[]")
			metadata = domain.DocumentMetadata{}
		} else {
			contentJSON, err = s.pdfProcessor.ConvertToJSON(blocks)
			if err != nil {
				s.logger.Error("Failed to convert blocks to JSON", err, "doc_id", docID)
				contentJSON = json.RawMessage("[]")
			}

			if pdfMetadata.Title != "" {
				title = pdfMetadata.Title
			}

			metadata = domain.DocumentMetadata{
				Author:      pdfMetadata.Author,
				PageCount:   pdfMetadata.PageCount,
				HasPassword: pdfMetadata.HasPassword,
			}

			s.logger.Info("Document processed synchronously",
				"doc_id", docID,
				"blocks_count", len(blocks),
				"page_count", pdfMetadata.PageCount,
			)
		}
	} else {
		// For larger files, create document first and process in background
		contentJSON = json.RawMessage("[]")
		metadata = domain.DocumentMetadata{}

		// Process in background goroutine
		go func() {
			blocks, pdfMetadata, err := s.pdfProcessor.ProcessPDF(fileBytes)
			if err != nil {
				s.logger.Error("Failed to process PDF in background", err, "doc_id", docID)
				return
			}

			contentJSON, err := s.pdfProcessor.ConvertToJSON(blocks)
			if err != nil {
				s.logger.Error("Failed to convert blocks to JSON in background", err, "doc_id", docID)
				return
			}

			// Determine title
			docTitle := originalName
			if pdfMetadata.Title != "" {
				docTitle = pdfMetadata.Title
			}

			// Update document with processed content
			updatedDoc := &domain.Document{
				ID:      docID,
				UserID:  userID,
				Title:   docTitle,
				Content: contentJSON,
				Metadata: domain.DocumentMetadata{
					Author:      pdfMetadata.Author,
					PageCount:   pdfMetadata.PageCount,
					HasPassword: pdfMetadata.HasPassword,
				},
				UpdatedAt: time.Now().UTC(),
			}

			if err := s.repo.Update(updatedDoc, token); err != nil {
				s.logger.Error("Failed to update document with processed content", err, "doc_id", docID)
				return
			}

			s.logger.Info("Document processed in background",
				"doc_id", docID,
				"blocks_count", len(blocks),
				"page_count", pdfMetadata.PageCount,
			)
		}()

		s.logger.Info("Document created, processing in background", "doc_id", docID, "file_size", totalSize)
	}

	doc := &domain.Document{
		ID:           docID,
		UserID:       userID,
		OriginalName: originalName,
		Title:        title,
		Content:      contentJSON,
		Metadata:     metadata,
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
