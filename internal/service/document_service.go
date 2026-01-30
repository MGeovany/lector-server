package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pdf-text-reader/internal/domain"

	"github.com/google/uuid"
)

type DocumentService struct {
	storage      StorageService
	repo         domain.DocumentRepository
	prefsRepo    domain.UserPreferencesRepository
	logger       domain.Logger
	pdfProcessor *PDFProcessor
}

func NewDocumentService(
	repo domain.DocumentRepository,
	prefsRepo domain.UserPreferencesRepository,
	storage StorageService,
	logger domain.Logger,
) *DocumentService {
	return &DocumentService{
		storage:      storage,
		repo:         repo,
		prefsRepo:    prefsRepo,
		logger:       logger,
		pdfProcessor: NewPDFProcessor(logger),
	}
}

func (s *DocumentService) GetDocumentsByUserID(userID string, token string) ([]*domain.DocumentData, error) {
	documents, err := s.repo.GetByUserID(userID, token)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (s *DocumentService) GetDocument(documentID string, token string) (*domain.DocumentData, error) {
	document, err := s.repo.GetByID(documentID, token)
	if err != nil {
		return nil, err
	}
	return document, nil
}

func (s *DocumentService) GetOptimizedDocument(documentID string, token string) (*domain.OptimizedDocument, error) {
	return s.repo.GetOptimizedByID(documentID, token)
}

func (s *DocumentService) GetOptimizedDocumentMeta(documentID string, token string) (*domain.OptimizedDocument, error) {
	return s.repo.GetOptimizedMetaByID(documentID, token)
}

func (s *DocumentService) DeleteDocument(documentID string, token string) error {
	err := s.repo.Delete(documentID, token)
	if err != nil {
		return err
	}
	return nil
}

func (s *DocumentService) SearchDocuments(userID, query string, token string) ([]*domain.DocumentData, error) {
	documents, err := s.repo.Search(userID, query, token)
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (s *DocumentService) SetFavorite(userID string, documentID string, isFavorite bool, token string) error {
	// Verify ownership to prevent cross-user writes.
	doc, err := s.repo.GetByID(documentID, token)
	if err != nil {
		return err
	}
	if doc.UserID != userID {
		return fmt.Errorf("access denied")
	}
	return s.repo.SetFavorite(userID, documentID, isFavorite, token)
}

func (s *DocumentService) GetDocumentTags(userID string, token string) ([]string, error) {
	tags, err := s.repo.GetTagsByUserID(userID, token)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *DocumentService) CreateTag(userID string, tagName string, token string) error {
	// Validate tag name
	if tagName == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// Trim whitespace
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	err := s.repo.CreateTag(userID, tagName, token)
	if err != nil {
		return err
	}

	return nil
}

func (s *DocumentService) DeleteTag(userID string, tagName string, token string) error {
	// Validate tag name
	if tagName == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// Trim whitespace
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	err := s.repo.DeleteTag(userID, tagName, token)
	if err != nil {
		return err
	}

	return nil
}

func (s *DocumentService) UpdateDocumentDetails(
	userID string,
	documentID string,
	title *string,
	author *string,
	tag *string,
	token string,
) (*domain.DocumentData, error) {
	doc, err := s.repo.GetByID(documentID, token)
	if err != nil {
		return nil, err
	}
	if doc.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	if title != nil {
		doc.Title = *title
	}
	if author != nil {
		doc.Author = author
	}
	if tag != nil {
		doc.Tag = tag
	}

	doc.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(doc, token); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(documentID, token)
	if err != nil {
		// If re-fetch fails, at least return our updated in-memory doc.
		return doc, nil
	}
	return updated, nil
}

func (s *DocumentService) Upload(
	ctx context.Context,
	userID string,
	file io.Reader,
	token string,
	originalName string,
) (*domain.DocumentData, error) {
	// Determine per-user storage quota from preferences.
	// Default: 15MB (free). Paid: 50GB.
	maxUserStorage := domain.StorageLimitBytesForPlan("free")
	if s.prefsRepo != nil {
		if prefs, err := s.prefsRepo.GetPreferences(userID, token); err == nil && prefs != nil {
			// Prefer explicit storage_limit_bytes, but fall back to computing from plan.
			if prefs.StorageLimitBytes > 0 {
				maxUserStorage = prefs.StorageLimitBytes
			} else {
				maxUserStorage = domain.StorageLimitBytesForPlan(prefs.SubscriptionPlan)
			}
		}
	}

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

	// Compute original checksum + mime type
	origSum := sha256.Sum256(fileBytes)
	origChecksum := hex.EncodeToString(origSum[:])
	sample := fileBytes
	if len(sample) > 512 {
		sample = sample[:512]
	}
	origMime := http.DetectContentType(sample)

	// Enforce per-user storage quota BEFORE uploading to storage
	// Get current documents to calculate total storage used
	existingDocs, err := s.repo.GetByUserID(userID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current storage usage: %w", err)
	}

	var currentUsage int64
	for _, d := range existingDocs {
		if d.OriginalSizeBytes != nil && *d.OriginalSizeBytes > 0 {
			currentUsage += *d.OriginalSizeBytes
			continue
		}
		currentUsage += d.Metadata.FileSize
	}

	if currentUsage+totalSize > maxUserStorage {
		return nil, fmt.Errorf("storage limit exceeded: user has %d bytes used, upload would exceed %d bytes", currentUsage, maxUserStorage)
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

	// Create document row first (offline-first: server can later fill optimized_content)
	// Then process synchronously (small) or asynchronously (large).
	const asyncThreshold = 2 * 1024 * 1024 // 2MB

	// Base metadata: always include file info.
	baseMetadata := domain.DocumentMetadata{
		OriginalTitle: originalName,
		FileSize:      totalSize,
		Format:        "pdf",
		Source:        "upload",
	}

	origPath := path
	origName := originalName
	origMimeCopy := origMime
	origSize := totalSize
	origChecksumCopy := origChecksum

	doc := &domain.DocumentData{
		ID:        docID,
		UserID:    userID,
		Title:     originalName,
		Content:   json.RawMessage("[]"),
		Metadata:  baseMetadata,
		CreatedAt: now,
		UpdatedAt: now,

		OriginalStoragePath:    &origPath,
		OriginalFileName:       &origName,
		OriginalMimeType:       &origMimeCopy,
		OriginalSizeBytes:      &origSize,
		OriginalChecksumSHA256: &origChecksumCopy,

		OptimizedVersion: 1,
		ProcessingStatus: "processing",
	}

	if err := s.repo.Create(doc, token); err != nil {
		return nil, err
	}

	processAndUpdate := func(target *domain.DocumentData) {
		// Build an initial placeholder optimized array so the client can open quickly.
		// We'll fill pages as we process them and write intermediate updates.
		var optimizedPages []string
		lastIntermediateUpdatePage := 0

		blocks, pdfMetadata, err := s.pdfProcessor.ProcessPDFWithCallbacks(
			fileBytes,
			func(meta PDFMetadata) {
				// Pre-size so partial payload keeps correct total page count.
				if meta.PageCount > 0 {
					optimizedPages = make([]string, meta.PageCount)
					target.Metadata.PageCount = meta.PageCount
				}
			},
			func(pageNumber int, pageText string) {
				if optimizedPages == nil {
					optimizedPages = make([]string, 0)
				}
				idx := pageNumber - 1
				for len(optimizedPages) <= idx {
					optimizedPages = append(optimizedPages, "")
				}
				optimizedPages[idx] = pageText

				// Throttle intermediate DB writes.
				if pageNumber == 1 || pageNumber-lastIntermediateUpdatePage >= 12 {
					lastIntermediateUpdatePage = pageNumber
					if b, err := json.Marshal(optimizedPages); err == nil {
						// Update just the optimized content + status; keep content empty until the end.
						target.OptimizedContent = json.RawMessage(b)
						size := int64(len(b))
						sum := sha256.Sum256(b)
						checksum := hex.EncodeToString(sum[:])
						target.OptimizedSizeBytes = &size
						target.OptimizedChecksumSHA256 = &checksum
						target.ProcessingStatus = "processing"
						target.UpdatedAt = time.Now().UTC()
						_ = s.repo.Update(target, token)
					}
				}
			},
		)
		if err != nil {
			s.logger.Error("Failed to process PDF", err, "doc_id", docID)
			msg := err.Error()
			target.ProcessingStatus = "failed"
			target.ProcessingError = &msg
			target.UpdatedAt = time.Now().UTC()
			if err := s.repo.Update(target, token); err != nil {
				s.logger.Error("Failed to update document after processing failure", err, "doc_id", docID)
			}
			return
		}

		// Ensure optimizedPages length matches the PDF page count.
		if pdfMetadata.PageCount > 0 {
			if optimizedPages == nil {
				optimizedPages = make([]string, pdfMetadata.PageCount)
			} else if len(optimizedPages) < pdfMetadata.PageCount {
				for len(optimizedPages) < pdfMetadata.PageCount {
					optimizedPages = append(optimizedPages, "")
				}
			}
		}

		contentJSON, err := s.pdfProcessor.ConvertToJSON(blocks)
		if err != nil {
			s.logger.Error("Failed to convert blocks to JSON", err, "doc_id", docID)
			contentJSON = json.RawMessage("[]")
		}

		optimizedJSON, err := s.pdfProcessor.ConvertToOptimizedPagesJSON(blocks, pdfMetadata.PageCount)
		if err != nil {
			s.logger.Error("Failed to convert blocks to optimized pages JSON", err, "doc_id", docID)
			optimizedJSON = json.RawMessage("[]")
		}

		// Prefer PDF title when present.
		if pdfMetadata.Title != "" {
			target.Title = pdfMetadata.Title
		}
		// Author from PDF metadata.
		if pdfMetadata.Author != "" {
			a := pdfMetadata.Author
			target.Author = &a
			target.Metadata.OriginalAuthor = pdfMetadata.Author
		}
		target.Metadata.PageCount = pdfMetadata.PageCount
		target.Metadata.HasPassword = pdfMetadata.HasPassword

		// Optimized checksums/sizes
		optSize := int64(len(optimizedJSON))
		optSum := sha256.Sum256([]byte(optimizedJSON))
		optChecksum := hex.EncodeToString(optSum[:])
		target.OptimizedContent = optimizedJSON
		target.OptimizedSizeBytes = &optSize
		target.OptimizedChecksumSHA256 = &optChecksum

		target.Content = contentJSON
		processedAt := time.Now().UTC()
		target.ProcessedAt = &processedAt
		target.ProcessingStatus = "ready"
		target.ProcessingError = nil
		target.UpdatedAt = processedAt

		if err := s.repo.Update(target, token); err != nil {
			s.logger.Error("Failed to update document with processed content", err, "doc_id", docID)
			return
		}

		s.logger.Info("Document processed",
			"doc_id", docID,
			"blocks_count", len(blocks),
			"page_count", pdfMetadata.PageCount,
		)
	}

	if totalSize < asyncThreshold {
		processAndUpdate(doc)
		return doc, nil
	}

	// IMPORTANT: avoid mutating the response object after returning.
	// Create an independent copy for background processing.
	backgroundDoc := *doc
	go processAndUpdate(&backgroundDoc)
	s.logger.Info("Document created; processing in background", "doc_id", docID, "file_size", totalSize)
	return doc, nil
}
