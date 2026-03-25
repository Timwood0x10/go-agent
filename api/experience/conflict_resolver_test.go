// Package experience provides tests for experience conflict resolver.
package experience

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewConflictResolver tests the creation of a new ConflictResolver.
func TestNewConflictResolver(t *testing.T) {
	resolver := NewConflictResolver()

	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.logger)
	assert.Equal(t, 0.9, resolver.problemSimilarityThreshold)
}

// TestConflictResolver_Configure tests the configuration of conflict resolver.
func TestConflictResolver_Configure(t *testing.T) {
	resolver := NewConflictResolver()

	tests := []struct {
		name        string
		threshold   float64
		expectError bool
	}{
		{
			name:        "valid threshold (high)",
			threshold:   0.95,
			expectError: false,
		},
		{
			name:        "valid threshold (low)",
			threshold:   0.7,
			expectError: false,
		},
		{
			name:        "invalid threshold (zero)",
			threshold:   0.0,
			expectError: true,
		},
		{
			name:        "invalid threshold (negative)",
			threshold:   -0.1,
			expectError: true,
		},
		{
			name:        "invalid threshold (greater than 1)",
			threshold:   1.1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.Configure(tt.threshold)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.threshold, resolver.problemSimilarityThreshold)
			}
		})
	}
}

// TestDetectConflictGroups tests conflict group detection.
func TestDetectConflictGroups(t *testing.T) {
	ctx := context.Background()
	resolver := NewConflictResolver()

	tests := []struct {
		name               string
		experiences        []*Experience
		expectedGroupCount int
		description        string
	}{
		{
			name:               "empty list",
			experiences:        []*Experience{},
			expectedGroupCount: 1, // Returns [[]] for empty input
			description:        "Should return single empty group for empty input",
		},
		{
			name: "single experience",
			experiences: []*Experience{
				{
					ID:        "exp1",
					Problem:   "Database optimization",
					Solution:  "Add index",
					Embedding: []float64{0.1, 0.2, 0.3},
				},
			},
			expectedGroupCount: 1,
			description:        "Should return single group for single experience",
		},
		{
			name: "no conflicts (different problems)",
			experiences: []*Experience{
				{
					ID:        "exp1",
					Problem:   "Database optimization",
					Solution:  "Add index",
					Embedding: []float64{1.0, 0.0, 0.0},
				},
				{
					ID:        "exp2",
					Problem:   "Memory leak fix",
					Solution:  "Add context cancellation",
					Embedding: []float64{0.0, 1.0, 0.0}, // Very different (orthogonal)
				},
				{
					ID:        "exp3",
					Problem:   "Rate limiting",
					Solution:  "Token bucket",
					Embedding: []float64{0.0, 0.0, 1.0}, // Different (orthogonal)
				},
			},
			expectedGroupCount: 3,
			description:        "Should return separate groups for different problems",
		},
		{
			name: "one conflict group",
			experiences: []*Experience{
				{
					ID:        "exp1",
					Problem:   "Database optimization",
					Solution:  "Add index",
					Embedding: []float64{0.1, 0.2, 0.3},
				},
				{
					ID:        "exp2",
					Problem:   "Database query optimization",
					Solution:  "Add composite index",
					Embedding: []float64{0.12, 0.22, 0.32}, // Very similar (> 0.9)
				},
				{
					ID:        "exp3",
					Problem:   "Memory leak fix",
					Solution:  "Add context cancellation",
					Embedding: []float64{0.9, 0.8, 0.7}, // Different
				},
			},
			expectedGroupCount: 2,
			description:        "Should group similar experiences together",
		},
		{
			name: "multiple conflict groups",
			experiences: []*Experience{
				{
					ID:        "exp1",
					Problem:   "Database optimization",
					Solution:  "Add index",
					Embedding: []float64{1.0, 0.0, 0.0},
				},
				{
					ID:        "exp2",
					Problem:   "Database query optimization",
					Solution:  "Add composite index",
					Embedding: []float64{0.95, 0.0, 0.0}, // Similar to exp1 (> 0.9)
				},
				{
					ID:        "exp3",
					Problem:   "Memory leak fix",
					Solution:  "Add context cancellation",
					Embedding: []float64{0.0, 1.0, 0.0},
				},
				{
					ID:        "exp4",
					Problem:   "Fix memory leak",
					Solution:  "Add defer statements",
					Embedding: []float64{0.0, 0.95, 0.0}, // Similar to exp3 (> 0.9)
				},
			},
			expectedGroupCount: 2,
			description:        "Should create multiple conflict groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := resolver.DetectConflictGroups(ctx, tt.experiences)

			assert.Equal(t, tt.expectedGroupCount, len(groups), tt.description)

			// Verify that all experiences are in exactly one group
			totalExperiences := 0
			seenIDs := make(map[string]bool)

			for _, group := range groups {
				for _, exp := range group {
					assert.False(t, seenIDs[exp.ID], "Experience %s appears in multiple groups", exp.ID)
					seenIDs[exp.ID] = true
					totalExperiences++
				}
			}

			assert.Equal(t, len(tt.experiences), totalExperiences, "Not all experiences were grouped")
		})
	}
}

