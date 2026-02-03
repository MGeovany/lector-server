package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"pdf-text-reader/internal/domain"

	"github.com/pgvector/pgvector-go"
	"github.com/supabase-community/postgrest-go"
)

// AIRepository implements VectorRepository, ChatRepository, and UsageRepository.
type AIRepository struct {
	supabaseClient domain.SupabaseClient
	logger         domain.Logger
}

func NewAIRepository(supabaseClient domain.SupabaseClient, logger domain.Logger) *AIRepository {
	return &AIRepository{
		supabaseClient: supabaseClient,
		logger:         logger,
	}
}

// --- VectorRepository Implementation ---

func (r *AIRepository) SavePage(ctx context.Context, page *domain.DocumentPage, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	data := map[string]interface{}{
		"document_id": page.DocumentID,
		"page_number": page.PageNumber,
		"text":        page.Text,
		"created_at":  time.Now(),
	}
	if page.ID != "" {
		data["id"] = page.ID
	}

	// Upsert on (document_id, page_number) so re-ingest (retries, concurrent requests) does not hit unique constraint.
	resp, _, err := client.From("document_pages").Insert(data, true, "document_id,page_number", "id", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to save page: %w", err)
	}

	var result []struct {
		ID string `json:"id"`
	}
	if len(resp) > 0 {
		if err := json.Unmarshal(resp, &result); err == nil && len(result) > 0 {
			page.ID = result[0].ID
			return nil
		}
	}
	// Upsert with return=minimal or UPDATE path can return empty body; fetch id by lookup.
	id, err := r.getPageID(page.DocumentID, page.PageNumber, token)
	if err != nil {
		return fmt.Errorf("failed to get page id after save: %w", err)
	}
	page.ID = id
	return nil
}

func (r *AIRepository) getPageID(documentID string, pageNumber int, token string) (string, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return "", err
	}
	resp, _, err := client.From("document_pages").Select("id", "", false).Eq("document_id", documentID).Eq("page_number", fmt.Sprint(pageNumber)).Limit(1, "").Execute()
	if err != nil {
		return "", err
	}
	var rows []struct {
		ID string `json:"id"`
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("page not found")
	}
	if err := json.Unmarshal(resp, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("page not found")
	}
	return rows[0].ID, nil
}

func (r *AIRepository) GetPageText(ctx context.Context, documentID string, pageNumber int, token string) (string, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return "", err
	}
	resp, _, err := client.From("document_pages").Select("text", "", false).Eq("document_id", documentID).Eq("page_number", fmt.Sprint(pageNumber)).Limit(1, "").Execute()
	if err != nil {
		return "", err
	}
	var rows []struct {
		Text string `json:"text"`
	}
	if len(resp) == 0 {
		return "", nil
	}
	if err := json.Unmarshal(resp, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	return rows[0].Text, nil
}

func (r *AIRepository) SaveEmbedding(ctx context.Context, embedding *domain.PageEmbedding, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	data := map[string]interface{}{
		"document_id": embedding.DocumentID,
		"page_id":     embedding.PageID,
		"page_number": embedding.PageNumber,
		"chunk_index": embedding.ChunkIndex,
		"embedding":   embedding.Embedding,
		"created_at":  time.Now(),
	}

	_, _, err = client.From("page_embeddings").Insert(data, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}
	return nil
}

func (r *AIRepository) SearchSimilar(ctx context.Context, documentID string, queryVector pgvector.Vector, limit int, token string) ([]domain.SearchResult, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	params := map[string]interface{}{
		"query_embedding":    queryVector,
		"match_threshold":    0.3,
		"match_count":        limit,
		"filter_document_id": documentID,
	}

	// Rpc returns a string, we don't need to call Execute()
	// NOTE: supabase-go v0.0.4 Rpc returns just string. If error occurs, it might return empty string or panic?
	// Unfortunately invalid signature in library or my understanding.
	// The user reported: assignment mismatch: 2 variables but client.Rpc returns 1 value.
	resp := client.Rpc("match_page_embeddings", "", params)

	// We assume empty response might mean error or empty list?
	// Typically libraries returning 1 value might panic on error or we check if response is valid JSON.
	if resp == "" {
		return nil, fmt.Errorf("rpc returned empty response")
	}

	var results []struct {
		PageID     string  `json:"page_id"`
		PageNumber int     `json:"page_number"`
		ChunkIndex int     `json:"chunk_index"`
		Text       string  `json:"text"`
		Similarity float32 `json:"similarity"`
	}

	if err := json.Unmarshal([]byte(resp), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	domainResults := make([]domain.SearchResult, len(results))
	for i, res := range results {
		domainResults[i] = domain.SearchResult{
			PageID:     res.PageID,
			PageNumber: res.PageNumber,
			Text:       res.Text,
			Score:      res.Similarity,
		}
	}

	return domainResults, nil
}

func (r *AIRepository) DeleteByDocumentID(ctx context.Context, documentID string, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	_, _, err = client.From("page_embeddings").Delete("", "").Eq("document_id", documentID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}
	_, _, err = client.From("document_pages").Delete("", "").Eq("document_id", documentID).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete pages: %w", err)
	}
	return nil
}

// --- ChatRepository Implementation ---

