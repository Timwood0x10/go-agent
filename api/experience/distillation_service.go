// Package experience provides experience distillation service.
// This package re-exports types and services from internal/experience for backward compatibility.
package experience

import (
	internalexperience "goagent/internal/experience"
	"goagent/internal/llm"
	"goagent/internal/storage/postgres/embedding"
	"goagent/internal/storage/postgres/repositories"
)

// DistillationService re-exports from internal/experience.
type DistillationService = internalexperience.DistillationService

// ExtractedExperience re-exports from internal/experience.
type ExtractedExperience = internalexperience.ExtractedExperience

// NewDistillationService re-exports from internal/experience.
func NewDistillationService(
	llmClient *llm.Client,
	embeddingClient *embedding.EmbeddingClient,
	experienceRepo repositories.ExperienceRepositoryInterface,
) *DistillationService {
	return internalexperience.NewDistillationService(llmClient, embeddingClient, experienceRepo)
}
