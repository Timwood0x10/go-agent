package leader

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"goagent/internal/core/models"
)

// taskIDCounter is used to generate unique task IDs.
var taskIDCounter uint64

// getRandomSuffix returns a random suffix for extra uniqueness.
func getRandomSuffix() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(100000000))
	return fmt.Sprintf("%08d", n.Int64())
}

// generateTaskID generates a unique task ID.
func generateTaskID() string {
	id := atomic.AddUint64(&taskIDCounter, 1)
	randSuffix := getRandomSuffix()
	return fmt.Sprintf("task_%s_%d_%s", time.Now().Format("20060102150405"), id, randSuffix)
}

// SubAgentConfig represents sub agent configuration (mirrors config.SubAgentConfig).
type SubAgentConfig struct {
	ID       string
	Type     string
	Triggers []string
}

// taskPlanner creates tasks based on user profile and config.
type taskPlanner struct {
	maxTasks  int
	subAgents []SubAgentConfig
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
		return nil, fmt.Errorf("profile cannot be nil")
	}

	tasks := make([]*models.Task, 0)

	// Create a default task for processing user input
	task := &models.Task{
		TaskID:      generateTaskID(),
		TaskType:    models.AgentTypeTop,
		AgentType:   models.AgentTypeTop,
		UserProfile: profile,
		Payload:     map[string]any{"action": "analyze_profile"},
		Priority:    1,
		CreatedAt:   time.Now(),
	}

	tasks = append(tasks, task)

	// Limit total tasks
	if len(tasks) > p.maxTasks {
		tasks = tasks[:p.maxTasks]
	}

	return tasks, nil
}
