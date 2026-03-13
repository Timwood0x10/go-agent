package leader

import (
	"context"
	"time"

	"styleagent/internal/core/errors"
	"styleagent/internal/core/models"
)

// taskPlanner creates tasks based on user profile.
type taskPlanner struct {
	maxTasks int
}

// NewTaskPlanner creates a new TaskPlanner.
func NewTaskPlanner(maxTasks int) TaskPlanner {
	if maxTasks <= 0 {
		maxTasks = 5
	}
	return &taskPlanner{
		maxTasks: maxTasks,
	}
}

// Plan creates tasks based on user profile.
func (p *taskPlanner) Plan(ctx context.Context, profile *models.UserProfile) ([]*models.Task, error) {
	if profile == nil {
		return nil, errors.ErrNilPointer
	}

	tasks := make([]*models.Task, 0)

	// Generate tasks based on style tags
	for _, style := range profile.Style {
		task := models.NewTask(generateTaskID(), getAgentTypeForStyle(style), profile)
		task.Priority = calculatePriority(style)
		task.Deadline = time.Now().Add(1 * time.Hour)
		tasks = append(tasks, task)
	}

	// Add occasion-based tasks
	for _, occasion := range profile.Occasions {
		task := models.NewTask(generateTaskID(), getAgentTypeForOccasion(occasion), profile)
		task.Priority = calculatePriorityForOccasion(occasion)
		task.Deadline = time.Now().Add(1 * time.Hour)
		tasks = append(tasks, task)
	}

	// Limit total tasks
	if len(tasks) > p.maxTasks {
		tasks = tasks[:p.maxTasks]
	}

	return tasks, nil
}

func generateTaskID() string {
	return "task_" + time.Now().Format("20060102150405")
}

func getAgentTypeForStyle(style models.StyleTag) models.AgentType {
	switch style {
	case models.StyleCasual, models.StyleMinimalist:
		return models.AgentTypeTop
	case models.StyleFormal:
		return models.AgentTypeBottom
	case models.StyleStreet, models.StyleVintage:
		return models.AgentTypeBottom
	default:
		return models.AgentTypeTop
	}
}

func getAgentTypeForOccasion(occasion models.Occasion) models.AgentType {
	switch occasion {
	case models.OccasionWork, models.OccasionFormal:
		return models.AgentTypeBottom
	case models.OccasionParty, models.OccasionDate:
		return models.AgentTypeAccessory
	case models.OccasionSports:
		return models.AgentTypeShoes
	default:
		return models.AgentTypeTop
	}
}

func calculatePriority(style models.StyleTag) int {
	switch style {
	case models.StyleFormal:
		return 10
	case models.StyleCasual:
		return 5
	default:
		return 3
	}
}

func calculatePriorityForOccasion(occasion models.Occasion) int {
	switch occasion {
	case models.OccasionWork, models.OccasionFormal:
		return 10
	case models.OccasionParty, models.OccasionDate:
		return 8
	default:
		return 5
	}
}
