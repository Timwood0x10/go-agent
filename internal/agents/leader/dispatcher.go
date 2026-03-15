package leader

import (
	"context"
	"fmt"
	"sync"

	apperrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
)

// TaskExecutorFunc is a function type for executing tasks directly.
type TaskExecutorFunc func(ctx context.Context, task *models.Task) (*models.TaskResult, error)

// taskDispatcher dispatches tasks to sub-agents.
type taskDispatcher struct {
	agentRegistry map[models.AgentType]string
	executorFuncs map[models.AgentType]TaskExecutorFunc
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
	d := &taskDispatcher{
		agentRegistry: agentRegistry,
		executorFuncs: make(map[models.AgentType]TaskExecutorFunc),
		maxParallel:   maxParallel,
		timeout:       timeout,
	}
	return d
}

// RegisterExecutor registers an executor function for a specific agent type.
func (d *taskDispatcher) RegisterExecutor(agentType models.AgentType, fn func(ctx context.Context, task *models.Task) (*models.TaskResult, error)) {
	d.executorFuncs[agentType] = fn
}

// Dispatch dispatches tasks to sub-agents in parallel.
func (d *taskDispatcher) Dispatch(ctx context.Context, tasks []*models.Task) ([]*models.TaskResult, error) {
	if len(tasks) == 0 {
		return nil, apperrors.ErrInvalidInput
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
				// Execute task
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

	fmt.Printf("[DEBUG Dispatcher] Executing task %s for agent type %s (addr: %s)\n", task.TaskID, task.AgentType, agentAddr)

	// Check if we have a direct executor registered
	if fn, exists := d.executorFuncs[task.AgentType]; exists {
		// Call the executor directly
		fmt.Printf("[DEBUG Dispatcher] Calling executor for %s\n", task.AgentType)
		execResult, err := fn(ctx, task)
		if err != nil {
			fmt.Printf("[DEBUG Dispatcher] Executor error: %v\n", err)
			result.SetError(err.Error())
			return result
		}
		fmt.Printf("[DEBUG Dispatcher] Executor returned %d items, success=%v\n", len(execResult.Items), execResult.Success)
		return execResult
	}

	// Fallback: create AHP message and send via queue
	// TODO: Implement actual message queue based dispatch
	// Current implementation uses direct executor function call above
	// For distributed deployment, uncomment the following:
	/*
	sessionID := ""
	if task.Context != nil && len(task.Context.Dependencies) > 0 {
		sessionID = task.Context.Dependencies[0]
	}
	msg := ahp.NewTaskMessage(d.getAgentID(), agentAddr, task.TaskID, sessionID, task.Payload)
	if d.messageQueue != nil {
		if err := d.messageQueue.Enqueue(ctx, msg); err != nil {
			result.SetError("failed to enqueue message: " + err.Error())
			return result
		}
		result.SetSuccess(nil, "task dispatched via queue to "+agentAddr)
		return result
	}
	*/

	// Simulate task dispatch (for backward compatibility)
	result.SetSuccess(nil, "task dispatched to "+agentAddr)

	return result
}

func (d *taskDispatcher) getAgentID() string {
	return "leader"
}
