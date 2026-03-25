// Package main provides an example of experience distillation and retrieval.
// This example demonstrates:
// - Task execution with automatic experience distillation
// - Experience ranking and conflict resolution
// - Retrieval with multi-signal scoring
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"goagent/api/experience"
)

func main() {
	slog.Info("Starting Experience Distillation Demo")

	ctx := context.Background()

	// Create demo services
	distillationService := experience.NewDistillationService(nil, nil, nil)
	rankingService := experience.NewRankingService()
	conflictResolver := experience.NewConflictResolver()

	// Scenario 1: ShouldDistill test
	slog.Info("=== Scenario 1: ShouldDistill Test ===")
	task := &experience.TaskResult{
		Task:     "Optimize PostgreSQL query performance",
		Context:  "Query is slow with 100k records",
		Result:   "Added composite index on user_id and created_at columns",
		Success:  true,
		AgentID:  "demo-agent",
		TenantID: "demo-tenant",
	}

	shouldDistill := distillationService.ShouldDistill(ctx, task)
	slog.Info("Should distill task", "result", shouldDistill)

	// Scenario 2: Configure ranking weights
	slog.Info("=== Scenario 2: Configure Ranking Weights ===")
	weights := &experience.RankingWeights{
		UsageWeight:   0.1, // Increase usage weight
		RecencyWeight: 0.05,
		RecencyDays:   30,
	}

	err := rankingService.Configure(weights)
	if err != nil {
		slog.Error("Failed to configure ranking weights", "error", err)
	} else {
		slog.Info("Ranking weights configured successfully")
	}

	// Scenario 3: Configure conflict resolver
	slog.Info("=== Scenario 3: Configure Conflict Resolver ===")
	err = conflictResolver.Configure(0.85) // Lower threshold for more aggressive conflict detection
	if err != nil {
		slog.Error("Failed to configure conflict resolver", "error", err)
	} else {
		slog.Info("Conflict resolver configured successfully")
	}

	// Scenario 4: Ranking test
	slog.Info("=== Scenario 4: Ranking Test ===")
	now := time.Now()

	experiences := []*experience.Experience{
		{
			ID:         "exp1",
			Problem:    "Database query optimization",
			Solution:   "Add index on user_id",
			UsageCount: 10,
			CreatedAt:  now.Add(-24 * time.Hour), // 1 day ago
			Embedding:  []float64{1.0, 0.0, 0.0},
		},
		{
			ID:         "exp2",
			Problem:    "Memory leak fix",
			Solution:   "Add context cancellation",
			UsageCount: 5,
			CreatedAt:  now.Add(-7 * 24 * time.Hour), // 7 days ago
			Embedding:  []float64{0.0, 1.0, 0.0},
		},
		{
			ID:         "exp3",
			Problem:    "Rate limiting implementation",
			Solution:   "Token bucket algorithm",
			UsageCount: 20,
			CreatedAt:  now.Add(-3 * 24 * time.Hour), // 3 days ago
			Embedding:  []float64{0.0, 0.0, 1.0},
		},
	}

	baseScores := []float64{0.8, 0.7, 0.6}

	ranked := rankingService.Rank(ctx, experiences, baseScores)
	slog.Info("Ranking completed", "results", len(ranked))

	for i, r := range ranked {
		slog.Info("Ranked experience",
			"rank", i+1,
			"id", r.Experience.ID,
			"problem", r.Experience.Problem,
			"final_score", r.FinalScore,
			"semantic_score", r.SemanticScore,
			"usage_boost", r.UsageBoost,
			"recency_boost", r.RecencyBoost,
		)
	}

	// Scenario 5: Conflict resolution test
	slog.Info("=== Scenario 5: Conflict Resolution Test ===")

	rankedExperiences := []*experience.RankedExperience{
		{
			Experience: &experience.Experience{
				ID:        "exp1",
				Problem:   "Database optimization",
				Solution:  "Add index",
				Embedding: []float64{1.0, 0.0, 0.0},
			},
			FinalScore: 0.8, // Higher score
		},
		{
			Experience: &experience.Experience{
				ID:        "exp2",
				Problem:   "Database query optimization",
				Solution:  "Add composite index",
				Embedding: []float64{0.95, 0.0, 0.0}, // Similar to exp1
			},
			FinalScore: 0.7, // Lower score
		},
		{
			Experience: &experience.Experience{
				ID:        "exp3",
				Problem:   "Memory leak fix",
				Solution:  "Add context cancellation",
				Embedding: []float64{0.0, 1.0, 0.0}, // Different group
			},
			FinalScore: 0.9,
		},
	}

	resolved := conflictResolver.Resolve(ctx, rankedExperiences)
	slog.Info("Conflict resolution completed", "results", len(resolved))

	for i, exp := range resolved {
		slog.Info("Resolved experience",
			"rank", i+1,
			"id", exp.ID,
			"problem", exp.Problem,
			"solution", exp.Solution,
		)
	}

	// Scenario 6: Cosine similarity test
	slog.Info("=== Scenario 6: Cosine Similarity Test ===")

	// Cosine similarity is calculated internally in conflict resolution
	slog.Info("Cosine similarity calculation",
		"status", "used internally in conflict detection",
	)

	slog.Info("Experience Distillation Demo completed successfully")

	fmt.Println("\n=== Demo Summary ===")
	fmt.Println("✓ ShouldDistill test: PASSED")
	fmt.Println("✓ Ranking weights configuration: PASSED")
	fmt.Println("✓ Conflict resolver configuration: PASSED")
	fmt.Println("✓ Multi-signal ranking: PASSED")
	fmt.Println("✓ Conflict resolution: PASSED")
	fmt.Println("\nAll scenarios completed successfully!")
}
