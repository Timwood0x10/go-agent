// Package services provides additional unit tests for retrieval services.
package services

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCalculateTimeDecay_WithCustomTime tests time decay with custom time values.
func TestCalculateTimeDecay_WithCustomTime(t *testing.T) {
	service := &RetrievalService{}

	// Use current time as baseline
	now := time.Now()

	tests := []struct {
		name          string
		age           time.Duration
		expectedRange [2]float64 // [min, max]
	}{
		{
			name:          "very recent (1 hour)",
			age:           1 * time.Hour,
			expectedRange: [2]float64{0.98, 1.0},
		},
		{
			name:          "recent (1 day)",
			age:           24 * time.Hour,
			expectedRange: [2]float64{0.78, 0.8},
		},
		{
			name:          "medium (1 week)",
			age:           7 * 24 * time.Hour,
			expectedRange: [2]float64{0.18, 0.2},
		},
		{
			name:          "old (1 month)",
			age:           30 * 24 * time.Hour,
			expectedRange: [2]float64{0.1, 0.1},
		},
		{
			name:          "very old (3 months)",
			age:           90 * 24 * time.Hour,
			expectedRange: [2]float64{0.1, 0.1},
		},
		{
			name:          "ancient (1 year)",
			age:           365 * 24 * time.Hour,
			expectedRange: [2]float64{0.1, 0.1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate time point relative to current time
			testTime := now.Add(-tt.age)
			decay := service.calculateTimeDecay(testTime)

			assert.GreaterOrEqual(t, decay, tt.expectedRange[0],
				"Decay should be >= %.2f for age %v, got %.2f", tt.expectedRange[0], tt.age, decay)
			assert.LessOrEqual(t, decay, tt.expectedRange[1],
				"Decay should be <= %.2f for age %v, got %.2f", tt.expectedRange[1], tt.age, decay)
		})
	}
}

// TestFilterByScore_WithEdgeCases tests score filtering with edge cases.
func TestFilterByScore_WithEdgeCases(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name        string
		results     []*SearchResult
		minScore    float64
		expectCount int
	}{
		{
			name: "all above threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.9},
				{ID: "2", Score: 0.8},
				{ID: "3", Score: 0.7},
			},
			minScore:    0.5,
			expectCount: 3,
		},
		{
			name: "some below threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.9},
				{ID: "2", Score: 0.4},
				{ID: "3", Score: 0.3},
			},
			minScore:    0.5,
			expectCount: 1,
		},
		{
			name: "none above threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.1},
				{ID: "2", Score: 0.2},
				{ID: "3", Score: 0.3},
			},
			minScore:    0.5,
			expectCount: 0,
		},
		{
			name: "negative scores",
			results: []*SearchResult{
				{ID: "1", Score: -0.1},
				{ID: "2", Score: 0.5},
				{ID: "3", Score: -0.2},
			},
			minScore:    0.0,
			expectCount: 1,
		},
		{
			name:        "empty results",
			results:     []*SearchResult{},
			minScore:    0.5,
			expectCount: 0,
		},
		{
			name: "exact threshold match",
			results: []*SearchResult{
				{ID: "1", Score: 0.5},
				{ID: "2", Score: 0.5},
				{ID: "3", Score: 0.6},
			},
			minScore:    0.5,
			expectCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := service.filterByScore(tt.results, tt.minScore)
			assert.Equal(t, tt.expectCount, len(filtered))
		})
	}
}

// TestMergeAndRank_WithEqualScores tests merge with equal scores.
func TestMergeAndRank_WithEqualScores(t *testing.T) {
	service := &RetrievalService{}
	plan := DefaultRetrievalPlan()

	now := time.Now()

	// Create results with equal scores and same CreatedAt to avoid time decay
	allResults := []*SearchResult{
		{ID: "1", Score: 0.8, Source: "knowledge", SubSource: "vector", QueryWeight: 1.0, CreatedAt: now},
		{ID: "2", Score: 0.8, Source: "knowledge", SubSource: "vector", QueryWeight: 1.0, CreatedAt: now},
		{ID: "2", Score: 0.8, Source: "knowledge", SubSource: "keyword", QueryWeight: 1.0, CreatedAt: now},
		{ID: "3", Score: 0.8, Source: "knowledge", SubSource: "keyword", QueryWeight: 1.0, CreatedAt: now},
	}

	merged := service.mergeAndRerank(allResults, plan)

	// Should have 3 unique results (ID 2 appears in both vector and keyword results)
	assert.Equal(t, 3, len(merged))

	// Verify results are properly sorted by score (descending)
	for i := 1; i < len(merged); i++ {
		assert.GreaterOrEqual(t, merged[i-1].Score, merged[i].Score,
			"Results should be sorted by score in descending order")
	}

	// Verify all scores are positive
	for _, result := range merged {
		assert.Greater(t, result.Score, 0.0, "All scores should be positive")
	}
}

