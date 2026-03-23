// Package memory provides implementation for memory distillation operations.
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"goagent/internal/memory/distillation"
	"goagent/internal/storage/postgres/embedding"
)

// DistillationServiceImpl implements the DistillationService interface.
type DistillationServiceImpl struct {
	distiller *distillation.Distiller
	config    *DistillationConfig
	metrics   *DistillationMetrics
}

// NewDistillationService creates a new DistillationService instance.
//
// Args:
// distiller - internal distiller instance.
//
// Returns new distillation service instance.
func NewDistillationService(distiller *distillation.Distiller) *DistillationServiceImpl {
	if distiller == nil {
		return nil
	}

	return &DistillationServiceImpl{
		distiller: distiller,
		config:    defaultDistillationConfig(),
		metrics:   convertToAPIDistillationMetrics(distiller.GetMetrics()),
	}
}

// NewDistillationServiceWithEmbedder creates a new DistillationService with embedder and repository.
//
// Args:
// config - distillation configuration.
// embedder - embedding service for generating vectors.
// repo - experience repository for storage and retrieval.
//
// Returns new distillation service instance or error.
func NewDistillationServiceWithEmbedder(config *DistillationConfig, embedder embedding.EmbeddingService, repo ExperienceRepository) (*DistillationServiceImpl, error) {
	if config == nil {
		config = defaultDistillationConfig()
	}

	// Convert API config to internal config
	internalConfig := convertFromAPIDistillationConfig(config)

	// Create adapter for ExperienceRepository
	internalRepo := &experienceRepositoryAdapter{apiRepo: repo}

	// Create internal distiller
	internalDistiller := distillation.NewDistiller(internalConfig, embedder, internalRepo)

	return &DistillationServiceImpl{
		distiller: internalDistiller,
		config:    config,
		metrics:   &DistillationMetrics{},
	}, nil
}

// experienceRepositoryAdapter adapts API ExperienceRepository to internal ExperienceRepository interface
type experienceRepositoryAdapter struct {
	apiRepo ExperienceRepository
}

// SearchByVector implements internal ExperienceRepository interface

func (a *experienceRepositoryAdapter) SearchByVector(ctx interface{}, vector []float64, tenantID string, limit int) ([]distillation.Experience, error) {

	// Convert interface{} to context.Context if possible

	ctxTyped, ok := ctx.(context.Context)

	if !ok {

		return []distillation.Experience{}, nil

	}



	apiExperiences, err := a.apiRepo.SearchByVector(ctxTyped, vector, tenantID, limit)

	if err != nil {

		return nil, err

	}



	// Convert API experiences to internal experiences

	internalExperiences := make([]distillation.Experience, len(apiExperiences))

	for i, exp := range apiExperiences {

		internalExperiences[i] = distillation.Experience{

			Problem:         exp.Problem,

			Solution:        exp.Solution,

			Confidence:      exp.Confidence,

			ExtractionMethod: distillation.ExtractionMethod(exp.ExtractionMethod),

		}

	}



	return internalExperiences, nil

}

// GetByMemoryType implements internal ExperienceRepository interface
func (a *experienceRepositoryAdapter) GetByMemoryType(ctx interface{}, tenantID string, memoryType distillation.MemoryType) ([]distillation.Experience, error) {
	// Convert interface{} to context.Context if possible
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return []distillation.Experience{}, nil
	}

	apiExperiences, err := a.apiRepo.GetByMemoryType(ctxTyped, tenantID, MemoryType(memoryType))
	if err != nil {
		return nil, err
	}

	// Convert API experiences to internal experiences
	internalExperiences := make([]distillation.Experience, len(apiExperiences))
	for i, exp := range apiExperiences {
		internalExperiences[i] = distillation.Experience{
			Problem:          exp.Problem,
			Solution:         exp.Solution,
			Confidence:       exp.Confidence,
			ExtractionMethod: distillation.ExtractionMethod(exp.ExtractionMethod),
		}
	}

	return internalExperiences, nil
}

// Update implements internal ExperienceRepository interface
func (a *experienceRepositoryAdapter) Update(ctx interface{}, experience *distillation.Experience) error {
	// Convert interface{} to context.Context if possible
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return nil
	}

	apiExperience := &Experience{
		Problem:          experience.Problem,
		Solution:         experience.Solution,
		Confidence:       experience.Confidence,
		ExtractionMethod: ExtractionMethod(experience.ExtractionMethod),
	}

	return a.apiRepo.Update(ctxTyped, apiExperience)
}

// Delete implements internal ExperienceRepository interface
func (a *experienceRepositoryAdapter) Delete(ctx interface{}, id string) error {
	// Convert interface{} to context.Context if possible
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return nil
	}

	return a.apiRepo.Delete(ctxTyped, id)
}

// GetDistiller returns the internal distiller instance (for advanced usage).
func (s *DistillationServiceImpl) GetDistiller() interface{} {
	return s.distiller
}

