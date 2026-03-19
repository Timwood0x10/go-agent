// Package services provides retrieval services for the storage system.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"goagent/internal/core/errors"
	"goagent/internal/storage/postgres"
	"goagent/internal/storage/postgres/embedding"
	"goagent/internal/storage/postgres/repositories"
)

// SearchRequest represents a search request with configuration.
type SearchRequest struct {
	Query       string          `json:"query"`           // Search query text
	TenantID    string          `json:"tenant_id"`       // Tenant ID for isolation
	TopK        int             `json:"top_k"`           // Number of results to return
	MinScore    float64         `json:"min_score"`       // Minimum similarity score
	Plan        *RetrievalPlan  `json:"plan"`            // Retrieval strategy
	EnableTrace bool            `json:"enable_trace"`    // Enable trace logging
	Trace       *RetrievalTrace `json:"trace,omitempty"` // Trace information
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Score     float64                `json:"score"`
	Source    string                 `json:"source"`   // knowledge, experience, tool, task_result
	Type      string                 `json:"type"`     // Result type for filtering
	Metadata  map[string]interface{} `json:"metadata"` // Additional metadata
	CreatedAt time.Time              `json:"created_at"`
}

// RetrievalPlan defines the retrieval strategy for multi-source search.
type RetrievalPlan struct {
	SearchKnowledge   bool `json:"search_knowledge"`    // Search in knowledge base
	SearchExperience  bool `json:"search_experience"`   // Search in experiences
	SearchTools       bool `json:"search_tools"`        // Search in tools
	SearchTaskResults bool `json:"search_task_results"` // Search in task results

	KnowledgeWeight   float64 `json:"knowledge_weight"`    // Weight for knowledge results (default 0.4)
	ExperienceWeight  float64 `json:"experience_weight"`   // Weight for experience results (default 0.3)
	ToolsWeight       float64 `json:"tools_weight"`        // Weight for tool results (default 0.2)
	TaskResultsWeight float64 `json:"task_results_weight"` // Weight for task result results (default 0.1)

	EnableQueryRewrite  bool `json:"enable_query_rewrite"`  // Enable query rewriting
	EnableKeywordSearch bool `json:"enable_keyword_search"` // Enable keyword/BM25 search
	EnableTimeDecay     bool `json:"enable_time_decay"`     // Enable time-based scoring decay

	TopK int `json:"top_k"` // Maximum results per source
}

// RetrievalTrace contains debugging information for retrieval operations.
type RetrievalTrace struct {
	OriginalQuery   string         `json:"original_query"`
	RewrittenQuery  string         `json:"rewritten_query"`
	RewriteUsed     bool           `json:"rewrite_used"`
	VectorResults   int            `json:"vector_results"`
	KeywordResults  int            `json:"keyword_results"`
	FinalResults    int            `json:"final_results"`
	ExecutionTime   time.Duration  `json:"execution_time"`
	VectorError     error          `json:"vector_error,omitempty"`
	SearchBreakdown map[string]int `json:"search_breakdown,omitempty"` // Results per source
}

// RetrievalService provides intelligent retrieval across multiple data sources.
// It implements hybrid search (vector + keyword), query rewriting, and time-based decay.
type RetrievalService struct {
	db              *postgres.Pool
	embeddingClient *embedding.EmbeddingClient
	tenantGuard     *postgres.TenantGuard
	retrievalGuard  *postgres.RetrievalGuard
	kbRepo          *repositories.KnowledgeRepository
	logger          *slog.Logger
}

// NewRetrievalService creates a new RetrievalService instance.
// Args:
// pool - database connection pool.
// embeddingClient - embedding service client for vector search.
// tenantGuard - tenant isolation guard.
// retrievalGuard - rate limiting and circuit breaker for retrieval.
// kbRepo - knowledge repository for data access.
// Returns new RetrievalService instance.
func NewRetrievalService(
	pool *postgres.Pool,
	embeddingClient *embedding.EmbeddingClient,
	tenantGuard *postgres.TenantGuard,
	retrievalGuard *postgres.RetrievalGuard,
	kbRepo *repositories.KnowledgeRepository,
) *RetrievalService {
	return &RetrievalService{
		db:              pool,
		embeddingClient: embeddingClient,
		tenantGuard:     tenantGuard,
		retrievalGuard:  retrievalGuard,
		kbRepo:          kbRepo,
		logger:          slog.Default(),
	}
}