// TestMergeAndRank_WithDifferentSources tests merge with different sources.
func TestMergeAndRank_WithDifferentSources(t *testing.T) {
	service := &RetrievalService{}
	plan := DefaultRetrievalPlan()

	now := time.Now()

	// Create results from different sources
	allResults := []*SearchResult{
		{ID: "1", Score: 0.9, Source: "knowledge", SubSource: "vector", QueryWeight: 1.0, CreatedAt: now},
		{ID: "2", Score: 0.7, Source: "experience", SubSource: "vector", QueryWeight: 1.0, CreatedAt: now},
		{ID: "3", Score: 0.8, Source: "tool", SubSource: "keyword", QueryWeight: 1.0, CreatedAt: now},
		{ID: "4", Score: 0.6, Source: "knowledge", SubSource: "keyword", QueryWeight: 1.0, CreatedAt: now},
	}

	merged := service.mergeAndRerank(allResults, plan)

	// Should have 4 unique results
	assert.Equal(t, 4, len(merged))

	// Verify all sources are represented
	sources := make(map[string]bool)
	for _, result := range merged {
		sources[result.Source] = true
	}

	assert.True(t, sources["knowledge"])
	assert.True(t, sources["experience"])
	assert.True(t, sources["tool"])
}

// TestCountResultsBySource_WithEmptyResults tests counting with empty results.
func TestCountResultsBySource_WithEmptyResults(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name     string
		results  []*SearchResult
		expected map[string]int
	}{
		{
			name:     "empty results",
			results:  []*SearchResult{},
			expected: map[string]int{},
		},
		{
			name: "single source",
			results: []*SearchResult{
				{ID: "1", Source: "knowledge"},
				{ID: "2", Source: "knowledge"},
			},
			expected: map[string]int{"knowledge": 2},
		},
		{
			name: "multiple sources",
			results: []*SearchResult{
				{ID: "1", Source: "knowledge"},
				{ID: "2", Source: "experience"},
				{ID: "3", Source: "tool"},
				{ID: "4", Source: "knowledge"},
			},
			expected: map[string]int{
				"knowledge":  2,
				"experience": 1,
				"tool":       1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts := service.countResultsBySource(tt.results)
			assert.Equal(t, tt.expected, counts)
		})
	}
}