// TestResolve tests conflict resolution.
func TestResolve(t *testing.T) {
	ctx := context.Background()
	resolver := NewConflictResolver()

	tests := []struct {
		name              string
		rankedExperiences []*RankedExperience
		expectedCount     int
		description       string
	}{
		{
			name:              "empty list",
			rankedExperiences: []*RankedExperience{},
			expectedCount:     0,
			description:       "Should return empty list for empty input",
		},
		{
			name: "single experience",
			rankedExperiences: []*RankedExperience{
				{
					Experience: &Experience{
						ID:        "exp1",
						Problem:   "Database optimization",
						Solution:  "Add index",
						Embedding: []float64{0.1, 0.2, 0.3},
					},
					FinalScore: 0.8,
				},
			},
			expectedCount: 1,
			description:   "Should return single experience",
		},
		{
			name: "no conflicts",
			rankedExperiences: []*RankedExperience{
				{
					Experience: &Experience{
						ID:        "exp1",
						Problem:   "Database optimization",
						Solution:  "Add index",
						Embedding: []float64{0.1, 0.2, 0.3},
					},
					FinalScore: 0.8,
				},
				{
					Experience: &Experience{
						ID:        "exp2",
						Problem:   "Memory leak fix",
						Solution:  "Add context cancellation",
						Embedding: []float64{0.9, 0.8, 0.7},
					},
					FinalScore: 0.7,
				},
			},
			expectedCount: 2,
			description:   "Should return all experiences when no conflicts",
		},
		{
			name: "with conflicts (should select best per group)",
			rankedExperiences: []*RankedExperience{
				{
					Experience: &Experience{
						ID:        "exp1",
						Problem:   "Database optimization",
						Solution:  "Add index",
						Embedding: []float64{1.0, 0.0, 0.0},
					},
					FinalScore: 0.8, // Higher score
				},
				{
					Experience: &Experience{
						ID:        "exp2",
						Problem:   "Database query optimization",
						Solution:  "Add composite index",
						Embedding: []float64{0.95, 0.0, 0.0}, // Similar to exp1
					},
					FinalScore: 0.7, // Lower score
				},
				{
					Experience: &Experience{
						ID:        "exp3",
						Problem:   "Memory leak fix",
						Solution:  "Add context cancellation",
						Embedding: []float64{0.0, 1.0, 0.0}, // Different group
					},
					FinalScore: 0.9,
				},
			},
			expectedCount: 2, // Should select exp1 (best of group 1) and exp3 (group 2)
			description:   "Should select best experience per conflict group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := resolver.Resolve(ctx, tt.rankedExperiences)

			assert.Equal(t, tt.expectedCount, len(resolved), tt.description)

			// Verify that conflict resolved flags are set
			for _, ranked := range tt.rankedExperiences {
				if len(tt.rankedExperiences) > 1 && tt.name == "with conflicts (should select best per group)" {
					assert.True(t, ranked.ConflictChecked, "Conflict checked flag should be set")
				}
			}
		})
	}
}

// TestCosineSimilarity tests the cosine similarity calculation.
func TestCosineSimilarity(t *testing.T) {
	resolver := NewConflictResolver()

	tests := []struct {
		name     string
		vec1     []float64
		vec2     []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			vec1:     []float64{1.0, 2.0, 3.0},
			vec2:     []float64{1.0, 2.0, 3.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			vec1:     []float64{1.0, 0.0, 0.0},
			vec2:     []float64{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			vec1:     []float64{1.0, 1.0, 1.0},
			vec2:     []float64{-1.0, -1.0, -1.0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			vec1:     []float64{1.0, 2.0, 3.0},
			vec2:     []float64{1.1, 2.1, 3.1},
			expected: 0.999,
		},
		{
			name:     "different length vectors",
			vec1:     []float64{1.0, 2.0},
			vec2:     []float64{1.0, 2.0, 3.0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			vec1:     []float64{0.0, 0.0, 0.0},
			vec2:     []float64{1.0, 2.0, 3.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.cosineSimilarity(tt.vec1, tt.vec2)

			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

// TestFindBestInGroup tests finding the best experience in a group.
func TestFindBestInGroup(t *testing.T) {
	resolver := NewConflictResolver()

	experiences := []*Experience{
		{ID: "exp1", Problem: "Test1", Solution: "Test1", Embedding: []float64{0.1}},
		{ID: "exp2", Problem: "Test2", Solution: "Test2", Embedding: []float64{0.2}},
		{ID: "exp3", Problem: "Test3", Solution: "Test3", Embedding: []float64{0.3}},
	}

	rankedExperiences := []*RankedExperience{
		{Experience: experiences[0], FinalScore: 0.7},
		{Experience: experiences[1], FinalScore: 0.9}, // Highest score
		{Experience: experiences[2], FinalScore: 0.8},
	}

	best := resolver.findBestInGroup(experiences, rankedExperiences)

	assert.Equal(t, "exp2", best.ID)
}