// DefaultRetrievalPlan returns the default retrieval plan.
func DefaultRetrievalPlan() *RetrievalPlan {
	return &RetrievalPlan{
		SearchKnowledge:     true,
		SearchExperience:    false, // TODO: Implement when ExperienceRepository is available
		SearchTools:         false, // TODO: Implement when ToolRepository is available
		SearchTaskResults:   false,
		KnowledgeWeight:     1.0, // Only knowledge is enabled
		ExperienceWeight:    0.0,
		ToolsWeight:         0.0,
		TaskResultsWeight:   0.0,
		EnableQueryRewrite:  false,
		EnableKeywordSearch: true,
		EnableTimeDecay:     true,
		TopK:                10,
	}
}

// Search performs intelligent retrieval across multiple data sources.
// This implements the core retrieval pipeline with hybrid search, query rewriting, and time decay.
// Args:
// ctx - database operation context.
// req - search request with query and configuration.
// Returns search results or error if retrieval fails.
func (s *RetrievalService) Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error) {
	startTime := time.Now()

	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, err
	}

	// Set default plan if not provided
	if req.Plan == nil {
		req.Plan = DefaultRetrievalPlan()
	}

	// Apply tenant isolation
	if err := s.tenantGuard.SetTenantContext(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	// Check rate limiting and circuit breaker
	if err := s.retrievalGuard.AllowRateLimit(); err != nil {
		return nil, err
	}

	// 1. Optional Query Rewrite
	originalQuery, rewrittenQuery := req.Query, req.Query
	rewriteUsed := false

	if req.Plan.EnableQueryRewrite && s.shouldRewriteQuery(req.Query) {
		rewritten, err := s.queryRewrite(ctx, req.Query)
		if err == nil && rewritten != "" {
			rewrittenQuery = rewritten
			rewriteUsed = true
			s.logger.Debug("Query rewritten", "original", originalQuery, "rewritten", rewrittenQuery)
		}
		// Query rewrite failure is not fatal, continue with original query
	}

	// 2. Try vector search
	var vectorResults []*SearchResult
	var vectorErr error

	s.logger.Info("Checking embedding client", "client_nil", s.embeddingClient == nil, "enabled", s.embeddingClient != nil && s.embeddingClient.IsEnabled())

	if s.embeddingClient != nil && s.embeddingClient.IsEnabled() {
		// Check embedding circuit breaker
		s.logger.Info("Checking embedding circuit breaker")
		if err := s.retrievalGuard.CheckEmbeddingCircuitBreaker(); err == nil {
			s.logger.Info("Getting embeddings for queries")
			originalEmbedding := s.getEmbedding(ctx, originalQuery)
			rewrittenEmbedding := s.getEmbedding(ctx, rewrittenQuery)

			s.logger.Info("Embeddings obtained", "original_len", len(originalEmbedding), "rewritten_len", len(rewrittenEmbedding))

			// Parallel vector search with timeout protection
			vectorResults, vectorErr = s.parallelVectorSearch(ctx, originalEmbedding, rewrittenEmbedding, req)

			if vectorErr == nil {
				s.retrievalGuard.RecordEmbeddingSuccess()
				s.logger.Info("Vector search succeeded", "results_count", len(vectorResults))
			} else {
				s.retrievalGuard.RecordEmbeddingFailure()
				s.logger.Error("Vector search failed", "error", vectorErr)
			}
		} else {
			s.logger.Warn("Embedding circuit breaker open, using keyword search only", "error", err)
			vectorErr = err
		}
	} else {
		s.logger.Warn("Embedding client not available, skipping vector search")
	}

	// 3. BM25 fallback (if vector search failed or configured)
	var keywordResults []*SearchResult
	if vectorErr != nil || req.Plan.EnableKeywordSearch {
		keywordResults = s.bm25Search(ctx, req)
	}

	// 4. Merge and rank results
	var finalResults []*SearchResult
	if len(vectorResults) > 0 && len(keywordResults) > 0 {
		// Hybrid search: merge vector and keyword results
		finalResults = s.mergeAndRank(ctx, vectorResults, keywordResults, req.Plan)
	} else if len(vectorResults) > 0 {
		// Vector search only
		finalResults = vectorResults
	} else {
		// Keyword search only
		finalResults = keywordResults
	}

	// 5. Apply TopK limit
	if len(finalResults) > req.TopK {
		finalResults = finalResults[:req.TopK]
	}

	// 6. Apply minimum score filter
	// Debug: log results before filtering
	s.logger.Info("Before score filter", "results_count", len(finalResults), "min_score", req.MinScore)
	for i, result := range finalResults {
		s.logger.Info("Result before filter", "index", i, "score", result.Score, "content", truncateForLog(result.Content, 50))
	}

	finalResults = s.filterByScore(finalResults, req.MinScore)

	// Debug: log results after filtering
	s.logger.Info("After score filter", "results_count", len(finalResults))

	// 7. Generate retrieval trace (if enabled)
	if req.EnableTrace {
		req.Trace = &RetrievalTrace{
			OriginalQuery:   originalQuery,
			RewrittenQuery:  rewrittenQuery,
			RewriteUsed:     rewriteUsed,
			VectorResults:   len(vectorResults),
			KeywordResults:  len(keywordResults),
			FinalResults:    len(finalResults),
			ExecutionTime:   time.Since(startTime),
			VectorError:     vectorErr,
			SearchBreakdown: s.countResultsBySource(finalResults),
		}
	}

	return finalResults, nil
}

