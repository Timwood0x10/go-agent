// Package services provides additional unit tests for retrieval services.
package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCalculateTimeDecay_WithCustomTime tests time decay with custom time values.
func TestCalculateTimeDecay_WithCustomTime(t *testing.T) {
	service := &RetrievalService{}

	// 使用当前时间作为基准
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
			// 计算相对于当前时间的时间点
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
			name:        "all above threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.9},
				{ID: "2", Score: 0.8},
				{ID: "3", Score: 0.7},
			},
			minScore:    0.5,
			expectCount: 3,
		},
		{
			name:        "some below threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.9},
				{ID: "2", Score: 0.4},
				{ID: "3", Score: 0.3},
			},
			minScore:    0.5,
			expectCount: 1,
		},
		{
			name:        "none above threshold",
			results: []*SearchResult{
				{ID: "1", Score: 0.1},
				{ID: "2", Score: 0.2},
				{ID: "3", Score: 0.3},
			},
			minScore:    0.5,
			expectCount: 0,
		},
		{
			name:        "negative scores",
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
			name:        "exact threshold match",
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
	vectorResults := []*SearchResult{
		{ID: "1", Score: 0.8, Source: "knowledge", CreatedAt: now},
		{ID: "2", Score: 0.8, Source: "knowledge", CreatedAt: now},
	}

	keywordResults := []*SearchResult{
		{ID: "2", Score: 0.8, Source: "knowledge", CreatedAt: now},
		{ID: "3", Score: 0.8, Source: "knowledge", CreatedAt: now},
	}

	merged := service.mergeAndRank(context.Background(), vectorResults, keywordResults, plan)

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
	vectorResults := []*SearchResult{
		{ID: "1", Score: 0.9, Source: "knowledge", CreatedAt: now},
		{ID: "2", Score: 0.7, Source: "experience", CreatedAt: now},
	}

	keywordResults := []*SearchResult{
		{ID: "3", Score: 0.8, Source: "tool", CreatedAt: now},
		{ID: "4", Score: 0.6, Source: "knowledge", CreatedAt: now},
	}

	merged := service.mergeAndRank(context.Background(), vectorResults, keywordResults, plan)

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
				"tool":      1,
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