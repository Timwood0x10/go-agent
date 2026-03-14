package sub

import (
	"context"
	"fmt"
	"time"

	apperrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/llm/output"
)

// taskExecutor executes recommendation tasks.
type taskExecutor struct {
	toolBinder ToolBinder
	llmAdapter output.LLMAdapter
	template   *output.TemplateEngine
	promptTpl  string
	validator  *output.Validator
	maxRetries int
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
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &taskExecutor{
		toolBinder: toolBinder,
		llmAdapter: llmAdapter,
		template:   template,
		promptTpl:  promptTpl,
		validator:  validator,
		maxRetries: maxRetries,
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
	} else if p, ok := task.Payload["profile"].(*models.UserProfile); ok {
		profile = p
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
		fmt.Printf("[DEBUG] LLM execution failed: %v\n", err)
		// Fallback to type-specific execution
		fallbackItems, reason, fallbackErr := e.executeByType(ctx, task)
		if fallbackErr != nil {
			fmt.Printf("[DEBUG] Fallback also failed: %v\n", fallbackErr)
			result.SetError(err.Error())
			return result, nil
		}
		fmt.Printf("[DEBUG] Using fallback, got %d items\n", len(fallbackItems))
		result.SetSuccess(fallbackItems, reason)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.SetSuccess(items, "LLM recommendation completed")
	result.Duration = time.Since(startTime)
	return result, nil
}

func (e *taskExecutor) executeWithLLM(ctx context.Context, task *models.Task, profile *models.UserProfile) ([]*models.RecommendItem, error) {
	// Render prompt
	promptData := map[string]any{
		"style":    profile.Style,
		"occasion": profile.Occasions,
		"budget":   formatBudget(profile.Budget),
		"category": string(task.AgentType),
	}
	prompt, err := e.template.Render(e.promptTpl, promptData)
	if err != nil {
		return nil, fmt.Errorf("render prompt: %w", err)
	}
	fmt.Printf("[DEBUG] Prompt: %s\n", prompt[:min(200, len(prompt))])

	// Call LLM
	response, err := e.llmAdapter.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	fmt.Printf("[DEBUG] LLM Response: %s\n", response[:min(200, len(response))])

	// Parse response
	parser := output.NewParser()
	result, err := parser.ParseRecommendResult(response)
	if err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	if result == nil || result.Items == nil {
		return nil, fmt.Errorf("empty result from LLM")
	}

	fmt.Printf("[DEBUG] Got %d items\n", len(result.Items))
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
	fmt.Printf("[DEBUG] executeByType called for agent type: %s\n", task.AgentType)
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
			ItemID:      "top_001",
			Name:        "Cotton T-Shirt",
			Category:    "top",
			Price:       299.00,
			ImageURL:    "https://example.com/images/top_001.jpg",
			Style:       []models.StyleTag{models.StyleCasual},
			MatchReason: "Comfortable cotton material, perfect for daily wear",
		},
	}
	return items, "top recommendation completed", nil
}

func (e *taskExecutor) executeBottomRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:      "bottom_001",
			Name:        "Slim Fit Jeans",
			Category:    "bottom",
			Price:       399.00,
			ImageURL:    "https://example.com/images/bottom_001.jpg",
			Style:       []models.StyleTag{models.StyleCasual},
			MatchReason: "Classic slim fit, versatile for any occasion",
		},
	}
	return items, "bottom recommendation completed", nil
}

func (e *taskExecutor) executeShoesRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:      "shoes_001",
			Name:        "Casual Sneakers",
			Category:    "shoes",
			Price:       599.00,
			ImageURL:    "https://example.com/images/shoes_001.jpg",
			Style:       []models.StyleTag{models.StyleCasual},
			MatchReason: "Comfortable sole, easy to match",
		},
	}
	return items, "shoes recommendation completed", nil
}

func (e *taskExecutor) executeHeadRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:      "head_001",
			Name:        "Baseball Cap",
			Category:    "head",
			Price:       129.00,
			ImageURL:    "https://example.com/images/head_001.jpg",
			Style:       []models.StyleTag{models.StyleStreet},
			MatchReason: "Street style accessory",
		},
	}
	return items, "head recommendation completed", nil
}

func (e *taskExecutor) executeAccessoryRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:      "acc_001",
			Name:        "Leather Belt",
			Category:    "accessory",
			Price:       199.00,
			ImageURL:    "https://example.com/images/acc_001.jpg",
			Style:       []models.StyleTag{models.StyleFormal},
			MatchReason: "Genuine leather, classic design",
		},
	}
	return items, "accessory recommendation completed", nil
}
