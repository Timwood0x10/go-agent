package leader

import (
	"context"
	"sync"

	"styleagent/internal/core/errors"
	"styleagent/internal/core/models"
	"styleagent/internal/protocol/ahp"
)

// taskDispatcher dispatches tasks to sub-agents.
type taskDispatcher struct {
	agentRegistry map[models.AgentType]string
	maxParallel   int
	timeout       int
}

// NewTaskDispatcher creates a new TaskDispatcher.
func NewTaskDispatcher(agentRegistry map[models.AgentType]string, maxParallel int, timeout int) TaskDispatcher {
	if maxParallel <= 0 {
		maxParallel = 10
	}
	if timeout <= 0 {
		timeout = 300
	}
	return &taskDispatcher{
		agentRegistry: agentRegistry,
		maxParallel:   maxParallel,
		timeout:       timeout,
	}
}

// Dispatch dispatches tasks to sub-agents in parallel.
func (d *taskDispatcher) Dispatch(ctx context.Context, tasks []*models.Task) ([]*models.TaskResult, error) {
	if len(tasks) == 0 {
		return nil, errors.ErrInvalidInput
	}

	// Limit parallel execution
	sem := make(chan struct{}, d.maxParallel)
	var wg sync.WaitGroup
	results := make([]*models.TaskResult, len(tasks))
	errCh := make(chan error, 1)

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t *models.Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
				// Simulate task dispatch
				result := d.executeTask(ctx, t)
				results[idx] = result
			}
		}(i, task)
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok && err != nil {
		return nil, err
	}

	return results, nil
}

func (d *taskDispatcher) executeTask(ctx context.Context, task *models.Task) *models.TaskResult {
	result := models.NewTaskResult(task.TaskID, task.AgentType)

	// Get agent address from registry
	agentAddr, ok := d.agentRegistry[task.AgentType]
	if !ok {
		result.SetError("agent not found in registry")
		return result
	}

	// Get session ID from task context
	sessionID := ""
	if task.Context != nil && len(task.Context.Dependencies) > 0 {
		sessionID = task.Context.Dependencies[0]
	}

	// Create AHP message using helper function
	msg := ahp.NewTaskMessage(d.getAgentID(), agentAddr, task.TaskID, sessionID, task.Payload)
	_ = msg // Message would be sent via message queue in real implementation

	// Simulate task dispatch
	result.SetSuccess(nil, "task dispatched to "+agentAddr)

	return result
}

func (d *taskDispatcher) getAgentID() string {
	// Return a default leader ID or make it configurable
	return "leader"
}
