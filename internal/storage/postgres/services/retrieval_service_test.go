// Package services provides retrieval services for the storage system.
package services

import (
	"context"
	"testing"
	"time"

	"goagent/internal/core/errors"
)

// TestDefaultRetrievalPlan tests the default retrieval plan configuration.
func TestDefaultRetrievalPlan(t *testing.T) {
	plan := DefaultRetrievalPlan()

	if !plan.SearchKnowledge {
		t.Error("knowledge search should be enabled by default")
	}
	if !plan.SearchExperience {
		t.Error("experience search should be enabled by default")
	}
	if !plan.SearchTools {
		t.Error("tool search should be enabled by default")
	}
	if plan.SearchTaskResults {
		t.Error("task result search should be disabled by default")
	}

	if plan.KnowledgeWeight != 0.4 {
		t.Errorf("knowledge weight should be 0.4, got %f", plan.KnowledgeWeight)
	}
	if plan.ExperienceWeight != 0.3 {
		t.Errorf("experience weight should be 0.3, got %f", plan.ExperienceWeight)
	}
	if plan.ToolsWeight != 0.2 {
		t.Errorf("tool weight should be 0.2, got %f", plan.ToolsWeight)
	}
	if plan.TaskResultsWeight != 0.1 {
		t.Errorf("task result weight should be 0.1, got %f", plan.TaskResultsWeight)
	}

	if plan.EnableQueryRewrite {
		t.Error("query rewrite should be disabled by default")
	}
	if !plan.EnableKeywordSearch {
		t.Error("keyword search should be enabled by default")
	}
	if !plan.EnableTimeDecay {
		t.Error("time decay should be enabled by default")
	}

	if plan.TopK != 10 {
		t.Errorf("top_k should be 10 by default, got %d", plan.TopK)
	}
}

// TestCalculateTimeDecay tests the time decay calculation.
func TestCalculateTimeDecay(t *testing.T) {
	service := &RetrievalService{}

	now := time.Now()

	// Test recent content (should have high decay factor)
	recentDecay := service.calculateTimeDecay(now.Add(-1 * time.Hour))
	if recentDecay <= 0.9 {
		t.Errorf("recent content should have high decay factor > 0.9, got %f", recentDecay)
	}

	// Test old content (should have lower decay factor)
	oldDecay := service.calculateTimeDecay(now.Add(-30 * 24 * time.Hour))
	if oldDecay >= recentDecay {
		t.Error("old content should have lower decay factor than recent content")
	}
	if oldDecay < 0.1 {
		t.Errorf("decay factor should not go below 0.1, got %f", oldDecay)
	}

	// Test very old content (should hit minimum threshold)
	veryOldDecay := service.calculateTimeDecay(now.Add(-365 * 24 * time.Hour))
	if veryOldDecay != 0.1 {
		t.Errorf("very old content should hit minimum decay threshold 0.1, got %f", veryOldDecay)
	}
}

// TestFilterByScore tests score filtering.
func TestFilterByTestScore(t *testing.T) {
	service := &RetrievalService{}

	results := []*SearchResult{
		{ID: "1", Score: 0.9},
		{ID: "2", Score: 0.7},
		{ID: "3", Score: 0.5},
		{ID: "4", Score: 0.3},
		{ID: "5", Score: 0.1},
	}

	// Test filtering with minimum score 0.5
	filtered := service.filterByScore(results, 0.5)
	if len(filtered) != 3 {
		t.Errorf("should return 3 results with score >= 0.5, got %d", len(filtered))
	}

	// Test filtering with minimum score 0.0 (should return all)
	filtered = service.filterByScore(results, 0.0)
	if len(filtered) != 5 {
		t.Errorf("should return all results when min score is 0, got %d", len(filtered))
	}

	// Test filtering with high minimum score
	filtered = service.filterByScore(results, 0.8)
	if len(filtered) != 1 {
		t.Errorf("should return only 1 result with score >= 0.8, got %d", len(filtered))
	}
}