// validateRequest validates the search request.
func (s *RetrievalService) validateRequest(req *SearchRequest) error {
	if req == nil {
		return errors.ErrInvalidArgument
	}
	if req.Query == "" {
		return errors.ErrInvalidArgument
	}
	if req.TenantID == "" {
		return errors.ErrInvalidArgument
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	return nil
}

// getEmbedding retrieves embedding for a query with caching.
func (s *RetrievalService) getEmbedding(ctx context.Context, query string) []float64 {
	if query == "" {
		return nil
	}

	embedding, err := s.embeddingClient.Embed(ctx, query)
	if err != nil {
		s.logger.Warn("Failed to get embedding", "query", query, "error", err)
		return nil
	}

	return embedding
}

// shouldRewriteQuery determines if a query should be rewritten.
func (s *RetrievalService) shouldRewriteQuery(query string) bool {
	// Skip short queries
	if len(query) < 10 {
		return false
	}

	// Skip if query is in cache (simple check)
	if s.isQueryInCache(query) {
		return false
	}

	// Complex query patterns that benefit from rewriting
	complexPatterns := []string{
		"如何", "怎么", "什么", "why", "为什么",
		"what", "how", "explain", "解释", "describe", "描述",
	}

	queryLower := toLower(query)
	for _, pattern := range complexPatterns {
		if contains(queryLower, toLower(pattern)) {
			return true
		}
	}

	return false
}

// isQueryInCache checks if query results are already cached.
// This implements query cache check as specified in design standard.
func (s *RetrievalService) isQueryInCache(query string) bool {
	// Simple implementation - check if query was recently processed
	// In production, this would check Redis cache or LRU cache
	// For now, return false to enable all query rewrites
	return false
}

// queryRewrite rewrites a query for better retrieval.
// This uses LLM to expand and refine the query.
func (s *RetrievalService) queryRewrite(ctx context.Context, query string) (string, error) {
	// TODO: implement LLM-based query rewriting
	// For now, return original query
	return query, nil
}

// parallelVectorSearch performs parallel vector search across multiple sources.
// Uses errgroup for concurrency control and error handling as per design standard.
func (s *RetrievalService) parallelVectorSearch(ctx context.Context, originalEmb, rewrittenEmb []float64, req *SearchRequest) ([]*SearchResult, error) {
	var mu sync.Mutex
	var allResults []*SearchResult

	// Set 2 second timeout for vector search
	searchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Use database timeout from retrieval guard
	searchCtx, _ = s.retrievalGuard.WithDBTimeout(searchCtx)

	// Create errgroup for parallel search (per design standard)
	eg, ctx := errgroup.WithContext(searchCtx)
	eg.SetLimit(3) // Limit concurrent goroutines

	// Search knowledge base with rewritten query (parallel)
	if req.Plan.SearchKnowledge && len(rewrittenEmb) > 0 {
		eg.Go(func() error {
			results := s.searchKnowledgeVector(ctx, rewrittenEmb, req)
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
			return nil // Don't fail other searches if one fails
		})
	}

	// Search experiences with original query (parallel)
	if req.Plan.SearchExperience && len(originalEmb) > 0 {
		eg.Go(func() error {
			results := s.searchExperienceVector(ctx, originalEmb, req)
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
			return nil
		})
	}

	// Search tools with rewritten query (parallel)
	if req.Plan.SearchTools && len(rewrittenEmb) > 0 {
		eg.Go(func() error {
			results := s.searchToolsVector(ctx, rewrittenEmb, req)
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
			return nil
		})
	}

	// Wait for all parallel searches to complete
	if err := eg.Wait(); err != nil {
		s.logger.Warn("Some parallel searches failed", "error", err)
	}

	return allResults, nil
}

// searchKnowledgeVector performs vector search on knowledge base using pgvector.
// This uses cosine similarity to find the most relevant knowledge chunks.
func (s *RetrievalService) searchKnowledgeVector(ctx context.Context, embedding []float64, req *SearchRequest) []*SearchResult {
	if len(embedding) == 0 {
		return []*SearchResult{}
	}

	// Use Repository layer to search knowledge base
	chunks, err := s.kbRepo.SearchByVector(ctx, embedding, req.TenantID, req.Plan.TopK)
	if err != nil {
		s.logger.Error("Knowledge vector search failed", "error", err)
		return []*SearchResult{}
	}

	// Convert KnowledgeChunk to SearchResult
	results := make([]*SearchResult, 0, len(chunks))
	for _, chunk := range chunks {
		result := &SearchResult{
			ID:        chunk.ID,
			Content:   chunk.Content,
			Source:    chunk.SourceType,
			Type:      "knowledge",
			Metadata:  chunk.Metadata,
			CreatedAt: chunk.CreatedAt,
		}

		// Extract similarity score from metadata if available
		if similarity, ok := chunk.Metadata["similarity"].(float64); ok {
			result.Score = similarity
		}

		results = append(results, result)
	}

	return results
}

// searchExperienceVector performs vector search on experiences using pgvector.
// This uses cosine similarity to find the most relevant agent experiences.
func (s *RetrievalService) searchExperienceVector(ctx context.Context, embedding []float64, req *SearchRequest) []*SearchResult {
	// TODO: Implement using ExperienceRepository when available
	// For now, return empty results
	return []*SearchResult{}
}

// searchToolsVector performs vector search on tools using pgvector.
// This combines semantic search with usage statistics for tool ranking.
func (s *RetrievalService) searchToolsVector(ctx context.Context, embedding []float64, req *SearchRequest) []*SearchResult {
	// TODO: Implement using ToolRepository when available
	// For now, return empty results
	return []*SearchResult{}
}

// bm25Search performs BM25 full-text search using PostgreSQL tsvector.
// This serves as a fallback when vector search fails or is disabled.
func (s *RetrievalService) bm25Search(ctx context.Context, req *SearchRequest) []*SearchResult {
	if req.Query == "" {
		return []*SearchResult{}
	}

	var results []*SearchResult

	// Search knowledge base using BM25
	knowledgeResults := s.bm25SearchKnowledge(ctx, req.Query, req.TenantID, req.Plan.TopK)
	results = append(results, knowledgeResults...)

	// Search experiences using BM25
	experienceResults := s.bm25SearchExperience(ctx, req.Query, req.TenantID, req.Plan.TopK)
	results = append(results, experienceResults...)

	// Search tools using BM25
	toolResults := s.bm25SearchTools(ctx, req.Query, req.TenantID, req.Plan.TopK)
	results = append(results, toolResults...)

	return results
}

// bm25SearchKnowledge performs BM25 search on knowledge base.
func (s *RetrievalService) bm25SearchKnowledge(ctx context.Context, query string, tenantID string, limit int) []*SearchResult {
	// Use Repository layer for keyword search
	chunks, err := s.kbRepo.SearchByKeyword(ctx, query, tenantID, limit)
	if err != nil {
		s.logger.Error("Knowledge BM25 search failed", "error", err)
		return []*SearchResult{}
	}

	// Convert KnowledgeChunk to SearchResult
	results := make([]*SearchResult, 0, len(chunks))
	for _, chunk := range chunks {
		result := &SearchResult{
			ID:        chunk.ID,
			Content:   chunk.Content,
			Source:    chunk.SourceType,
			Type:      "knowledge",
			Metadata:  chunk.Metadata,
			CreatedAt: chunk.CreatedAt,
		}

		// Extract keyword score from metadata if available
		if score, ok := chunk.Metadata["keyword_score"].(float64); ok {
			result.Score = score
		}

		results = append(results, result)
	}

	return results
}

// bm25SearchExperience performs BM25 search on experiences.
func (s *RetrievalService) bm25SearchExperience(ctx context.Context, query string, tenantID string, limit int) []*SearchResult {
	// TODO: Implement using ExperienceRepository when available
	// For now, return empty results
	return []*SearchResult{}
}

// bm25SearchTools performs BM25 search on tools.
func (s *RetrievalService) bm25SearchTools(ctx context.Context, query string, tenantID string, limit int) []*SearchResult {
	// TODO: Implement using ToolRepository when available
	// For now, return empty results
	return []*SearchResult{}
}

// mergeAndRank merges and ranks results from multiple sources using RRF and time decay.
func (s *RetrievalService) mergeAndRank(ctx context.Context, vectorResults, keywordResults []*SearchResult, plan *RetrievalPlan) []*SearchResult {
	// RRF (Reciprocal Rank Fusion) algorithm with time decay
	type scoreEntry struct {
		id     string
		score  float64
		result *SearchResult
	}

	scores := make(map[string]*scoreEntry)

	// Process vector search results with time decay
	for i, result := range vectorResults {
		// Use original score normalized by position
		rrScore := result.Score / float64(i+1) // Combine original score with position
		timeDecay := 1.0

		if plan.EnableTimeDecay {
			timeDecay = s.calculateTimeDecay(result.CreatedAt)
		}

		// Apply source-specific weight
		var weight float64
		switch result.Source {
		case "knowledge":
			weight = plan.KnowledgeWeight
		case "experience":
			weight = plan.ExperienceWeight
		case "tool":
			weight = plan.ToolsWeight
		case "task_result":
			weight = plan.TaskResultsWeight
		default:
			weight = 1.0
		}

		finalScore := rrScore * weight * timeDecay

		if entry, exists := scores[result.ID]; exists {
			entry.score += finalScore
		} else {
			scores[result.ID] = &scoreEntry{
				id:     result.ID,
				score:  finalScore,
				result: result,
			}
		}
	}

	// Process keyword search results with time decay
	for i, result := range keywordResults {
		// Use original score normalized by position
		rrScore := result.Score / float64(i+1) // Combine original score with position
		timeDecay := 1.0

		if plan.EnableTimeDecay {
			timeDecay = s.calculateTimeDecay(result.CreatedAt)
		}

		// Keyword results have lower weight (0.3)
		finalScore := rrScore * 0.3 * timeDecay

		if entry, exists := scores[result.ID]; exists {
			entry.score += finalScore
		} else {
			scores[result.ID] = &scoreEntry{
				id:     result.ID,
				score:  finalScore,
				result: result,
			}
		}
	}

	// Sort by score (descending)
	var sortedEntries []*scoreEntry
	for _, entry := range scores {
		sortedEntries = append(sortedEntries, entry)
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].score > sortedEntries[j].score
	})

	// Return sorted results
	finalResults := make([]*SearchResult, len(sortedEntries))
	for i, entry := range sortedEntries {
		finalResults[i] = entry.result
		finalResults[i].Score = entry.score // Update with merged score
	}

	return finalResults
}

