// Package services provides simplified retrieval services for the storage system.
// This service focuses on pure vector similarity search without complex weight calculations.
package services

import (
	"context"
	"fmt"
	"log/slog"

	"goagent/internal/storage/postgres/embedding"
	"goagent/internal/storage/postgres/repositories"
)

// SimpleRetrievalConfig configuration for simple retrieval service
type SimpleRetrievalConfig struct {
	TopK        int     `json:"top_k"`        // Number of results to return
	MinScore    float64 `json:"min_score"`    // Minimum similarity score
	QueryPrefix string  `json:"query_prefix"` // Prefix for query embedding (e.g., "query:")
}

// SimpleSearchResult simple search result with only essential fields
type SimpleSearchResult struct {
	Content string  `json:"content"` // Content text
	Source  string  `json:"source"`  // Source document/file path
	Score   float64 `json:"score"`   // Cosine similarity score (1 - distance)
}

// SimpleRetrievalService provides pure vector similarity search
// This is inspired by ChromaDB's simple and direct approach:
// - Direct vector similarity search (1 - cosine_distance)
// - No complex weight calculations
// - No time decay
// - No query rewrites
// - Simple and effective for single knowledge base scenarios
type SimpleRetrievalService struct {
	repo      *repositories.KnowledgeRepository
	embedding *embedding.EmbeddingClient
	config    *SimpleRetrievalConfig
}

// NewSimpleRetrievalService creates a new simple retrieval service
func NewSimpleRetrievalService(
	repo *repositories.KnowledgeRepository,
	embeddingClient *embedding.EmbeddingClient,
	config *SimpleRetrievalConfig,
) *SimpleRetrievalService {
	if config == nil {
		config = &SimpleRetrievalConfig{
			TopK:        5,
			MinScore:    0.6,
			QueryPrefix: "query:",
		}
	}

	return &SimpleRetrievalService{
		repo:      repo,
		embedding: embeddingClient,
		config:    config,
	}
}

// Search performs simple vector similarity search
// Returns results with cosine similarity score (1 - distance), where:
// - 1.0 = perfect match
// - 0.0 = orthogonal (no relation)
// - -1.0 = opposite meaning
func (s *SimpleRetrievalService) Search(ctx context.Context, tenantID, query string) ([]*SimpleSearchResult, error) {
	slog.Info("SimpleRetrievalService.Search",
		"tenant_id", tenantID,
		"query", query,
		"top_k", s.config.TopK,
		"min_score", s.config.MinScore)

	// Generate embedding for the query with optional prefix
	fullQuery := query
	if s.config.QueryPrefix != "" {
		fullQuery = s.config.QueryPrefix + query
	}

	queryEmbedding, err := s.embedding.EmbedWithPrefix(ctx, fullQuery, s.config.QueryPrefix)
	if err != nil {
		slog.Error("Failed to embed query", "error", err)
		return nil, fmt.Errorf("embed query: %w", err)
	}

	slog.Debug("Query embedding generated", "dimension", len(queryEmbedding))

	// Perform pure vector similarity search
	// SearchByVector returns chunks with similarity in metadata
	// Use larger limit to get more candidates before filtering
	chunks, err := s.repo.SearchByVector(ctx, queryEmbedding, tenantID, s.config.TopK*5)
	if err != nil {
		slog.Error("Vector search failed", "error", err)
		return nil, fmt.Errorf("vector search: %w", err)
	}

	slog.Debug("Raw chunks retrieved", "count", len(chunks))

	// Convert to simple results and filter by min_score
	var results []*SimpleSearchResult
	for _, chunk := range chunks {
		// Extract similarity from metadata (set by SearchByVector)
		// SearchByVector computes: 1 - cosine_distance
		similarity, ok := chunk.Metadata["similarity"].(float64)
		if !ok {
			slog.Warn("No similarity score found for chunk", "id", chunk.ID)
			continue
		}

		// Log all results before filtering
		slog.Info("Chunk found",
			"source", chunk.Source,
			"similarity", similarity,
			"min_score", s.config.MinScore,
			"content_preview", truncateString(chunk.Content, 100))

		// Filter by min_score threshold
		if similarity < s.config.MinScore {
			slog.Debug("Skipping low similarity result",
				"similarity", similarity,
				"threshold", s.config.MinScore)
			continue
		}

		results = append(results, &SimpleSearchResult{
			Content: chunk.Content,
			Source:  chunk.Source,
			Score:   similarity,
		})
	}

	// Limit to TopK results
	if len(results) > s.config.TopK {
		results = results[:s.config.TopK]
	}

	slog.Info("Search completed",
		"results_count", len(results),
		"min_score", s.config.MinScore)

	return results, nil
}

// SetConfig updates the retrieval configuration
func (s *SimpleRetrievalService) SetConfig(config *SimpleRetrievalConfig) {
	s.config = config
}

// GetConfig returns the current configuration
func (s *SimpleRetrievalService) GetConfig() *SimpleRetrievalConfig {
	return s.config
}

// truncateString truncate string for log output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