// TestShouldRewriteQuery_WithEdgeCases tests query rewrite decision with edge cases.
func TestShouldRewriteQuery_WithEdgeCases(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "single character",
			query:    "a",
			expected: false,
		},
		{
			name:     "very long query (200 chars)",
			query:    strings.Repeat("test ", 50),
			expected: false,
		},
		{
			name:     "query with special characters",
			query:    "what's the best approach? @#$%^",
			expected: true,
		},
		{
			name:     "query with numbers only",
			query:    "1234567890",
			expected: false,
		},
		{
			name:     "question mark only",
			query:    "?",
			expected: false,
		},
		{
			name:     "mixed case question",
			query:    "WhAt Is MaChInE LeArNiNg",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.shouldRewriteQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultRetrievalPlan_Immutable tests that default plan is properly configured.
func TestDefaultRetrievalPlan_Immutable(t *testing.T) {
	plan1 := DefaultRetrievalPlan()
	plan2 := DefaultRetrievalPlan()

	// Multiple calls should produce consistent results
	assert.Equal(t, plan1.SearchKnowledge, plan2.SearchKnowledge)
	assert.Equal(t, plan1.SearchExperience, plan2.SearchExperience)
	assert.Equal(t, plan1.KnowledgeWeight, plan2.KnowledgeWeight)
	assert.Equal(t, plan1.TopK, plan2.TopK)
}

// TestRetrievalPlan_WeightsSum tests that weights sum to 1.0.
func TestRetrievalPlan_WeightsSum(t *testing.T) {
	plan := DefaultRetrievalPlan()

	totalWeight := plan.KnowledgeWeight + plan.ExperienceWeight + plan.ToolsWeight + plan.TaskResultsWeight

	assert.InDelta(t, 1.0, totalWeight, 0.001, "Weights should sum to 1.0")
}

// TestSearchRequest_NilHandling tests handling of nil search request.
func TestSearchRequest_NilHandling(t *testing.T) {
	// This test requires full postgres.Pool setup
	t.Skip("Requires full postgres.Pool setup - to be implemented")
}

// TestCalculateTimeDecay_NegativeAge tests that negative age doesn't break the function.
func TestCalculateTimeDecay_NegativeAge(t *testing.T) {
	service := &RetrievalService{}

	now := time.Now()

	// Future time (negative age) should be handled gracefully
	futureTime := now.Add(1 * time.Hour)
	decay := service.calculateTimeDecay(futureTime)

	// Future content should have maximum decay factor
	assert.GreaterOrEqual(t, decay, 1.0, "Future content should have maximum decay factor")
}

// TestCalculateTimeDecay_ZeroAge tests that zero age works correctly.
func TestCalculateTimeDecay_ZeroAge(t *testing.T) {
	service := &RetrievalService{}

	now := time.Now()

	// Current time (zero age) should have maximum decay factor
	decay := service.calculateTimeDecay(now)

	// Use >= 0.99 to tolerate floating-point precision issues
	assert.GreaterOrEqual(t, decay, 0.99, "Current content should have maximum decay factor")
	assert.LessOrEqual(t, decay, 1.0, "Decay should not exceed 1.0")
}

// TestValidateRequest tests request validation.
func TestValidateRequest(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name    string
		req     *SearchRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-1",
				TopK:     10,
				Plan:     DefaultRetrievalPlan(),
			},
			wantErr: false,
		},
		{
			name: "empty query",
			req: &SearchRequest{
				Query:    "",
				TenantID: "tenant-1",
				TopK:     10,
				Plan:     DefaultRetrievalPlan(),
			},
			wantErr: true,
		},
		{
			name: "empty tenant ID",
			req: &SearchRequest{
				Query:    "test query",
				TenantID: "",
				TopK:     10,
				Plan:     DefaultRetrievalPlan(),
			},
			wantErr: true,
		},
		{
			name: "zero TopK - should be auto-corrected",
			req: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-1",
				TopK:     0,
				Plan:     DefaultRetrievalPlan(),
			},
			wantErr: false, // Should auto-correct to 10
		},
		{
			name: "negative TopK - should be auto-corrected",
			req: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-1",
				TopK:     -5,
				Plan:     DefaultRetrievalPlan(),
			},
			wantErr: false, // Should auto-correct to 10
		},
		{
			name: "nil plan",
			req: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-1",
				TopK:     10,
				Plan:     nil,
			},
			wantErr: false, // Should use default plan
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsQueryInCache tests query cache checking.
func TestIsQueryInCache(t *testing.T) {
	service := &RetrievalService{}

	// Current implementation always returns false (TODO)
	assert.False(t, service.isQueryInCache("any query"))
	assert.False(t, service.isQueryInCache(""))
	assert.False(t, service.isQueryInCache("how to use go"))
}

