package sub

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	apperrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/llm/output"
)

// taskExecutor executes recommendation tasks.
type taskExecutor struct {
	toolBinder  ToolBinder
	llmAdapter  output.LLMAdapter
	template    *output.TemplateEngine
	promptTpl   string
	validator   *output.Validator
	maxRetries  int
	retryOnFail bool // Retry LLM call when validation fails
	strictMode  bool // Return error on validation failure
	logger      *slog.Logger
}

// ValidationConfig holds validation configuration for executor.
type ValidationConfig struct {
	Enabled     bool
	SchemaType  string
	RetryOnFail bool
	MaxRetries  int
	StrictMode  bool
}

// NewTaskExecutor creates a new TaskExecutor with LLM support.
func NewTaskExecutor(
	toolBinder ToolBinder,
	llmAdapter output.LLMAdapter,
	template *output.TemplateEngine,
	promptTpl string,
	validator *output.Validator,
	maxRetries int,
) TaskExecutor {
	return NewTaskExecutorWithValidation(toolBinder, llmAdapter, template, promptTpl, validator, maxRetries, false, false)
}

// NewTaskExecutorWithValidation creates a new TaskExecutor with validation config.
func NewTaskExecutorWithValidation(
	toolBinder ToolBinder,
	llmAdapter output.LLMAdapter,
	template *output.TemplateEngine,
	promptTpl string,
	validator *output.Validator,
	maxRetries int,
	retryOnFail bool,
	strictMode bool,
) TaskExecutor {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &taskExecutor{
		toolBinder:  toolBinder,
		llmAdapter:  llmAdapter,
		template:    template,
		promptTpl:   promptTpl,
		validator:   validator,
		maxRetries:  maxRetries,
		retryOnFail: retryOnFail,
		strictMode:  strictMode,
		logger:      slog.Default(),
	}
}

