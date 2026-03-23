// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"context"
	"testing"
)

func TestConflictResolver_ResolveConflict(t *testing.T) {
	repo := NewMockExperienceRepository([]Experience{})
	resolver := NewConflictResolver(repo)

	tests := []struct {
		name           string
		newMemory      *Experience
		existingMemory *Experience
		expected       ResolutionStrategy
	}{
		{
			name:           "nil old memory",
			newMemory:      &Experience{Problem: "test", Solution: "fix", Confidence: 0.5},
			existingMemory: nil,
			expected:       ReplaceOld,
		},
		{
			name:           "new memory higher confidence",
			newMemory:      &Experience{Problem: "test", Solution: "fix", Confidence: 0.9},
			existingMemory: &Experience{Problem: "test", Solution: "old fix", Confidence: 0.5},
			expected:       ReplaceOld,
		},
		{
			name:           "old memory higher confidence",
			newMemory:      &Experience{Problem: "test", Solution: "fix", Confidence: 0.5},
			existingMemory: &Experience{Problem: "test", Solution: "old fix", Confidence: 0.9},
			expected:       KeepBoth,
		},
		{
			name:           "equal confidence",
			newMemory:      &Experience{Problem: "test", Solution: "fix", Confidence: 0.7},
			existingMemory: &Experience{Problem: "test", Solution: "old fix", Confidence: 0.7},
			expected:       KeepBoth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveConflict(tt.newMemory, tt.existingMemory)
			if result != tt.expected {
				t.Errorf("ResolveConflict() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConflictResolver_cosineSimilarity(t *testing.T) {
	repo := NewMockExperienceRepository([]Experience{})
	resolver := NewConflictResolver(repo)

	tests := []struct {
		name     string
		v1       []float64
		v2       []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			v1:       []float64{1.0, 0.0, 0.0},
			v2:       []float64{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			v1:       []float64{1.0, 0.0},
			v2:       []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "different dimensions",
			v1:       []float64{1.0, 0.0},
			v2:       []float64{1.0, 0.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "empty vectors",
			v1:       []float64{},
			v2:       []float64{},
			expected: 0.0,
		},
		{
			name:     "normalized vectors",
			v1:       []float64{0.707, 0.707},
			v2:       []float64{0.707, 0.707},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.cosineSimilarity(tt.v1, tt.v2)
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("cosineSimilarity() = %v, want ~%v", result, tt.expected)
			}
		})
	}
}

func TestConflictResolver_DetectConflict(t *testing.T) {
	existingExperiences := []Experience{
		{
			Problem:  "docker container won't start",
			Solution: "restart docker daemon",
		},
		{
			Problem:  "I prefer Go",
			Solution: "Use Go examples",
		},
	}

	repo := NewMockExperienceRepository(existingExperiences)
	resolver := NewConflictResolver(repo)

	tests := []struct {
		name           string
		newMemory      *Experience
		tenantID       string
		expectConflict bool
	}{
		{
			name: "new memory",
			newMemory: &Experience{
				Problem:  "docker error",
				Solution: "fix it",
			},
			tenantID:       "test",
			expectConflict: false,
		},
		{
			name: "low similarity",
			newMemory: &Experience{
				Problem:  "database connection error",
				Solution: "update connection string",
			},
			tenantID:       "test",
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			conflict, err := resolver.DetectConflict(ctx, tt.newMemory, tt.tenantID)

			if err != nil {
				t.Errorf("DetectConflict() returned error: %v", err)
			}

			if tt.expectConflict && conflict == nil {
				t.Error("Expected conflict but got none")
			}

			if !tt.expectConflict && conflict != nil {
				t.Error("Expected no conflict but got one")
			}
		})
	}
}
