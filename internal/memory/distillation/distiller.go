// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"context"
	"fmt"
	"time"

	"goagent/internal/storage/postgres/embedding"
)

// DistillationConfig holds configuration for the distillation process.
type DistillationConfig struct {
	// MinImportance is the minimum importance score for memories to be kept.
	MinImportance float64

	// ConflictThreshold is the similarity threshold for conflict detection.
	ConflictThreshold float64

	// MaxMemoriesPerDistillation is the maximum number of memories to keep per distillation.
	MaxMemoriesPerDistillation int

	// MaxSolutionsPerTenant is the global cap on solution memories per tenant.
	MaxSolutionsPerTenant int

	// EnableCodeFilter enables code block filtering.
	EnableCodeFilter bool

	// EnableStacktraceFilter enables stacktrace filtering.
	EnableStacktraceFilter bool

	// EnableLogFilter enables log filtering.
	EnableLogFilter bool

	// EnableMarkdownTableFilter enables markdown table filtering.
	EnableMarkdownTableFilter bool

	// EnableCrossTurnExtraction enables cross-turn conversation extraction.
	EnableCrossTurnExtraction bool

	// EnableLengthBonus enables length bonus in importance scoring.
	EnableLengthBonus bool

	// LengthThreshold is the threshold for length bonus.
	LengthThreshold int

	// LengthBonus is the bonus value for length threshold.
	LengthBonus float64

	// TopNBeforeConflict enables top-N filtering before conflict detection.
	TopNBeforeConflict bool

	// ConflictSearchLimit is the limit for vector search in conflict detection.
	ConflictSearchLimit int

	// PrecisionOverRecall prioritizes precision over recall.
	PrecisionOverRecall bool
}

// DefaultDistillationConfig returns the default configuration for distillation.
func DefaultDistillationConfig() *DistillationConfig {
	return &DistillationConfig{
		MinImportance:                0.6,
		ConflictThreshold:            0.85,
		MaxMemoriesPerDistillation:   3,
		MaxSolutionsPerTenant:        5000,
		EnableCodeFilter:             true,
		EnableStacktraceFilter:       true,
		EnableLogFilter:              true,
		EnableMarkdownTableFilter:    true,
		EnableCrossTurnExtraction:    true,
		EnableLengthBonus:            true,
		LengthThreshold:              60,
		LengthBonus:                  0.1,
		TopNBeforeConflict:           true,
		ConflictSearchLimit:          5,
		PrecisionOverRecall:          true,
	}
}

// DistillationMetrics holds metrics for the distillation process.
type DistillationMetrics struct {
	AttemptTotal      int64
	SuccessTotal      int64
	FilteredNoise     int64
	FilteredSecurity  int64
	ConflictResolved  int64
	MemoriesCreated   int64
}

// Distiller is the unified distillation engine that orchestrates all components.
type Distiller struct {
	config        *DistillationConfig
	extractor     *ExperienceExtractor
	classifier    *MemoryClassifier
	scorer        *ImportanceScorer
	resolver      *ConflictResolver
	noiseFilter   *NoiseFilter
	embedder      embedding.EmbeddingService
	repo          ExperienceRepository
	metrics       *DistillationMetrics
}

// NewDistiller creates a new Distiller instance.
//
// Args:
//   config - distillation configuration.
//   embedder - embedding service for generating vectors.
//   repo - experience repository for storage and retrieval.
//
// Returns:
//   *Distiller - configured distiller instance.
func NewDistiller(config *DistillationConfig, embedder embedding.EmbeddingService, repo ExperienceRepository) *Distiller {
	if config == nil {
		config = DefaultDistillationConfig()
	}

	// Create noise filter with configuration
	noiseFilterConfig := &NoiseFilterConfig{
		EnableCodeFilter:         config.EnableCodeFilter,
		EnableStacktraceFilter:   config.EnableStacktraceFilter,
		EnableLogFilter:          config.EnableLogFilter,
		EnableMarkdownTableFilter: config.EnableMarkdownTableFilter,
	}

	return &Distiller{
		config:        config,
		extractor:     NewExperienceExtractorWithConfig(config.EnableCrossTurnExtraction),
		classifier:    NewMemoryClassifier(),
		scorer:        NewImportanceScorerWithConfig(config.MinImportance, config.EnableLengthBonus),
		resolver:      NewConflictResolverWithConfig(repo, config.ConflictThreshold, config.ConflictSearchLimit),
		noiseFilter:   NewNoiseFilterWithConfig(noiseFilterConfig),
		embedder:      embedder,
		repo:          repo,
		metrics:       &DistillationMetrics{},
	}
}