// Execute executes a task and returns result.
func (e *taskExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	result := models.NewTaskResult("", models.AgentTypeTop)
	if task == nil {
		result.SetError(apperrors.ErrInvalidInput.Error())
		return result, nil
	}

	result = models.NewTaskResult(task.TaskID, task.AgentType)
	startTime := time.Now()

	// If no LLM adapter, use fallback execution
	if e.llmAdapter == nil {
		items, reason, err := e.executeByType(ctx, task)
		if err != nil {
			result.SetError(err.Error())
			return result, nil
		}
		result.SetSuccess(items, reason)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Get profile from task (either from UserProfile field or Payload)
	var profile *models.UserProfile
	if task.UserProfile != nil {
		profile = task.UserProfile
	} else if task.Payload != nil {
		if p, ok := task.Payload["profile"].(*models.UserProfile); ok {
			profile = p
		}
	}

	if profile == nil {
		// Fallback to type-specific execution
		items, reason, err := e.executeByType(ctx, task)
		if err != nil {
			result.SetError(err.Error())
			return result, nil
		}
		result.SetSuccess(items, reason)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Execute LLM-based recommendation
	items, err := e.executeWithLLM(ctx, task, profile)
	if err != nil {
		slog.Debug("LLM execution failed, using fallback", "error", err)
		// Fallback to type-specific execution
		fallbackItems, reason, fallbackErr := e.executeByType(ctx, task)
		if fallbackErr != nil {
			slog.Debug("Fallback also failed", "error", fallbackErr)
			result.SetError(err.Error())
			return result, nil
		}
		slog.Debug("Using fallback", "item_count", len(fallbackItems))
		result.SetSuccess(fallbackItems, reason)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.SetSuccess(items, "LLM recommendation completed")
	result.Duration = time.Since(startTime)
	return result, nil
}

func (e *taskExecutor) executeWithLLM(ctx context.Context, task *models.Task, profile *models.UserProfile) ([]*models.RecommendItem, error) {
	// Retry loop
	var lastErr error
	for attempt := 0; attempt < e.maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("Retry attempt", "attempt", attempt+1, "max_retries", e.maxRetries)
		}

		// Execute LLM call
		items, err := e.executeWithLLMSingle(ctx, task, profile)
		if err != nil {
			lastErr = err
			slog.Error("LLM call failed", "attempt", attempt+1, "error", err)
			continue
		}

		// Validate results using validator
		if e.validator != nil {
			if err := e.validator.ValidateRecommendResult(&models.RecommendResult{Items: items}); err != nil {
				slog.Debug("Validation failed", "error", err)
				// Retry if enabled and not already at max retries
				if e.retryOnFail && attempt < e.maxRetries-1 {
					slog.Debug("Will retry LLM call", "next_attempt", attempt+2, "max_retries", e.maxRetries)
					continue
				}
				// Strict mode: return error
				if e.strictMode {
					return nil, fmt.Errorf("validation failed: %w", err)
				}
				// Non-strict mode: log and continue with whatever we got
				slog.Debug("Continuing with unvalidated result", "strict_mode", false)
			} else {
				slog.Debug("Validation passed")
			}
		}

		slog.Info("Got items from LLM", "count", len(items))
		return items, nil
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

func (e *taskExecutor) executeWithLLMSingle(ctx context.Context, task *models.Task, profile *models.UserProfile) ([]*models.RecommendItem, error) {
	// Render prompt - support both fashion and travel profiles
	// Use lowercase keys to match template's {{index . "key"}} syntax
	promptData := map[string]any{
		"Category": string(task.AgentType), // Uppercase to match template
	}

	// Check if this is a travel request - use Preferences map
	if profile.Preferences != nil && len(profile.Preferences) > 0 {
		// Copy all preferences to promptData (lowercase keys)
		for k, v := range profile.Preferences {
			promptData[k] = v
		}
	}

	// Always include budget from profile.Budget (for fashion/travel compatibility)
	promptData["budget"] = formatBudget(profile.Budget)

	// Also set style from profile
	if len(profile.Style) > 0 {
		promptData["style"] = profile.Style
	}

	prompt, err := e.template.Render(e.promptTpl, promptData)
	if err != nil {
		return nil, fmt.Errorf("render prompt: %w", err)
	}
	slog.Debug("Generated prompt", "preview", prompt[:min(200, len(prompt))])

	// Call LLM
	response, err := e.llmAdapter.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	slog.Debug("LLM response", "preview", response[:min(500, len(response))])

	// Parse response
	parser := output.NewParser()
	result, err := parser.ParseRecommendResult(response)
	if err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	if result == nil || result.Items == nil {
		return nil, fmt.Errorf("empty result from LLM")
	}

	slog.Info("Parsed result items", "count", len(result.Items))
	return result.Items, nil
}

func formatBudget(budget *models.PriceRange) string {
	if budget == nil {
		return "0 - 10000"
	}
	return fmt.Sprintf("%.0f - %.0f", budget.Min, budget.Max)
}

// executeByType dispatches to type-specific handlers.
func (e *taskExecutor) executeByType(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	slog.Debug("executeByType called", "agent_type", task.AgentType)
	switch task.AgentType {
	case models.AgentTypeTop:
		return e.executeTopRecommendation(ctx, task)
	case models.AgentTypeBottom:
		return e.executeBottomRecommendation(ctx, task)
	case models.AgentTypeShoes:
		return e.executeShoesRecommendation(ctx, task)
	case models.AgentTypeHead:
		return e.executeHeadRecommendation(ctx, task)
	case models.AgentTypeAccessory:
		return e.executeAccessoryRecommendation(ctx, task)
	default:
		return nil, "unknown agent type", nil
	}
}

// Legacy methods - kept for backward compatibility.
func (e *taskExecutor) executeTopRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:           "top_001",
			Name:             "Cotton T-Shirt",
			Category:         "top",
			Price:            299.00,
			ImageURL:         "https://example.com/images/top_001.jpg",
			AgentPreferences: []models.StyleTag{models.StyleCasual},
			MatchReason:      "Comfortable cotton material, perfect for daily wear",
		},
	}
	return items, "top recommendation completed", nil
}

func (e *taskExecutor) executeBottomRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:           "bottom_001",
			Name:             "Slim Fit Jeans",
			Category:         "bottom",
			Price:            399.00,
			ImageURL:         "https://example.com/images/bottom_001.jpg",
			AgentPreferences: []models.StyleTag{models.StyleCasual},
			MatchReason:      "Classic slim fit, versatile for any occasion",
		},
	}
	return items, "bottom recommendation completed", nil
}

func (e *taskExecutor) executeShoesRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:           "shoes_001",
			Name:             "Casual Sneakers",
			Category:         "shoes",
			Price:            599.00,
			ImageURL:         "https://example.com/images/shoes_001.jpg",
			AgentPreferences: []models.StyleTag{models.StyleCasual},
			MatchReason:      "Comfortable sole, easy to match",
		},
	}
	return items, "shoes recommendation completed", nil
}

func (e *taskExecutor) executeHeadRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:           "head_001",
			Name:             "Baseball Cap",
			Category:         "head",
			Price:            129.00,
			ImageURL:         "https://example.com/images/head_001.jpg",
			AgentPreferences: []models.StyleTag{models.StyleStreet},
			MatchReason:      "Street style accessory",
		},
	}
	return items, "head recommendation completed", nil
}

func (e *taskExecutor) executeAccessoryRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:           "acc_001",
			Name:             "Leather Belt",
			Category:         "accessory",
			Price:            199.00,
			ImageURL:         "https://example.com/images/acc_001.jpg",
			AgentPreferences: []models.StyleTag{models.StyleFormal},
			MatchReason:      "Genuine leather, classic design",
		},
	}
	return items, "accessory recommendation completed", nil
}
