package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"pdf-text-reader/internal/domain"

	"cloud.google.com/go/vertexai/genai"
	"github.com/pgvector/pgvector-go"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
)

type AIService struct {
	vectorRepo domain.VectorRepository
	chatRepo   domain.ChatRepository
	docRepo    domain.DocumentRepository
	usageRepo  domain.UsageRepository
	prefsRepo  domain.UserPreferencesRepository
	logger     domain.Logger

	projectID string
	location  string

	genaiClient *genai.Client
}

func NewAIService(
	vectorRepo domain.VectorRepository,
	chatRepo domain.ChatRepository,
	docRepo domain.DocumentRepository,
	usageRepo domain.UsageRepository,
	prefsRepo domain.UserPreferencesRepository,
	logger domain.Logger,
	projectID string,
	location string,
) (*AIService, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to create vertex ai client: %w", err)
	}

	return &AIService{
		vectorRepo:  vectorRepo,
		chatRepo:    chatRepo,
		docRepo:     docRepo,
		usageRepo:   usageRepo,
		prefsRepo:   prefsRepo,
		logger:      logger,
		projectID:   projectID,
		location:    location,
		genaiClient: client,
	}, nil
}

const (
	ingestPageWorkers  = 15 // max concurrent SavePage calls to Supabase
	ingestEmbedWorkers = 8  // max concurrent Vertex AI embedding calls
)

func (s *AIService) IngestDocument(ctx context.Context, userID, documentID string, token string) error {
	if err := s.ensureAskAIEntitled(ctx, userID, token); err != nil {
		return err
	}

	optDoc, err := s.docRepo.GetOptimizedByID(documentID, token)
	if err != nil {
		return fmt.Errorf("failed to get optimized document: %w", err)
	}
	if optDoc == nil || len(optDoc.Pages) == 0 {
		return fmt.Errorf("document has no content to ingest")
	}

	if err := s.vectorRepo.DeleteByDocumentID(ctx, documentID, token); err != nil {
		s.logger.Warn("Failed to clean up old embeddings", "error", err, "doc_id", documentID)
	}

	type pageJob struct {
		pageNumber int
		pageText   string
	}
	var jobs []pageJob
	for i, pageText := range optDoc.Pages {
		if strings.TrimSpace(pageText) == "" {
			continue
		}
		jobs = append(jobs, pageJob{pageNumber: i + 1, pageText: pageText})
	}

	// Phase 1: save all pages in parallel with limited concurrency.
	pagesByNumber := make(map[int]*domain.DocumentPage)
	var pagesMu sync.Mutex
	pageSem := make(chan struct{}, ingestPageWorkers)
	g1, ctx1 := errgroup.WithContext(ctx)
	for _, j := range jobs {
		j := j
		g1.Go(func() error {
			select {
			case pageSem <- struct{}{}:
				defer func() { <-pageSem }()
			case <-ctx1.Done():
				return ctx1.Err()
			}
			page := &domain.DocumentPage{
				DocumentID: documentID,
				PageNumber: j.pageNumber,
				Text:       j.pageText,
			}
			if err := s.vectorRepo.SavePage(ctx1, page, token); err != nil {
				s.logger.Error("Failed to save page", err, "doc_id", documentID, "page", j.pageNumber)
				return nil // continue with others
			}
			pagesMu.Lock()
			pagesByNumber[j.pageNumber] = page
			pagesMu.Unlock()
			return nil
		})
	}
	if err := g1.Wait(); err != nil {
		return err
	}

	// Phase 2: generate embeddings and save in parallel, with limited concurrency for Vertex.
	sem := make(chan struct{}, ingestEmbedWorkers)
	g2, ctx2 := errgroup.WithContext(ctx)
	for _, j := range jobs {
		j := j
		page := pagesByNumber[j.pageNumber]
		if page == nil || page.ID == "" {
			continue
		}
		g2.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx2.Done():
				return ctx2.Err()
			}
			embedding, err := s.generateEmbedding(ctx2, j.pageText)
			if err != nil {
				s.logger.Error("Failed to generate embedding", err, "doc_id", documentID, "page", j.pageNumber)
				return nil
			}
			if len(embedding) == 0 {
				return nil
			}
			vec := pgvector.NewVector(embedding)
			emb := &domain.PageEmbedding{
				DocumentID: documentID,
				PageID:     page.ID,
				PageNumber: j.pageNumber,
				ChunkIndex: 0,
				Embedding:  vec,
			}
			if err := s.vectorRepo.SaveEmbedding(ctx2, emb, token); err != nil {
				s.logger.Error("Failed to save embedding", err, "doc_id", documentID, "page", j.pageNumber)
			}
			return nil
		})
	}
	return g2.Wait()
}