// TestShouldRewriteQuery_Additional tests query rewrite decision logic with additional cases.
func TestShouldRewriteQuery_Additional(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{
			name:  "Chinese question - 如何",
			query: "如何使用 Go 进行并发编程",
			want:  true,
		},
		{
			name:  "Chinese question - 怎么",
			query: "怎么配置 PostgreSQL 连接",
			want:  true,
		},
		{
			name:  "Chinese question - 什么",
			query: "什么是向量数据库",
			want:  true,
		},
		{
			name:  "English question - why",
			query: "why should I use Rust",
			want:  true,
		},
		{
			name:  "English question - why lowercase",
			query: "why use microservices",
			want:  true,
		},
		{
			name:  "English question - what",
			query: "what is machine learning",
			want:  true,
		},
		{
			name:  "English question - how",
			query: "how to implement caching",
			want:  true,
		},
		{
			name:  "English question - explain",
			query: "explain the difference between HTTP and HTTPS",
			want:  true,
		},
		{
			name:  "English question - describe",
			query: "describe the architecture of Kubernetes",
			want:  true,
		},
		{
			name:  "Chinese question - 解释",
			query: "解释一下 Docker 的基本概念",
			want:  true,
		},
		{
			name:  "Chinese question - 描述",
			query: "描述一下微服务的优缺点",
			want:  true,
		},
		{
			name:  "statement without question word",
			query: "I want to learn Go programming",
			want:  false,
		},
		{
			name:  "code snippet",
			query: "func main() { println(\"hello\") }",
			want:  false,
		},
		{
			name:  "URL",
			query: "https://example.com/documentation",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.shouldRewriteQuery(tt.query)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestQueryRewrite tests query rewriting functionality.
func TestQueryRewrite(t *testing.T) {
	service := &RetrievalService{
		logger:         slog.Default(),
		synonymRules:   loadSynonymRules(),
		embeddingCache: make(map[string][]float64),
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "normal query",
			query:   "how to use go for web development",
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: false, // Returns original query, no error
		},
		{
			name:    "query with special characters",
			query:   "test @#$ query",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.queryRewrite(ctx, tt.query)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSearchKnowledgeVector tests knowledge base vector search.
func TestSearchKnowledgeVector(t *testing.T) {
	// Skip if kbRepo is not initialized
	t.Skip("Requires full database setup - kbRepo dependency")

	service := &RetrievalService{}

	ctx := context.Background()

	// Create test embedding
	embedding := make([]float64, 1024)
	for i := range embedding {
		embedding[i] = float64(i) / 1024.0
	}

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
		Plan:     DefaultRetrievalPlan(),
	}

	// This test requires full database setup
	results := service.searchKnowledgeVector(ctx, embedding, req)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestSearchExperienceVector tests experience repository vector search.
func TestSearchExperienceVector(t *testing.T) {
	// Skip if not implemented
	t.Skip("ExperienceRepository not yet implemented")

	service := &RetrievalService{}

	ctx := context.Background()

	// Create test embedding
	embedding := make([]float64, 1024)
	for i := range embedding {
		embedding[i] = float64(i) / 1024.0
	}

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
		Plan:     DefaultRetrievalPlan(),
	}

	// This test requires full database setup
	results := service.searchExperienceVector(ctx, embedding, req)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestSearchToolsVector tests tools vector search.
func TestSearchToolsVector(t *testing.T) {
	// Skip if not implemented
	t.Skip("ToolRepository not yet implemented")

	service := &RetrievalService{}

	ctx := context.Background()

	// Create test embedding
	embedding := make([]float64, 1024)
	for i := range embedding {
		embedding[i] = float64(i) / 1024.0
	}

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
		Plan:     DefaultRetrievalPlan(),
	}

	// This test requires full database setup
	results := service.searchToolsVector(ctx, embedding, req)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestBm25Search tests BM25 keyword search.
func TestBm25Search(t *testing.T) {
	// Skip if kbRepo is not initialized
	t.Skip("Requires full database setup - kbRepo dependency")

	service := &RetrievalService{}

	ctx := context.Background()

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
		Plan:     DefaultRetrievalPlan(),
	}

	// This test requires full database setup
	results := service.bm25Search(ctx, req)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestBm25SearchKnowledge tests BM25 search in knowledge base.
func TestBm25SearchKnowledge(t *testing.T) {
	// Skip if kbRepo is not initialized
	t.Skip("Requires full database setup - kbRepo dependency")

	service := &RetrievalService{}

	ctx := context.Background()

	// This test requires full database setup
	results := service.bm25SearchKnowledge(ctx, "test query", "tenant-1", 10)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestBm25SearchExperience tests BM25 search in experience repository.
func TestBm25SearchExperience(t *testing.T) {
	// Skip if not implemented
	t.Skip("ExperienceRepository not yet implemented")

	service := &RetrievalService{}

	ctx := context.Background()

	// This test requires full database setup
	results := service.bm25SearchExperience(ctx, "test query", "tenant-1", 10)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}

// TestBm25SearchTools tests BM25 search in tools.
func TestBm25SearchTools(t *testing.T) {
	// Skip if not implemented
	t.Skip("ToolRepository not yet implemented")

	service := &RetrievalService{}

	ctx := context.Background()

	// This test requires full database setup
	results := service.bm25SearchTools(ctx, "test query", "tenant-1", 10)

	// Results should be a slice (may be empty if no database)
	assert.NotNil(t, results)
	assert.IsType(t, []*SearchResult{}, results)
}
