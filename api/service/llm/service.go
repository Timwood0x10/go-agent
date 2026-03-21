// Package llm provides LLM service implementation.
package llm

import (
	"context"
	"fmt"
	"time"

	"goagent/api/core"
	"goagent/internal/llm"
)

// Service provides LLM operations.
type Service struct {
	client    *llm.Client
	repo      core.LLMRepository
	config    *core.BaseConfig
	llmConfig *core.LLMConfig
}

// Config represents service configuration.
type Config struct {
	// BaseConfig is the base configuration.
	BaseConfig *core.BaseConfig
	// LLMConfig is the LLM configuration.
	LLMConfig *core.LLMConfig
	// Repo is the LLM repository (optional, for logging/audit).
	Repo core.LLMRepository
}

// NewService creates a new LLM service instance.
// Args:
// config - service configuration.
// Returns new LLM service instance or error.
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	if config.LLMConfig == nil {
		return nil, ErrInvalidLLMConfig
	}

	if config.BaseConfig == nil {
		config.BaseConfig = &core.BaseConfig{
			RequestTimeout: 30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
		}
	}

	// Create internal LLM client
	internalConfig := &llm.Config{
		Provider: string(config.LLMConfig.Provider),
		APIKey:   config.LLMConfig.APIKey,
		BaseURL:  config.LLMConfig.BaseURL,
		Model:    config.LLMConfig.Model,
		Timeout:  config.LLMConfig.Timeout,
	}

	client, err := llm.NewClient(internalConfig)
	if err != nil {
		return nil, fmt.Errorf("create LLM client: %w", err)
	}

	return &Service{
		client:    client,
		repo:      config.Repo,
		config:    config.BaseConfig,
		llmConfig: config.LLMConfig,
	}, nil
}

// Generate generates text from the given messages.
// Args:
// ctx - operation context.
// request - the generation request.
// Returns the generation response or error.
func (s *Service) Generate(ctx context.Context, request *core.GenerateRequest) (*core.GenerateResponse, error) {
	if request == nil {
		return nil, ErrInvalidConfig
	}

	if len(request.Messages) == 0 {
		return nil, ErrInvalidMessages
	}

	// Build prompt from messages
	prompt := s.buildPrompt(request.Messages)

	// Generate text
	content, err := s.client.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate text: %w", err)
	}

	response := &core.GenerateResponse{
		Content:      content,
		FinishReason: "stop",
		Usage: core.TokenUsage{
			PromptTokens:     0, // TODO: Calculate actual tokens
			CompletionTokens: 0,
			TotalTokens:      0,
		},
		Model: s.getModel(),
	}

	// Log generation if repository is available
	if s.repo != nil {
		if err := s.repo.LogGeneration(ctx, request, response); err != nil {
			// Log error but don't fail the request
			fmt.Printf("warning: failed to log generation: %v\n", err)
		}
	}

	return response, nil
}

// GenerateSimple generates text from a simple prompt.
// Args:
// ctx - operation context.
// prompt - the prompt text.
// Returns the generated text or error.
func (s *Service) GenerateSimple(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", ErrInvalidPrompt
	}

	content, err := s.client.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generate text: %w", err)
	}

	return content, nil
}

// GenerateEmbedding generates an embedding for the given text.
// Args:
// ctx - operation context.
// request - the embedding request.
// Returns the embedding response or error.
func (s *Service) GenerateEmbedding(ctx context.Context, request *core.EmbeddingRequest) (*core.EmbeddingResponse, error) {
	if request == nil {
		return nil, ErrInvalidConfig
	}

	if request.Input == "" {
		return nil, ErrInvalidInput
	}

	// TODO: Implement embedding generation
	// This requires calling the embedding service
	embedding := make([]float32, 0) // Placeholder

	response := &core.EmbeddingResponse{
		Embedding: embedding,
		Model:     s.getModel(),
		Usage: core.TokenUsage{
			PromptTokens: 0,
			TotalTokens:  0,
		},
	}

	return response, nil
}

// GetConfig returns the current LLM configuration.
// Returns the LLM configuration.
func (s *Service) GetConfig() *core.LLMConfig {
	return s.llmConfig
}

// IsEnabled checks if the LLM service is properly configured and available.
// Returns true if enabled, false otherwise.
func (s *Service) IsEnabled() bool {
	return s.client.IsEnabled()
}

// GetProvider returns the current LLM provider.
// Returns the provider type.
func (s *Service) GetProvider() core.LLMProvider {
	if s.llmConfig != nil {
		return s.llmConfig.Provider
	}
	return ""
}

// GetModel returns the current model name.
// Returns the model name.
func (s *Service) GetModel() string {
	if s.llmConfig != nil {
		return s.llmConfig.Model
	}
	return ""
}

// buildPrompt builds a prompt from messages.
func (s *Service) buildPrompt(messages []*core.LLMMessage) string {
	prompt := ""
	for _, msg := range messages {
		prompt += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
	}
	return prompt
}

// getModel returns the model name to use.
func (s *Service) getModel() string {
	if s.llmConfig != nil && s.llmConfig.Model != "" {
		return s.llmConfig.Model
	}
	return "default"
}