func (s *AIService) Ask(ctx context.Context, userID string, req domain.ChatRequest, token string) (*domain.ChatResponse, error) {
	limit, used, err := s.ensureAskAIWithinQuota(ctx, userID, token)
	if err != nil {
		return nil, err
	}

	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID
		sess, err := s.chatRepo.GetSession(ctx, sessionID, token)
		if err != nil {
			return nil, fmt.Errorf("invalid session: %w", err)
		}
		if sess.UserID != userID {
			return nil, fmt.Errorf("access denied")
		}
	} else {
		newSess := &domain.ChatSession{
			UserID:     userID,
			DocumentID: nil,
			Title:      "New Chat",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if req.DocumentID != "" {
			newSess.DocumentID = &req.DocumentID
			newSess.Title = "Chat about document"
		}
		if err := s.chatRepo.CreateSession(ctx, newSess, token); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = newSess.ID
	}

	userMsg := &domain.ChatMessage{
		ChatSessionID: sessionID,
		Role:          "user",
		Content:       req.Prompt,
		CreatedAt:     time.Now(),
	}
	if err := s.chatRepo.CreateMessage(ctx, userMsg, token); err != nil {
		s.logger.Warn("Failed to log user message", "error", err)
	}

	embedding, err := s.generateEmbedding(ctx, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	queryVector := pgvector.NewVector(embedding)

	topK := 5
	searchResults, err := s.vectorRepo.SearchSimilar(ctx, req.DocumentID, queryVector, topK, token)
	if err != nil {
		s.logger.Warn("Vector search failed", "error", err)
	}

	var contextBuilder strings.Builder
	citations := make([]int, 0, len(searchResults))

	// If the user is on a specific page, include that page's content first so the model can answer "what is this page about?"
	if req.CurrentPage != nil && *req.CurrentPage > 0 {
		currentPageText, err := s.vectorRepo.GetPageText(ctx, req.DocumentID, *req.CurrentPage, token)
		if err == nil && strings.TrimSpace(currentPageText) != "" {
			contextBuilder.WriteString("Current page the user is viewing (page ")
			contextBuilder.WriteString(fmt.Sprint(*req.CurrentPage))
			contextBuilder.WriteString("):\n")
			contextBuilder.WriteString(currentPageText)
			contextBuilder.WriteString("\n\n---------------------\n")
			citations = append(citations, *req.CurrentPage)
		}
		if req.TotalPages != nil && *req.TotalPages > 0 {
			contextBuilder.WriteString(fmt.Sprintf("The user is currently viewing page %d of %d.\n", *req.CurrentPage, *req.TotalPages))
		} else {
			contextBuilder.WriteString(fmt.Sprintf("The user is currently viewing page %d.\n", *req.CurrentPage))
		}
	}

	contextBuilder.WriteString("Additional context from the document:\n---------------------\n")
	for _, result := range searchResults {
		// Skip if we already included this page as the current page
		if req.CurrentPage != nil && result.PageNumber == *req.CurrentPage {
			continue
		}
		contextBuilder.WriteString(fmt.Sprintf("Page %d: %s\n\n", result.PageNumber, result.Text))
		citations = append(citations, result.PageNumber)
	}
	contextBuilder.WriteString("---------------------\n")
	contextBuilder.WriteString("RULES: Answer the user's question using ONLY the document context above. ")
	contextBuilder.WriteString("Allowed questions include: what the document is about, summary, main topic, themes, specific passages, characters, plot, or any question that can be answered from the text. ")
	contextBuilder.WriteString("Only refuse if the question is clearly unrelated (e.g. coding, math, other books, or topics that cannot be answered from this document). ")
	contextBuilder.WriteString("If you must refuse, say: \"I can only answer questions about this document. Please ask something related to the text you're reading.\" ")
	contextBuilder.WriteString("Do not write code, role-play, or use outside knowledge. If the context is empty, say you don't have enough of the document to answer.\n")

	model := s.genaiClient.GenerativeModel("gemini-2.0-flash-001")
	model.SetTemperature(0.5)

	chat := model.StartChat()

	historyMsgs, err := s.chatRepo.GetMessages(ctx, sessionID, token)
	if err == nil {
		for _, m := range historyMsgs {
			role := "user"
			if m.Role == "model" {
				role = "model"
			}
			if m.Content == req.Prompt && m.Role == "user" {
				// avoid dup if we just inserted? No, query reads from db, but we just inserted.
				// Chat repo usually doesn't have read-after-write consistency guaranteed if async?
				// But here we might see it.
				// If we see it, we skip duplicate of LAST message or current message?
				// Better to trust GetMessages and if it includes current, great.
				// But we are building history for context. Current message is added via SendMessage.
				// So we should EXCLUDE current message from history passed to StartChat logic (if we were using history manually).
				// StartChat() creates empty history.
				// We populate chat.History.
				// We should populate history up to BEFORE current message.
				// If GetMessages includes current message, we skip it.
				// How to identify? ID? Content match at end?
				// Simple heuristic: if m.ID == userMsg.ID then skip.
				if m.ID == userMsg.ID {
					continue
				}
			}

			chat.History = append(chat.History, &genai.Content{
				Role: role,
				Parts: []genai.Part{
					genai.Text(m.Content),
				},
			})
		}
	}

	finalPrompt := contextBuilder.String() + "\nQuery: " + req.Prompt
	resp, err := chat.SendMessage(ctx, genai.Text(finalPrompt))
	if err != nil {
		return nil, fmt.Errorf("gemini call failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			sb.WriteString(string(t))
		}
	}
	answer := sb.String()
	tokenCount := int(resp.UsageMetadata.TotalTokenCount)

	modelMsg := &domain.ChatMessage{
		ChatSessionID: sessionID,
		Role:          "model",
		Content:       answer,
		Citations:     citations,
		TokenCount:    tokenCount,
		CreatedAt:     time.Now(),
	}
	if err := s.chatRepo.CreateMessage(ctx, modelMsg, token); err != nil {
		s.logger.Warn("Failed to save model response", "error", err)
	}

	_ = s.usageRepo.IncrementUsage(ctx, userID, int(resp.UsageMetadata.PromptTokenCount), int(resp.UsageMetadata.CandidatesTokenCount), token)

	// Best-effort logging if the request pushed user over quota (we gate before the call,
	// but the model can still use variable tokens).
	if limit > 0 {
		reqTokens := int(resp.UsageMetadata.PromptTokenCount) + int(resp.UsageMetadata.CandidatesTokenCount)
		if used+reqTokens > limit {
			s.logger.Warn("Ask AI exceeded monthly quota (post-call)", "user_id", userID, "limit", limit, "used_before", used, "used_after", used+reqTokens)
		}
	}

	return &domain.ChatResponse{
		SessionID: sessionID,
		Message:   answer,
		Citations: citations,
	}, nil
}

func (s *AIService) GetChatHistory(ctx context.Context, userID, sessionID string, token string) (*domain.ChatSessionData, error) {
	if err := s.ensureAskAIEntitled(ctx, userID, token); err != nil {
		return nil, err
	}

	sess, err := s.chatRepo.GetSession(ctx, sessionID, token)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	msgs, err := s.chatRepo.GetMessages(ctx, sessionID, token)
	if err != nil {
		return nil, err
	}

	return &domain.ChatSessionData{
		Session:  sess,
		Messages: msgs,
	}, nil
}

func (s *AIService) ensureAskAIEntitled(ctx context.Context, userID string, token string) error {
	if s.prefsRepo == nil {
		return fmt.Errorf("preferences repository not configured")
	}
	prefs, err := s.prefsRepo.GetPreferences(userID, token)
	if err != nil {
		return fmt.Errorf("failed to load user preferences: %w", err)
	}
	plan := "free"
	if prefs != nil && prefs.SubscriptionPlan != "" {
		plan = prefs.SubscriptionPlan
	}
	if !domain.AskAIEnabledForPlan(plan) {
		return domain.ErrPlanUpgradeRequired
	}
	return nil
}

// ensureAskAIWithinQuota validates entitlement and monthly token quota.
// Returns (monthlyLimit, tokensUsedSoFar, error).
func (s *AIService) ensureAskAIWithinQuota(ctx context.Context, userID string, token string) (int, int, error) {
	if err := s.ensureAskAIEntitled(ctx, userID, token); err != nil {
		return 0, 0, err
	}

	// We currently use the same quota for Pro/Founder.
	// If we later add per-plan quotas, compute from plan here.
	limit := domain.MonthlyAITokenLimitForPlan("pro_monthly")
	if limit <= 0 {
		return 0, 0, domain.ErrPlanUpgradeRequired
	}

	if s.usageRepo == nil {
		return limit, 0, fmt.Errorf("usage repository not configured")
	}

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	ledger, err := s.usageRepo.GetUsage(ctx, userID, periodStart, token)
	if err != nil {
		return limit, 0, fmt.Errorf("failed to load usage: %w", err)
	}

	used := 0
	if ledger != nil {
		used = ledger.TokensIn + ledger.TokensOut
	}

	if used >= limit {
		return limit, used, domain.ErrMonthlyTokenLimitHit
	}

	return limit, used, nil
}

// generateEmbedding generates embeddings using Vertex AI REST API manually
func (s *AIService) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Endpoint: https://{LOCATION}-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/{LOCATION}/publishers/google/models/{MODEL}:predict
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/text-embedding-004:predict", s.location, s.projectID, s.location)

	requestBody := map[string]interface{}{
		"instances": []map[string]interface{}{
			{"content": text},
		},
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// Get default credentials
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to get default credentials: %w", err)
	}

	// Create token source
	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Predictions []struct {
			Embeddings struct {
				Values []float32 `json:"values"`
			} `json:"embeddings"`
		} `json:"predictions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Predictions) == 0 || len(result.Predictions[0].Embeddings.Values) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Predictions[0].Embeddings.Values, nil
}