func (r *AIRepository) CreateSession(ctx context.Context, session *domain.ChatSession, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	data := map[string]interface{}{
		"user_id":    session.UserID,
		"title":      session.Title,
		"created_at": session.CreatedAt,
		"updated_at": session.UpdatedAt,
	}
	if session.DocumentID != nil {
		data["document_id"] = *session.DocumentID
	}

	resp, _, err := client.From("chat_sessions").Insert(data, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to create chat session: %w", err)
	}

	var result []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result) > 0 {
		session.ID = result[0].ID
	}
	return nil
}

func (r *AIRepository) GetSession(ctx context.Context, id string, token string) (*domain.ChatSession, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	resp, _, err := client.From("chat_sessions").Select("*", "", false).Eq("id", id).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var sessions []domain.ChatSession
	if err := json.Unmarshal(resp, &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("session not found")
	}
	return &sessions[0], nil
}

func (r *AIRepository) CreateMessage(ctx context.Context, msg *domain.ChatMessage, token string) error {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	data := map[string]interface{}{
		"chat_session_id": msg.ChatSessionID,
		"role":            msg.Role,
		"content":         msg.Content,
		"token_count":     msg.TokenCount,
		"created_at":      msg.CreatedAt,
	}

	if len(msg.Citations) > 0 {
		citationsJSON, _ := json.Marshal(msg.Citations)
		data["citations"] = string(citationsJSON)
	} else {
		data["citations"] = "[]"
	}

	resp, _, err := client.From("chat_messages").Insert(data, false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	var result []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result) > 0 {
		msg.ID = result[0].ID
	}
	return nil
}

func (r *AIRepository) GetMessages(ctx context.Context, sessionID string, token string) ([]*domain.ChatMessage, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	resp, _, err := client.From("chat_messages").
		Select("*", "", false).
		Eq("chat_session_id", sessionID).
		Order("created_at", &postgrest.OrderOpts{Ascending: true}).
		Execute()

	// Wait, domain.OrderOptions might be undefined.
	// I'll check if I need to use raw string for order: "created_at.asc"
	// Supabase-go Order method takes: column string, opts *OrderOptions.
	// If domain.OrderOptions is not defined, I should import supabase-go directly or just use default.
	// But `repository` imports `domain`. `domain` imports `supabase-go`.
	// domain/supabase.go imports supabase-go.
	// `domain` package doesn't expose `OrderOptions` unless I added it.
	// I'll check `domain/supabase.go` again. It only showed interface.
	// So I should import `github.com/supabase-community/postgrest-go` or `supabase-go` to use `OrderOptions` OR just pass `nil` if I can't sort?
	// But I need sorting.
	// `Order` signature: `Order(column string, opts *OrderOpts)`.
	// I'll assume I can't use `domain.OrderOptions` as it doesn't exist.
	// I will remove the `Order` call for now and sort in memory, or use `Order("created_at", nil)` if default is asc? Default is prob not guaranteed.
	// Better: Use `Order("created_at", &postgrest.OrderOpts{Ascending: true})`.
	// I need to import `github.com/supabase-community/postgrest-go` as `postgrest`.
	// I'll add the import.

	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var messages []*domain.ChatMessage
	if err := json.Unmarshal(resp, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return messages, nil
}

// --- UsageRepository Implementation ---

func (r *AIRepository) GetUsage(ctx context.Context, userID string, periodStart time.Time, token string) (*domain.UsageLedger, error) {
	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	encodedTime := periodStart.Format(time.RFC3339)
	resp, _, err := client.From("usage_ledger").
		Select("*", "", false).
		Eq("user_id", userID).
		Eq("period_start", encodedTime).
		Limit(1, "").
		Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	var ledgers []domain.UsageLedger
	if err := json.Unmarshal(resp, &ledgers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(ledgers) == 0 {
		return nil, nil
	}
	return &ledgers[0], nil
}

func (r *AIRepository) IncrementUsage(ctx context.Context, userID string, tokensIn, tokensOut int, token string) error {
	// MVP: Read-Modify-Write.
	// Production: Use RPC function for atomic increment.

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	ledger, err := r.GetUsage(ctx, userID, periodStart, token)
	if err != nil {
		return fmt.Errorf("failed to get usage for increment: %w", err)
	}

	client, err := r.supabaseClient.GetClientWithToken(token)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if ledger == nil {
		// Insert new
		data := map[string]interface{}{
			"user_id":            userID,
			"period_start":       periodStart,
			"tokens_used_input":  tokensIn,
			"tokens_used_output": tokensOut,
			"created_at":         now,
			"updated_at":         now,
		}
		_, _, err := client.From("usage_ledger").Insert(data, false, "", "", "").Execute()
		if err != nil {
			return fmt.Errorf("failed to create ledger: %w", err)
		}
	} else {
		// Update existing
		data := map[string]interface{}{
			"tokens_used_input":  ledger.TokensIn + tokensIn,
			"tokens_used_output": ledger.TokensOut + tokensOut,
			"updated_at":         now,
		}
		_, _, err := client.From("usage_ledger").Update(data, "", "").Eq("id", ledger.ID).Execute()
		if err != nil {
			return fmt.Errorf("failed to update ledger: %w", err)
		}
	}

	return nil
}