// DistillConversation distills memories from a conversation.
//
// Args:
// ctx - operation context.
// conversationID - unique identifier for the conversation.
// messages - conversation messages.
// tenantID - tenant ID for multi-tenancy.
// userID - user ID for the conversation.
//
// Returns distilled memories or error.
func (s *DistillationServiceImpl) DistillConversation(ctx context.Context, conversationID string, messages []ConversationMessage, tenantID, userID string) ([]*DistilledMemory, error) {
	if conversationID == "" {
		return nil, ErrInvalidConversationID
	}
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}

	slog.Info("Starting memory distillation",
		"conversation_id", conversationID,
		"tenant_id", tenantID,
		"user_id", userID,
		"message_count", len(messages))

	// Convert API messages to internal messages
	internalMessages := make([]distillation.Message, len(messages))
	for i, msg := range messages {
		internalMessages[i] = distillation.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Execute distillation
	internalMemories, err := s.distiller.DistillConversation(ctx, conversationID, internalMessages, tenantID, userID)
	if err != nil {
		slog.Error("Memory distillation failed",
			"conversation_id", conversationID,
			"error", err)
		return nil, fmt.Errorf("distill conversation: %w", err)
	}

	// Convert internal memories to API memories
	apiMemories := make([]*DistilledMemory, len(internalMemories))
	for i, mem := range internalMemories {
		apiMemories[i] = convertToAPIDistilledMemory(&mem)
	}

	slog.Info("Memory distillation completed",
		"conversation_id", conversationID,
		"memories_created", len(apiMemories))

	// Update metrics
	s.metrics = convertToAPIDistillationMetrics(s.distiller.GetMetrics())

	return apiMemories, nil
}

// GetMetrics returns the current distillation metrics.
//
// Returns the metrics.
func (s *DistillationServiceImpl) GetMetrics() *DistillationMetrics {
	if s.distiller == nil {
		return s.metrics
	}

	return convertToAPIDistillationMetrics(s.distiller.GetMetrics())
}

// ResetMetrics resets the distillation metrics.
func (s *DistillationServiceImpl) ResetMetrics() {
	if s.distiller == nil {
		s.metrics = &DistillationMetrics{}
		return
	}

	s.distiller.ResetMetrics()
	s.metrics = &DistillationMetrics{}
}

// GetConfig returns the current distillation configuration.
//
// Returns the configuration.
func (s *DistillationServiceImpl) GetConfig() *DistillationConfig {
	return s.config
}

// UpdateConfig updates the distillation configuration.
//
// Args:
// config - new configuration.
//
// Returns error if update fails.
func (s *DistillationServiceImpl) UpdateConfig(config *DistillationConfig) error {
	if config == nil {
		return ErrInvalidConfig
	}

	if s.distiller == nil {
		s.config = config
		return nil
	}

	// Note: The internal distiller doesn't support runtime config updates yet
	// For now, just update the API config
	s.config = config

	slog.Info("Distillation config updated",
		"min_importance", config.MinImportance,
		"max_memories", config.MaxMemoriesPerDistillation)

	return nil
}

// Helper functions for converting between API and internal types

func convertToAPIDistilledMemory(mem *distillation.Memory) *DistilledMemory {
	if mem == nil {
		return nil
	}

	var expiresAt *time.Time
	if !mem.ExpiresAt.IsZero() {
		expiresAt = &mem.ExpiresAt
	}

	return &DistilledMemory{
		ID:         mem.ID,
		Type:       MemoryType(mem.Type),
		Content:    mem.Content,
		Importance: mem.Importance,
		Source:     mem.Source,
		TenantID:   getMetadataString(mem.Metadata, "tenant_id"),
		UserID:     getMetadataString(mem.Metadata, "user_id"),
		CreatedAt:  mem.CreatedAt,
		ExpiresAt:  expiresAt,
		Metadata:   mem.Metadata,
	}
}

func convertFromAPIDistillationConfig(config *DistillationConfig) *distillation.DistillationConfig {
	if config == nil {
		return distillation.DefaultDistillationConfig()
	}

	return &distillation.DistillationConfig{
		MinImportance:              config.MinImportance,
		ConflictThreshold:          config.ConflictThreshold,
		MaxMemoriesPerDistillation: config.MaxMemoriesPerDistillation,
		MaxSolutionsPerTenant:      config.MaxSolutionsPerTenant,
		EnableCodeFilter:           config.EnableCodeFilter,
		EnableStacktraceFilter:     config.EnableStacktraceFilter,
		EnableLogFilter:            config.EnableLogFilter,
		EnableMarkdownTableFilter:  config.EnableMarkdownTableFilter,
		EnableCrossTurnExtraction:  config.EnableCrossTurnExtraction,
		EnableLengthBonus:          config.EnableLengthBonus,
		LengthThreshold:            config.LengthThreshold,
		LengthBonus:                config.LengthBonus,
		TopNBeforeConflict:         config.TopNBeforeConflict,
		ConflictSearchLimit:        config.ConflictSearchLimit,
		PrecisionOverRecall:        config.PrecisionOverRecall,
	}
}

func convertToAPIDistillationMetrics(metrics *distillation.DistillationMetrics) *DistillationMetrics {
	if metrics == nil {
		return &DistillationMetrics{}
	}

	return &DistillationMetrics{
		AttemptTotal:     metrics.AttemptTotal,
		SuccessTotal:     metrics.SuccessTotal,
		FilteredNoise:    metrics.FilteredNoise,
		FilteredSecurity: metrics.FilteredSecurity,
		ConflictResolved: metrics.ConflictResolved,
		MemoriesCreated:  metrics.MemoriesCreated,
	}
}

func getMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func defaultDistillationConfig() *DistillationConfig {
	return &DistillationConfig{
		MinImportance:              0.6,
		ConflictThreshold:          0.85,
		MaxMemoriesPerDistillation: 3,
		MaxSolutionsPerTenant:      5000,
		EnableCodeFilter:           true,
		EnableStacktraceFilter:     true,
		EnableLogFilter:            true,
		EnableMarkdownTableFilter:  true,
		EnableCrossTurnExtraction:  true,
		EnableLengthBonus:          true,
		LengthThreshold:            60,
		LengthBonus:                0.1,
		TopNBeforeConflict:         true,
		ConflictSearchLimit:        5,
		PrecisionOverRecall:        true,
	}
}
