package repository

import (
	"encoding/json"
	"fmt"
	"pdf-text-reader/internal/domain"
	"regexp"
	"strings"
	"time"

	"github.com/supabase-community/postgrest-go"
)

// HighlightRepository implements the domain.HighlightRepository interface using Supabase.
type HighlightRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

func NewHighlightRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) domain.HighlightRepository {
	return &HighlightRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

func (r *HighlightRepository) Create(highlight *domain.Highlight, token string) (*domain.Highlight, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	quote := sanitizeText(highlight.Quote)

	row := map[string]interface{}{
		"user_id":     highlight.UserID,
		"document_id": highlight.DocumentID,
		"quote":       quote,
	}
	if highlight.PageNumber != nil {
		row["page_number"] = *highlight.PageNumber
	}
	if highlight.Progress != nil {
		row["progress"] = *highlight.Progress
	}

	// Request "representation" so PostgREST returns the inserted row.
	data, _, err := client.From("highlights").
		Insert(row, false, "", "representation", "").
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create highlight: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("failed to create highlight: empty response")
	}

	return mapToHighlight(rows[0]), nil
}

func (r *HighlightRepository) ListByUser(userID string, documentID *string, token string) ([]*domain.Highlight, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	q := client.From("highlights").
		Select("*", "", false).
		Eq("user_id", userID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false})

	if documentID != nil && *documentID != "" {
		q = q.Eq("document_id", *documentID)
	}

	data, _, err := q.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list highlights: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	out := make([]*domain.Highlight, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapToHighlight(row))
	}
	return out, nil
}

func (r *HighlightRepository) Delete(userID string, highlightID string, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client with token: %w", err)
	}
	if client == nil {
		return fmt.Errorf("supabase client not initialized")
	}

	_, _, err = client.From("highlights").
		Delete("", "").
		Eq("id", highlightID).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to delete highlight: %w", err)
	}
	return nil
}

func mapToHighlight(data map[string]interface{}) *domain.Highlight {
	h := &domain.Highlight{
		ID:         getString(data, "id"),
		UserID:     getString(data, "user_id"),
		DocumentID: getString(data, "document_id"),
		Quote:      getString(data, "quote"),
	}

	if pn, ok := data["page_number"]; ok && pn != nil {
		switch v := pn.(type) {
		case float64:
			val := int(v)
			h.PageNumber = &val
		case int:
			val := v
			h.PageNumber = &val
		case int64:
			val := int(v)
			h.PageNumber = &val
		}
	}

	if p, ok := data["progress"]; ok && p != nil {
		switch v := p.(type) {
		case float64:
			val := float32(v)
			h.Progress = &val
		case float32:
			val := v
			h.Progress = &val
		case int:
			val := float32(v)
			h.Progress = &val
		case int64:
			val := float32(v)
			h.Progress = &val
		}
	}

	if createdAt := getString(data, "created_at"); createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			h.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
			h.CreatedAt = t
		}
	}

	return h
}

var reControl = regexp.MustCompile(`[\x00]`)

// sanitizeText removes characters that PostgreSQL rejects in text fields (notably NUL bytes).
func sanitizeText(s string) string {
	if s == "" {
		return s
	}
	// Remove any NUL bytes.
	s = reControl.ReplaceAllString(s, "")
	// Also remove escaped unicode NUL sequences that can appear in some extracted content.
	s = strings.ReplaceAll(s, "\\u0000", "")
	s = strings.ReplaceAll(s, "\u0000", "")
	return s
}

