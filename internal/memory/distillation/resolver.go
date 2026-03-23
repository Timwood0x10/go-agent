// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"context"
	"math"
)

// ConflictResolver detects and resolves memory conflicts.
type ConflictResolver struct {
	repo              ExperienceRepository
	conflictThreshold float64
	searchLimit       int
}

// NewConflictResolver creates a new ConflictResolver.
func NewConflictResolver(repo ExperienceRepository) *ConflictResolver {
	return &ConflictResolver{
		repo:              repo,
		conflictThreshold: 0.85,
		searchLimit:       5,
	}
}

// NewConflictResolverWithConfig creates a new ConflictResolver with custom configuration.
func NewConflictResolverWithConfig(repo ExperienceRepository, conflictThreshold float64, searchLimit int) *ConflictResolver {
	return &ConflictResolver{
		repo:              repo,
		conflictThreshold: conflictThreshold,
		searchLimit:       searchLimit,
	}
}

// ResolveConflict determines the resolution strategy for a conflict.
//
// Args:
//
//	newMemory - the new memory being added.
//	oldMemory - the conflicting existing memory.
//
// Returns:
//
//	ResolutionStrategy - the strategy to resolve the conflict.
func (r *ConflictResolver) ResolveConflict(newMemory *Experience, oldMemory *Experience) ResolutionStrategy {
	if oldMemory == nil {
		return ReplaceOld
	}

	// TODO: Use memory type from metadata when available
	// For now, default to replace for all types
	return ReplaceOld
}

// DetectConflict detects conflicts with existing memories.
//
// Args:
//
//	ctx - operation context.
//	memory - the memory to check for conflicts.
//	tenantID - tenant ID for multi-tenancy.
//
// Returns:
//
//	*Experience - the conflicting memory, or nil if no conflict.
//	error - any error encountered.
func (r *ConflictResolver) DetectConflict(ctx context.Context, memory *Experience, tenantID string) (*Experience, error) {
	// TODO: Implement conflict detection when vector storage is available
	// This requires:
	// 1. Search for similar experiences by vector
	// 2. Check similarity threshold
	// 3. Return conflicting experience if found
	return nil, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
//
// Args:
//
//	v1 - first vector.
//	v2 - second vector.
//
// Returns:
//
//	float64 - similarity score between 0 and 1.
func (r *ConflictResolver) cosineSimilarity(v1, v2 []float64) float64 {
	if len(v1) != len(v2) || len(v1) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := range v1 {
		dotProduct += v1[i] * v2[i]
		norm1 += v1[i] * v1[i]
		norm2 += v2[i] * v2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}