// calculateTimeDecay calculates time-based decay factor for scoring.
// Newer content gets higher scores to prevent old data from dominating.
func (s *RetrievalService) calculateTimeDecay(createdAt time.Time) float64 {
	ageHours := time.Since(createdAt).Hours()
	lambda := 0.01 // Decay coefficient (configurable)

	// Exponential decay: older content has lower weight
	decay := math.Exp(-lambda * ageHours)

	// Ensure minimum decay factor to avoid completely ignoring old data
	if decay < 0.1 {
		decay = 0.1
	}

	return decay
}

// filterByScore filters results by minimum score threshold.
func (s *RetrievalService) filterByScore(results []*SearchResult, minScore float64) []*SearchResult {
	// Filter by minimum score (negative minScore means no filtering)
	filtered := make([]*SearchResult, 0, len(results))
	for _, result := range results {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// countResultsBySource counts results by source for trace information.
func (s *RetrievalService) countResultsBySource(results []*SearchResult) map[string]int {
	counts := make(map[string]int)
	for _, result := range results {
		counts[result.Source]++
	}
	return counts
}

// Helper functions for string manipulation

func parseJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// truncateForLog truncates string for logging
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func toLower(s string) string {
	// Simple lowercase conversion
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(s) < len(substr) {
		return -1
	}

	s = toLower(s)
	substr = toLower(substr)

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
