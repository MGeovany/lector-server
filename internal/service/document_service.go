package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
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

	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = ".pdf"
	}
	format := strings.TrimPrefix(ext, ".")
	if format == "markdown" {
		format = "md"
	}
	// Path should be relative to bucket, not include bucket name
	path := fmt.Sprintf("%s/%s%s", userID, docID, ext)

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

	// Enforce per-user storage quota BEFORE uploading to storage
	// Get current documents to calculate total storage used
	existingDocs, err := s.repo.GetByUserID(userID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current storage usage: %w", err)
	}

	var currentUsage int64
	for _, d := range existingDocs {
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
		originalName = docID + ext
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
			metadata = domain.DocumentMetadata{
				OriginalTitle: originalName,
				FileSize:      totalSize,
				Format:        format,
				Source:        "upload",
			}
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
				OriginalTitle:  originalName,
				OriginalAuthor: pdfMetadata.Author,
				PageCount:      pdfMetadata.PageCount,
				HasPassword:    pdfMetadata.HasPassword,
				FileSize:       totalSize,
				Format:         "pdf",
				Source:         "upload",
			}

			s.logger.Info("DocumentData processed synchronously",
				"doc_id", docID,
				"blocks_count", len(blocks),
				"page_count", pdfMetadata.PageCount,
			)
		}
	} else {
		// For larger files, create document with placeholder; a single background worker (started after Create) will run processAndUpdate
		contentJSON = json.RawMessage("[]")
		metadata = domain.DocumentMetadata{
			OriginalTitle: originalName,
			FileSize:      totalSize,
			Format:        format,
			Source:        "upload",
		}
	}

	// Set author from PDF metadata if available, otherwise leave nil
	var author *string
	if metadata.OriginalAuthor != "" {
		author = &metadata.OriginalAuthor
	}

	// Ensure metadata includes file information
	if metadata.FileSize == 0 {
		metadata.FileSize = totalSize
	}
	if metadata.OriginalTitle == "" {
		metadata.OriginalTitle = originalName
	}
	if metadata.Format == "" {
		metadata.Format = "pdf"
	}

	doc := &domain.DocumentData{
		ID:        docID,
		UserID:    userID,
		Title:     title,
		Author:    author,
		Content:   contentJSON,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(doc, token); err != nil {
		return nil, err
	}

	processAndUpdate := func(target *domain.DocumentData) {
		// Prevent any panic from silently leaving the document stuck at "processing".
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("panic during processing: %v", r)
				s.logger.Error("Panic recovered in processAndUpdate", nil, "doc_id", docID, "panic", r)
				target.ProcessingStatus = "failed"
				target.ProcessingError = &errMsg
				target.UpdatedAt = time.Now().UTC()
				if updateErr := s.repo.Update(target, token); updateErr != nil {
					s.logger.Error("Failed to update document after panic recovery", updateErr, "doc_id", docID)
				}
			}
		}()

		switch format {
		case "pdf":
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

					target.Metadata.ProcessedPages = pageNumber

					// Throttle intermediate DB writes (fewer writes = less DB/CPU; client still gets partial quickly).
					if pageNumber == 1 || pageNumber-lastIntermediateUpdatePage >= 20 {
						lastIntermediateUpdatePage = pageNumber
						if b, err := json.Marshal(optimizedPages); err == nil {
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

			// Use the optimizedPages we built in the callback; no second pass over blocks.
			optimizedJSON, err := json.Marshal(optimizedPages)
			if err != nil {
				s.logger.Error("Failed to marshal optimized pages", err, "doc_id", docID)
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

			// Optimized checksums/sizes (reuse marshaled optimizedJSON)
			optSize := int64(len(optimizedJSON))
			optSum := sha256.Sum256(optimizedJSON)
			optChecksum := hex.EncodeToString(optSum[:])
			target.OptimizedContent = json.RawMessage(optimizedJSON)
			target.OptimizedSizeBytes = &optSize
			target.OptimizedChecksumSHA256 = &optChecksum

			target.Content = contentJSON
			processedAt := time.Now().UTC()
			target.ProcessedAt = &processedAt
			target.ProcessingStatus = "ready"
			target.ProcessingError = nil
			target.UpdatedAt = processedAt

			// Retry the final update up to 3 times with backoff to avoid stuck-at-processing documents.
			const maxRetries = 3
			var updateErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				if attempt > 0 {
					time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				}
				if updateErr = s.repo.Update(target, token); updateErr == nil {
					break
				}
				s.logger.Error(fmt.Sprintf("Failed to update document with processed content (attempt %d/%d)", attempt+1, maxRetries), updateErr, "doc_id", docID)
			}
			if updateErr != nil {
				s.logger.Error("All retries exhausted for document processing update", updateErr, "doc_id", docID)
				msg := updateErr.Error()
				target.ProcessingError = &msg
				target.ProcessingStatus = "failed"
				_ = s.repo.Update(target, token)
				return
			}

			readyPages := 0
			for _, s := range optimizedPages {
				if strings.TrimSpace(s) != "" {
					readyPages++
				}
			}
			s.logger.Info("[Doc] Document processed",
				"doc_id", docID,
				"blocks_count", len(blocks),
				"page_count", pdfMetadata.PageCount,
				"pages_with_text", readyPages,
			)
			if pdfMetadata.PageCount > 0 && readyPages == 0 {
				s.logger.Warn("[Doc] Document has no extractable text; likely scanned/image-only PDF", "doc_id", docID, "page_count", pdfMetadata.PageCount)
			}
			return

		case "txt", "md", "epub":
			extracted, err := ExtractTextDocument(format, originalName, fileBytes)
			if err != nil {
				s.logger.Error("Failed to extract text document", err, "doc_id", docID, "format", format)
				msg := err.Error()
				target.ProcessingStatus = "failed"
				target.ProcessingError = &msg
				target.UpdatedAt = time.Now().UTC()
				_ = s.repo.Update(target, token)
				return
			}

			blocks, pageCount, wordCount := BuildTextBlocksFromText(s.pdfProcessor, extracted.Text)
			contentJSON, err := s.pdfProcessor.ConvertToJSON(blocks)
			if err != nil {
				s.logger.Error("Failed to convert blocks to JSON", err, "doc_id", docID)
				contentJSON = json.RawMessage("[]")
			}

			optimizedJSON, err := s.pdfProcessor.ConvertToOptimizedPagesJSON(blocks, pageCount)
			if err != nil {
				s.logger.Error("Failed to convert blocks to optimized pages JSON", err, "doc_id", docID)
				optimizedJSON = json.RawMessage("[]")
			}

			if extracted.Title != "" {
				target.Title = extracted.Title
			}
			if extracted.Author != "" {
				a := extracted.Author
				target.Author = &a
				target.Metadata.OriginalAuthor = extracted.Author
			}
			target.Metadata.PageCount = pageCount
			target.Metadata.WordCount = wordCount
			target.Metadata.HasPassword = false

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

			// Retry the final update up to 3 times with backoff to avoid stuck-at-processing documents.
			const maxRetriesTxt = 3
			var updateErrTxt error
			for attempt := 0; attempt < maxRetriesTxt; attempt++ {
				if attempt > 0 {
					time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				}
				if updateErrTxt = s.repo.Update(target, token); updateErrTxt == nil {
					break
				}
				s.logger.Error(fmt.Sprintf("Failed to update document with processed content (attempt %d/%d)", attempt+1, maxRetriesTxt), updateErrTxt, "doc_id", docID)
			}
			if updateErrTxt != nil {
				s.logger.Error("All retries exhausted for document processing update", updateErrTxt, "doc_id", docID)
				msg := updateErrTxt.Error()
				target.ProcessingError = &msg
				target.ProcessingStatus = "failed"
				_ = s.repo.Update(target, token)
				return
			}

			s.logger.Info("Document processed",
				"doc_id", docID,
				"blocks_count", len(blocks),
				"page_count", pageCount,
				"format", format,
			)
			return

		default:
			msg := fmt.Sprintf("unsupported format: %s", format)
			target.ProcessingStatus = "failed"
			target.ProcessingError = &msg
			target.UpdatedAt = time.Now().UTC()
			_ = s.repo.Update(target, token)
			return
		}
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

// GetOptimizedDocument returns the optimized document with pages for offline-first clients.
func (s *DocumentService) GetOptimizedDocument(documentID string, token string) (*domain.OptimizedDocument, error) {
	return s.getOptimizedDocument(documentID, token, true)
}

// GetOptimizedDocumentMeta returns optimized document metadata without page content.
func (s *DocumentService) GetOptimizedDocumentMeta(documentID string, token string) (*domain.OptimizedDocument, error) {
	return s.getOptimizedDocument(documentID, token, false)
}

func (s *DocumentService) getOptimizedDocument(documentID string, token string, includePages bool) (*domain.OptimizedDocument, error) {
	doc, err := s.repo.GetByID(documentID, token)
	if err != nil {
		return nil, err
	}
	opt := &domain.OptimizedDocument{
		DocumentID:         doc.ID,
		UserID:             doc.UserID,
		OptimizedVersion:   domain.CurrentOptimizedPayloadVersion,
		OptimizedSizeBytes: doc.OptimizedSizeBytes,
		ProcessingStatus:   doc.ProcessingStatus,
		ProcessedAt:        doc.ProcessedAt,
	}
	if doc.OptimizedChecksumSHA256 != nil {
		opt.OptimizedChecksumSHA256 = doc.OptimizedChecksumSHA256
		// Backwards-compatible alias.
		opt.OptimizedChecksumSHA = doc.OptimizedChecksumSHA256
	}
	if includePages && len(doc.OptimizedContent) > 0 {
		var pages []string
		if err := json.Unmarshal(doc.OptimizedContent, &pages); err == nil {
			opt.Pages = pages
		}
	}
	return opt, nil
}