// TestMergeAndRank tests the merge and rank functionality.
func TestMergeAndRank(t *testing.T) {
	service := &RetrievalService{}
	plan := DefaultRetrievalPlan()

	now := time.Now()

	// Create mock vector results
	vectorResults := []*SearchResult{
		{ID: "1", Score: 0.9, Source: "knowledge", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "2", Score: 0.8, Source: "experience", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "3", Score: 0.7, Source: "tool", CreatedAt: now.Add(-3 * time.Hour)},
	}

	// Create mock keyword results (some overlapping IDs)
	keywordResults := []*SearchResult{
		{ID: "2", Score: 0.6, Source: "experience", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "4", Score: 0.5, Source: "knowledge", CreatedAt: now.Add(-4 * time.Hour)},
		{ID: "5", Score: 0.4, Source: "tool", CreatedAt: now.Add(-5 * time.Hour)},
	}

	// Merge and rank
	merged := service.mergeAndRank(context.Background(), vectorResults, keywordResults, plan)

	// Should have 5 unique results
	if len(merged) != 5 {
		t.Errorf("should have 5 unique results, got %d", len(merged))
	}

	// Results should be sorted by score (descending)
	for i := 1; i < len(merged); i++ {
		if merged[i-1].Score < merged[i].Score {
			t.Errorf("results should be sorted by score in descending order, got %f < %f at position %d",
				merged[i-1].Score, merged[i].Score, i)
		}
	}

	// Overlapping result (ID: 2) should have combined score

	result2 := findResultByID(merged, "2")

	if result2 == nil {

		t.Fatal("result with ID 2 should exist")

	}

	// Score should reflect combination from both sources

	// Since it appears in both vector (position 1) and keyword (position 0) results

	// the combined score should be: (0.8/2 * 0.3) + (0.6/1 * 0.3) = 0.12 + 0.18 = 0.30 (approx)

	if result2.Score <= 0.1 {

		t.Errorf("combined score should reflect combination from both sources, got %f", result2.Score)

	}
}

// TestSearchRequestValidation tests search request validation.
func TestSearchRequestValidation(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name        string
		request     *SearchRequest
		expectError bool
		errorType   error
	}{
		{
			name: "valid request",
			request: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-123",
				TopK:     10,
			},
			expectError: false,
		},
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
			errorType:   errors.ErrInvalidArgument,
		},
		{
			name: "empty query",
			request: &SearchRequest{
				Query:    "",
				TenantID: "tenant-123",
				TopK:     10,
			},
			expectError: true,
			errorType:   errors.ErrInvalidArgument,
		},
		{
			name: "empty tenant ID",
			request: &SearchRequest{
				Query:    "test query",
				TenantID: "",
				TopK:     10,
			},
			expectError: true,
			errorType:   errors.ErrInvalidArgument,
		},
		{
			name: "zero TopK (should be set to default)",
			request: &SearchRequest{
				Query:    "test query",
				TenantID: "tenant-123",
				TopK:     0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRequest(tt.request)

			if tt.expectError {
				if err == nil {
					t.Error("should return error")
				}
				if tt.errorType != nil && err != tt.errorType {
					t.Errorf("should return expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("should not return error, got %v", err)
				}
				if tt.request != nil && tt.request.TopK == 0 {
					if tt.request.TopK != 10 {
						t.Errorf("TopK should be set to default value 10, got %d", tt.request.TopK)
					}
				}
			}
		})
	}
}

// TestShouldRewriteQuery tests query rewrite decision logic.
func TestShouldRewriteQuery(t *testing.T) {
	service := &RetrievalService{}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "short query",
			query:    "test",
			expected: false,
		},
		{
			name:     "empty query",
			query:    "",
			expected: false,
		},
		{
			name:     "Chinese 'how' query",
			query:    "如何使用这个功能",
			expected: true,
		},
		{
			name:     "Chinese 'what' query",
			query:    "什么是机器学习",
			expected: true,
		},
		{
			name:     "English 'how' query",
			query:    "how do I use this feature",
			expected: true,
		},
		{
			name:     "English 'what' query",
			query:    "what is machine learning",
			expected: true,
		},
		{
			name:     "simple statement",
			query:    "this is a simple statement about something",
			expected: false,
		},
		{
			name:     "why question",
			query:    "why does this happen",
			expected: true,
		},
		{
			name:     "explain query",
			query:    "explain the concept of recursion",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.shouldRewriteQuery(tt.query)
			if result != tt.expected {
				t.Errorf("query rewrite decision should be %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCountResultsBySource tests result counting by source.
func TestCountResultsBySource(t *testing.T) {
	service := &RetrievalService{}

	results := []*SearchResult{
		{ID: "1", Source: "knowledge"},
		{ID: "2", Source: "knowledge"},
		{ID: "3", Source: "experience"},
		{ID: "4", Source: "tool"},
		{ID: "5", Source: "knowledge"},
		{ID: "6", Source: "tool"},
	}

	counts := service.countResultsBySource(results)

	if counts["knowledge"] != 3 {
		t.Errorf("should count 3 knowledge results, got %d", counts["knowledge"])
	}
	if counts["experience"] != 1 {
		t.Errorf("should count 1 experience result, got %d", counts["experience"])
	}
	if counts["tool"] != 2 {
		t.Errorf("should count 2 tool results, got %d", counts["tool"])
	}
	if counts["task_result"] != 0 {
		t.Errorf("should count 0 task result results, got %d", counts["task_result"])
	}
}

// Helper functions

func findResultByID(results []*SearchResult, id string) *SearchResult {
	for _, result := range results {
		if result.ID == id {
			return result
		}
	}
	return nil
}