// DistillConversation distills memories from a conversation.
// This is the main entry point for the distillation process.
//
// Args:
//   ctx - operation context.
//   conversationID - unique identifier for the conversation.
//   messages - conversation messages.
//   tenantID - tenant ID for multi-tenancy.
//   userID - user ID for the conversation.
//
// Returns:
//   []Memory - distilled memories.
//   error - any error encountered.
func (d *Distiller) DistillConversation(ctx context.Context, conversationID string, messages []Message, tenantID, userID string) ([]Memory, error) {
	d.metrics.AttemptTotal++

	if ctx.Err() != nil {
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Step 1: Extract experiences
	experiences := d.extractor.ExtractExperiences(messages)
	if len(experiences) == 0 {
		d.metrics.FilteredNoise++
		return []Memory{}, nil
	}

	// Step 2: Classify experiences and create memory candidates
	var memories []Memory
	for _, exp := range experiences {
		// Security filter
		if !SecurityFilter(exp.Problem) || !SecurityFilter(exp.Solution) {
			d.metrics.FilteredSecurity++
			continue
		}

		// Noise filter for solution
		if d.noiseFilter.IsNoise(exp.Solution) {
			d.metrics.FilteredNoise++
			continue
		}

		// Classify memory type
		memoryType := d.classifier.ClassifyMemory(&exp)

		// Score importance
		problem := exp.Problem
		solution := exp.Solution
		score := d.scorer.ScoreMemory(memoryType, problem, solution)

		// Update confidence with importance score
		exp.Confidence = score

		// Skip low importance memories
		if !d.scorer.ShouldKeep(score) {
			d.metrics.FilteredNoise++
			continue
		}

		// Create memory
		memory := Memory{
			Type:       memoryType,
			Content:    FormatExperience(&exp),
			Importance: score,
			Source:     conversationID,
			CreatedAt:  time.Now(),
			Metadata: map[string]interface{}{
				"memory_type":        memoryType.String(),
				"conversation_id":    conversationID,
				"source":             "distillation",
				"confidence":         exp.Confidence,
				"extraction_method":  "direct", // TODO: Track cross-turn vs direct
				"problem":            problem,
				"solution":           solution,
				"tenant_id":          tenantID,
				"user_id":            userID,
			},
		}

		memories = append(memories, memory)
	}

	// Step 3: Top-N filtering (before conflict detection for performance)
	if d.config.TopNBeforeConflict && len(memories) > d.config.MaxMemoriesPerDistillation {
		// Convert to experiences for scoring
		var exps []Experience
		for _, mem := range memories {
			exps = append(exps, Experience{
				Problem:    mem.Metadata["problem"].(string),
				Solution:   mem.Metadata["solution"].(string),
				Confidence: mem.Importance,
			})
		}

		filtered := d.scorer.TopNFilter(exps, d.config.MaxMemoriesPerDistillation)

		// Rebuild memories from filtered experiences
		memories = memories[:len(filtered)]
		for i, exp := range filtered {
			memories[i].Importance = exp.Confidence
			memories[i].Metadata["confidence"] = exp.Confidence
		}
	}

	// Step 4: Conflict detection and resolution
	var finalMemories []Memory
	for _, memory := range memories {
		// Generate embedding for conflict detection
		// Use "problem → solution" format for better retrieval
		embeddingText := fmt.Sprintf("%s → %s", memory.Metadata["problem"], memory.Metadata["solution"])
		embedding, err := d.embedder.EmbedWithPrefix(ctx, embeddingText, "memory:")
		if err != nil {
			// TODO: Log error but continue with other memories
			continue
		}
		memory.Vector = embedding

		// Detect conflicts (skip for now since Experience doesn't have Vector field)
		// TODO: Implement proper conflict detection when vector storage is available

		// Keep the memory
		finalMemories = append(finalMemories, memory)
	}

	// Step 5: Final Top-N filtering (after conflict resolution)
	if len(finalMemories) > d.config.MaxMemoriesPerDistillation {
		// Sort by importance and limit
		for i := 0; i < len(finalMemories); i++ {
			for j := i + 1; j < len(finalMemories); j++ {
				if finalMemories[j].Importance > finalMemories[i].Importance {
					finalMemories[i], finalMemories[j] = finalMemories[j], finalMemories[i]
				}
			}
		}
		finalMemories = finalMemories[:d.config.MaxMemoriesPerDistillation]
	}

	// Step 6: Enforce solution cap
	err := d.enforceSolutionCap(ctx, tenantID)
	if err != nil {
		// TODO: Log error but continue
	}

	d.metrics.SuccessTotal++
	d.metrics.MemoriesCreated += int64(len(finalMemories))

	return finalMemories, nil
}

// enforceSolutionCap enforces the global cap on solution memories per tenant.
//
// Args:
//   ctx - operation context.
//   tenantID - tenant ID for multi-tenancy.
//
// Returns:
//   error - any error encountered.
func (d *Distiller) enforceSolutionCap(ctx context.Context, tenantID string) error {
	// TODO: Implement solution cap enforcement
	// This should:
	// 1. Count current solutions for the tenant
	// 2. If count > MaxSolutionsPerTenant, delete lowest importance ones
	return nil
}

// GetMetrics returns the current distillation metrics.
//
// Returns:
//   *DistillationMetrics - the metrics.
func (d *Distiller) GetMetrics() *DistillationMetrics {
	return d.metrics
}

// ResetMetrics resets the distillation metrics.
func (d *Distiller) ResetMetrics() {
	d.metrics = &DistillationMetrics{}
}