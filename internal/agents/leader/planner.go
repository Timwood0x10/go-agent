package leader

import (
	"context"
	"time"

	"goagent/internal/core/errors"
	"goagent/internal/core/models"
)

// taskPlanner creates tasks based on user profile and config.
type taskPlanner struct {
	maxTasks  int
	subAgents []SubAgentConfig // Configuration from YAML
}

// SubAgentConfig represents sub agent configuration (mirrors config.SubAgentConfig).
type SubAgentConfig struct {
	ID       string
	Type     string
	Triggers []string
}

// NewTaskPlanner creates a new TaskPlanner.
func NewTaskPlanner(maxTasks int) TaskPlanner {
	if maxTasks <= 0 {
		maxTasks = 5
	}
	return &taskPlanner{
		maxTasks:  maxTasks,
		subAgents: nil,
	}
}

// NewTaskPlannerWithConfig creates a TaskPlanner with sub-agent configuration.
func NewTaskPlannerWithConfig(maxTasks int, subAgents []SubAgentConfig) TaskPlanner {
	if maxTasks <= 0 {
		maxTasks = 5
	}
	return &taskPlanner{
		maxTasks:  maxTasks,
		subAgents: subAgents,
	}
}

// Plan creates tasks based on user profile.
func (p *taskPlanner) Plan(ctx context.Context, profile *models.UserProfile) ([]*models.Task, error) {
	if profile == nil {
		return nil, errors.ErrNilPointer
	}

	tasks := make([]*models.Task, 0)

	// If we have sub-agent config with triggers, use config-driven approach
	if len(p.subAgents) > 0 {
		tasks = p.createTasksFromConfig(profile)
	} else {
		// Fallback to fashion/existing logic
		tasks = p.createFashionTasks(profile)
	}

	// Limit total tasks
	if len(tasks) > p.maxTasks {
		tasks = tasks[:p.maxTasks]
	}

	return tasks, nil
}

// createTasksFromConfig creates tasks based on sub-agent triggers in config.
func (p *taskPlanner) createTasksFromConfig(profile *models.UserProfile) []*models.Task {
	tasks := make([]*models.Task, 0)
	addedTypes := make(map[models.AgentType]bool)

	// Get all profile fields (from Preferences or direct fields)
	profileFields := p.getProfileFields(profile)

	// Check each sub-agent's triggers
	for _, agent := range p.subAgents {
		if len(agent.Triggers) == 0 {
			continue
		}

		// Check if any trigger matches profile fields
		matched := false
		for _, trigger := range agent.Triggers {
			if _, exists := profileFields[trigger]; exists {
				matched = true
				break
			}
		}

		if matched {
			agentType := models.AgentType(agent.Type)
			// Avoid duplicate tasks for same agent type
			if !addedTypes[agentType] {
				task := models.NewTask(generateTaskID(), agentType, profile)
				task.Deadline = time.Now().Add(1 * time.Hour)
				tasks = append(tasks, task)
				addedTypes[agentType] = true
			}
		}
	}

	// If no tasks matched (e.g., empty profile), add all agents as fallback
	if len(tasks) == 0 {
		for _, agent := range p.subAgents {
			agentType := models.AgentType(agent.Type)
			if !addedTypes[agentType] {
				task := models.NewTask(generateTaskID(), agentType, profile)
				task.Deadline = time.Now().Add(1 * time.Hour)
				tasks = append(tasks, task)
				addedTypes[agentType] = true
			}
		}
	}

	return tasks
}

// getProfileFields extracts all field names from profile for matching.
func (p *taskPlanner) getProfileFields(profile *models.UserProfile) map[string]bool {
	fields := make(map[string]bool)

	// Add style tags
	for _, style := range profile.Style {
		fields[string(style)] = true
	}

	// Add occasions
	for _, occasion := range profile.Occasions {
		fields[string(occasion)] = true
	}

	// Add preferences (travel-specific fields)
	if profile.Preferences != nil {
		for key := range profile.Preferences {
			fields[key] = true
			// Also add string values for partial matching
			if val, ok := profile.Preferences[key].(string); ok {
				fields[val] = true
			}
			// Add values from string arrays
			if vals, ok := profile.Preferences[key].([]interface{}); ok {
				for _, v := range vals {
					if s, ok := v.(string); ok {
						fields[s] = true
					}
				}
			}
		}
	}

	return fields
}

// createFashionTasks creates tasks for fashion recommendation (fallback).
func (p *taskPlanner) createFashionTasks(profile *models.UserProfile) []*models.Task {
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

	return tasks
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
