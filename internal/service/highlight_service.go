package service

import (
	"fmt"
	"pdf-text-reader/internal/domain"
	"time"
)

type HighlightService struct {
	repo   domain.HighlightRepository
	logger domain.Logger
}

func NewHighlightService(repo domain.HighlightRepository, logger domain.Logger) domain.HighlightService {
	return &HighlightService{
		repo:   repo,
		logger: logger,
	}
}

func (s *HighlightService) CreateHighlight(userID string, highlight *domain.Highlight, token string) (*domain.Highlight, error) {
	if highlight == nil {
		return nil, fmt.Errorf("highlight is required")
	}
	highlight.UserID = userID
	if highlight.DocumentID == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if highlight.Quote == "" {
		return nil, fmt.Errorf("quote is required")
	}
	// created_at is assigned by DB; keep a local value for logging if missing.
	if highlight.CreatedAt.IsZero() {
		highlight.CreatedAt = time.Now()
	}

	created, err := s.repo.Create(highlight, token)
	if err != nil {
		return nil, err
	}
	s.logger.Info("Highlight created", "user_id", userID, "document_id", highlight.DocumentID, "highlight_id", created.ID)
	return created, nil
}

func (s *HighlightService) ListHighlights(userID string, documentID *string, token string) ([]*domain.Highlight, error) {
	return s.repo.ListByUser(userID, documentID, token)
}

func (s *HighlightService) DeleteHighlight(userID string, highlightID string, token string) error {
	if highlightID == "" {
		return fmt.Errorf("highlight_id is required")
	}
	return s.repo.Delete(userID, highlightID, token)
}

