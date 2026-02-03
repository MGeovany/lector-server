package domain

import (
	"context"
	"time"

	"github.com/pgvector/pgvector-go"
)

// DocumentPage represents a single page of text from a document.
type DocumentPage struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	PageNumber int       `json:"page_number"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

// PageEmbedding represents the vector embedding for a page (or chunk).
type PageEmbedding struct {
	ID         string          `json:"id"`
	DocumentID string          `json:"document_id"`
	PageID     string          `json:"page_id"`
	PageNumber int             `json:"page_number"`
	ChunkIndex int             `json:"chunk_index"`
	Embedding  pgvector.Vector `json:"-"` // Vector is internal, usually not sent to frontend
	CreatedAt  time.Time       `json:"created_at"`
}

// ChatSession represents a conversation thread.
type ChatSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	DocumentID *string   `json:"document_id,omitempty"` // Optional: link to specific doc
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ChatMessage represents a single message in a chat session.
type ChatMessage struct {
	ID            string    `json:"id"`
	ChatSessionID string    `json:"chat_session_id"`
	Role          string    `json:"role"` // user, model, system
	Content       string    `json:"content"`
	Citations     []int     `json:"citations,omitempty"` // Page numbers referenced
	TokenCount    int       `json:"token_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// UsageLedger tracks token usage for billing/trial limits.
type UsageLedger struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end,omitempty"`
	TokensIn     int       `json:"tokens_in"`
	TokensOut    int       `json:"tokens_out"`
	RequestCount int       `json:"request_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SearchResult represents a retrieved chunk/page with similarity score.
type SearchResult struct {
	PageID     string  `json:"page_id"`
	PageNumber int     `json:"page_number"`
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
}

// AIService defines operations for the AI feature.
type AIService interface {
	// IngestDocument processes a document: text extraction -> chunking -> embedding -> storage.
	IngestDocument(ctx context.Context, userID, documentID string, token string) error

	// Ask processes a user query: retrieval -> generation -> storage.
	Ask(ctx context.Context, userID string, req ChatRequest, token string) (*ChatResponse, error)

	// GetChatHistory retrieves a chat session with messages.
	GetChatHistory(ctx context.Context, userID, sessionID string, token string) (*ChatSessionData, error)
}

// VectorRepository defines persistence for embeddings and text retrieval.
type VectorRepository interface {
	SavePage(ctx context.Context, page *DocumentPage, token string) error
	SaveEmbedding(ctx context.Context, embedding *PageEmbedding, token string) error
	SearchSimilar(ctx context.Context, documentID string, queryVector pgvector.Vector, limit int, token string) ([]SearchResult, error)
	DeleteByDocumentID(ctx context.Context, documentID string, token string) error
	// GetPageText returns the text of the given page (1-based). Empty string if not found.
	GetPageText(ctx context.Context, documentID string, pageNumber int, token string) (string, error)
}

// ChatRepository defines persistence for chat history.
type ChatRepository interface {
	CreateSession(ctx context.Context, session *ChatSession, token string) error
	GetSession(ctx context.Context, id string, token string) (*ChatSession, error)
	CreateMessage(ctx context.Context, msg *ChatMessage, token string) error
	GetMessages(ctx context.Context, sessionID string, token string) ([]*ChatMessage, error)
}

// UsageRepository defines persistence for usage tracking.
type UsageRepository interface {
	GetUsage(ctx context.Context, userID string, periodStart time.Time, token string) (*UsageLedger, error)
	IncrementUsage(ctx context.Context, userID string, tokensIn, tokensOut int, token string) error
}

// DTOs

type ChatRequest struct {
	SessionID   string `json:"session_id,omitempty"` // If empty, create new
	DocumentID  string `json:"document_id"`
	Prompt      string `json:"prompt"`
	CurrentPage *int   `json:"current_page,omitempty"` // 1-based page the user is viewing
	TotalPages  *int   `json:"total_pages,omitempty"`
}

type ChatResponse struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Citations []int  `json:"citations,omitempty"`
}

type ChatSessionData struct {
	Session  *ChatSession   `json:"session"`
	Messages []*ChatMessage `json:"messages"`
}
