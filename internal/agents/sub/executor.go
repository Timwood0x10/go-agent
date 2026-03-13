package sub

import (
	"context"
	"time"

	"goagent/internal/core/models"
)

// taskExecutor executes recommendation tasks.
type taskExecutor struct {
	toolBinder ToolBinder
}

// NewTaskExecutor creates a new TaskExecutor.
func NewTaskExecutor(toolBinder ToolBinder) TaskExecutor {
	return &taskExecutor{
		toolBinder: toolBinder,
	}
}

// Execute executes a task and returns result.
func (e *taskExecutor) Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error) {
	result := models.NewTaskResult("", models.AgentTypeTop)
	if task == nil {
		result.SetError("nil task received")
		return result, nil
	}

	result = models.NewTaskResult(task.TaskID, task.AgentType)
	startTime := time.Now()

	// Execute based on agent type
	items, reason, err := e.executeByType(ctx, task)
	if err != nil {
		result.SetError(err.Error())
		return result, nil
	}

	result.SetSuccess(items, reason)
	result.Duration = time.Since(startTime)

	return result, nil
}

func (e *taskExecutor) executeByType(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
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

func (e *taskExecutor) executeTopRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:     "top_001",
			Name:       "Cotton T-Shirt",
			Category:   "top",
			Price:      299.00,
			ImageURL:   "https://example.com/images/top_001.jpg",
			Style:      []models.StyleTag{models.StyleCasual},
			MatchReason: "Comfortable cotton material, perfect for daily wear",
		},
		{
			ItemID:     "top_002",
			Name:       "Linen Shirt",
			Category:   "top",
			Price:      459.00,
			ImageURL:   "https://example.com/images/top_002.jpg",
			Style:      []models.StyleTag{models.StyleCasual, models.StyleMinimalist},
			MatchReason: "Breathable linen, minimalist design",
		},
	}
	return items, "top recommendation completed", nil
}

func (e *taskExecutor) executeBottomRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:     "bottom_001",
			Name:       "Slim Fit Jeans",
			Category:   "bottom",
			Price:      399.00,
			ImageURL:   "https://example.com/images/bottom_001.jpg",
			Style:      []models.StyleTag{models.StyleCasual},
			MatchReason: "Classic slim fit, versatile for any occasion",
		},
	}
	return items, "bottom recommendation completed", nil
}

func (e *taskExecutor) executeShoesRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:     "shoes_001",
			Name:       "Casual Sneakers",
			Category:   "shoes",
			Price:      599.00,
			ImageURL:   "https://example.com/images/shoes_001.jpg",
			Style:      []models.StyleTag{models.StyleCasual},
			MatchReason: "Comfortable sole, easy to match",
		},
	}
	return items, "shoes recommendation completed", nil
}

func (e *taskExecutor) executeHeadRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:     "head_001",
			Name:       "Baseball Cap",
			Category:   "head",
			Price:      129.00,
			ImageURL:   "https://example.com/images/head_001.jpg",
			Style:      []models.StyleTag{models.StyleStreet},
			MatchReason: "Street style accessory",
		},
	}
	return items, "head recommendation completed", nil
}

func (e *taskExecutor) executeAccessoryRecommendation(ctx context.Context, task *models.Task) ([]*models.RecommendItem, string, error) {
	items := []*models.RecommendItem{
		{
			ItemID:     "acc_001",
			Name:       "Leather Belt",
			Category:   "accessory",
			Price:      199.00,
			ImageURL:   "https://example.com/images/acc_001.jpg",
			Style:      []models.StyleTag{models.StyleFormal},
			MatchReason: "Genuine leather, classic design",
		},
	}
	return items, "accessory recommendation completed", nil
}
