// Package experience provides tests for experience ranking service.
package experience

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRankingService tests the creation of a new RankingService.
func TestNewRankingService(t *testing.T) {
	service := NewRankingService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	assert.Equal(t, 0.05, service.usageWeight)
	assert.Equal(t, 0.05, service.recencyWeight)
	assert.Equal(t, 30.0, service.recencyDays)
}

// TestDefaultRankingWeights tests the default ranking weights.
func TestDefaultRankingWeights(t *testing.T) {
	weights := DefaultRankingWeights()

	assert.NotNil(t, weights)
	assert.Equal(t, 0.05, weights.UsageWeight)
	assert.Equal(t, 0.05, weights.RecencyWeight)
	assert.Equal(t, 30.0, weights.RecencyDays)
}

// TestRankingService_Configure tests the configuration of ranking weights.
func TestRankingService_Configure(t *testing.T) {
	service := NewRankingService()

	tests := []struct {
		name        string
		weights     *RankingWeights
		expectError bool
	}{
		{
			name: "valid weights",
			weights: &RankingWeights{
				UsageWeight:   0.1,
				RecencyWeight: 0.1,
				RecencyDays:   60.0,
			},
			expectError: false,
		},
		{
			name:        "nil weights",
			weights:     nil,
			expectError: false,
		},
		{
			name: "invalid usage weight (too high)",
			weights: &RankingWeights{
				UsageWeight:   1.5,
				RecencyWeight: 0.05,
				RecencyDays:   30.0,
			},
			expectError: true,
		},
		{
			name: "invalid usage weight (negative)",
			weights: &RankingWeights{
				UsageWeight:   -0.1,
				RecencyWeight: 0.05,
				RecencyDays:   30.0,
			},
			expectError: true,
		},
		{
			name: "invalid recency weight (too high)",
			weights: &RankingWeights{
				UsageWeight:   0.05,
				RecencyWeight: 1.5,
				RecencyDays:   30.0,
			},
			expectError: true,
		},
		{
			name: "invalid recency days (zero)",
			weights: &RankingWeights{
				UsageWeight:   0.05,
				RecencyWeight: 0.05,
				RecencyDays:   0,
			},
			expectError: true,
		},
		{
			name: "invalid recency days (negative)",
			weights: &RankingWeights{
				UsageWeight:   0.05,
				RecencyWeight: 0.05,
				RecencyDays:   -10.0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Configure(tt.weights)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRank tests the ranking method.
func TestRank(t *testing.T) {
	ctx := context.Background()
	service := NewRankingService()

	now := time.Now()

	experiences := []*Experience{
		{
			ID:         "exp1",
			Problem:    "Database query optimization",
			Solution:   "Add index on user_id",
			UsageCount: 10,
			CreatedAt:  now.Add(-24 * time.Hour), // 1 day ago
			Embedding:  []float64{0.1, 0.2, 0.3},
		},
		{
			ID:         "exp2",
			Problem:    "Memory leak fix",
			Solution:   "Add context cancellation",
			UsageCount: 5,
			CreatedAt:  now.Add(-7 * 24 * time.Hour), // 7 days ago
			Embedding:  []float64{0.2, 0.3, 0.4},
		},
		{
			ID:         "exp3",
			Problem:    "Rate limiting implementation",
			Solution:   "Token bucket algorithm",
			UsageCount: 20,
			CreatedAt:  now.Add(-3 * 24 * time.Hour), // 3 days ago
			Embedding:  []float64{0.3, 0.4, 0.5},
		},
	}

	baseScores := []float64{0.8, 0.7, 0.6}

	// Run ranking
	ranked := service.Rank(ctx, experiences, baseScores)

	// Assertions
	require.Len(t, ranked, 3)

	// Check that results are sorted by final score (descending)
	for i := 0; i < len(ranked)-1; i++ {
		assert.GreaterOrEqual(t, ranked[i].FinalScore, ranked[i+1].FinalScore)
	}

	// Check that all scores are calculated
	for _, r := range ranked {
		assert.GreaterOrEqual(t, r.FinalScore, 0.0)
		assert.GreaterOrEqual(t, r.SemanticScore, 0.0)
		assert.GreaterOrEqual(t, r.UsageBoost, 0.0)
		assert.GreaterOrEqual(t, r.RecencyBoost, 0.0)
		assert.LessOrEqual(t, r.UsageBoost, 0.2) // Usage boost should be capped
	}
}

// TestRankWithEmptyList tests ranking with empty experience list.
func TestRankWithEmptyList(t *testing.T) {
	ctx := context.Background()
	service := NewRankingService()

	ranked := service.Rank(ctx, []*Experience{}, []float64{})

	assert.Empty(t, ranked)
}

// TestRankWithMismatchedLengths tests ranking with mismatched lengths.
func TestRankWithMismatchedLengths(t *testing.T) {
	ctx := context.Background()
	service := NewRankingService()

	experiences := []*Experience{
		{ID: "exp1", Problem: "Test", Solution: "Test", CreatedAt: time.Now(), Embedding: []float64{0.1}},
	}

	baseScores := []float64{0.8, 0.7} // Mismatched length

	ranked := service.Rank(ctx, experiences, baseScores)

	assert.Empty(t, ranked) // Should return empty on error
}

// TestCalculateUsageBoost tests the usage boost calculation.
func TestCalculateUsageBoost(t *testing.T) {
	service := NewRankingService()

	tests := []struct {
		name           string
		usageCount     int
		expectBoost    float64
		shouldBeCapped bool
	}{
		{
			name:        "zero usage",
			usageCount:  0,
			expectBoost: 0.0,
		},
		{
			name:        "low usage",
			usageCount:  1,
			expectBoost: 0.034, // log(2) * 0.05 ≈ 0.034
		},
		{
			name:        "medium usage",
			usageCount:  10,
			expectBoost: 0.115, // log(11) * 0.05 ≈ 0.115
		},
		{
			name:           "high usage (should be capped)",
			usageCount:     1000,
			expectBoost:    0.2, // Capped at 0.2
			shouldBeCapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boost := service.calculateUsageBoost(tt.usageCount)

			if tt.shouldBeCapped {
				assert.Equal(t, 0.2, boost)
			} else {
				assert.InDelta(t, tt.expectBoost, boost, 0.01)
			}
		})
	}
}

// TestCalculateRecencyBoost tests the recency boost calculation.
func TestCalculateRecencyBoost(t *testing.T) {
	service := NewRankingService()
	now := time.Now()

	tests := []struct {
		name        string
		ageDays     float64
		expectBoost float64
	}{
		{
			name:        "very recent (0 days)",
			ageDays:     0.0,
			expectBoost: 0.05, // exp(0) * 0.05 = 0.05
		},
		{
			name:        "recent (1 day)",
			ageDays:     1.0,
			expectBoost: 0.048, // exp(-1/30) * 0.05 ≈ 0.048
		},
		{
			name:        "medium age (15 days)",
			ageDays:     15.0,
			expectBoost: 0.030, // exp(-15/30) * 0.05 ≈ 0.030
		},
		{
			name:        "old (30 days - half-life)",
			ageDays:     30.0,
			expectBoost: 0.018, // exp(-1) * 0.05 ≈ 0.018
		},
		{
			name:        "very old (90 days)",
			ageDays:     90.0,
			expectBoost: 0.002, // exp(-3) * 0.05 ≈ 0.002
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdAt := now.Add(-time.Duration(tt.ageDays*24) * time.Hour)
			boost := service.calculateRecencyBoost(createdAt, now)

			assert.InDelta(t, tt.expectBoost, boost, 0.005)
		})
	}
}

// TestCalculateRecencyBoostWithZeroTime tests recency boost with zero time.
func TestCalculateRecencyBoostWithZeroTime(t *testing.T) {
	service := NewRankingService()
	now := time.Now()

	boost := service.calculateRecencyBoost(time.Time{}, now)

	assert.Equal(t, 0.0, boost)
}
