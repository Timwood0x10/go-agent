package leader

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	apperrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/protocol/ahp"
)

// TaskExecutorFunc is a function type for executing tasks directly.
type TaskExecutorFunc func(ctx context.Context, task *models.Task) (*models.TaskResult, error)

// MessageSender sends messages to sub-agents (for distributed deployment).
type MessageSender interface {
	Send(ctx context.Context, agentAddr string, msg *ahp.AHPMessage) error
}

// LocalMessageSender sends messages to local agent queues.
type LocalMessageSender struct {
	queues map[string]*ahp.MessageQueue
	mu     sync.RWMutex
}

// NewLocalMessageSender creates a new LocalMessageSender.
func NewLocalMessageSender() *LocalMessageSender {
	return &LocalMessageSender{
		queues: make(map[string]*ahp.MessageQueue),
	}
}

// RegisterQueue registers a message queue for an agent address.
func (s *LocalMessageSender) RegisterQueue(agentAddr string, queue *ahp.MessageQueue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[agentAddr] = queue
}

// Send sends a message to the specified agent address.
func (s *LocalMessageSender) Send(ctx context.Context, agentAddr string, msg *ahp.AHPMessage) error {
	s.mu.RLock()
	queue, ok := s.queues[agentAddr]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no queue registered for agent: %s", agentAddr)
	}

	return queue.Enqueue(ctx, msg)
}

// taskDispatcher dispatches tasks to sub-agents.
type taskDispatcher struct {
	agentRegistry map[models.AgentType]string
	executorFuncs map[models.AgentType]TaskExecutorFunc
	messageSender MessageSender
	maxParallel   int
	timeout       int
}

// NewTaskDispatcher creates a new TaskDispatcher.
func NewTaskDispatcher(agentRegistry map[models.AgentType]string, maxParallel int, timeout int, sender MessageSender) TaskDispatcher {
	if maxParallel <= 0 {
		maxParallel = 10
	}
	if timeout <= 0 {
		timeout = 300
	}
	d := &taskDispatcher{
		agentRegistry: agentRegistry,
		executorFuncs: make(map[models.AgentType]TaskExecutorFunc),
		messageSender: sender,
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
				select {
				case errCh <- ctx.Err():
				default:
				}
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

	slog.Debug("Executing task", "task_id", task.TaskID, "agent_type", task.AgentType, "agent_addr", agentAddr)

	// Check if we have a direct executor registered
	if fn, exists := d.executorFuncs[task.AgentType]; exists {
		// Call the executor directly
		slog.Debug("Calling executor", "agent_type", task.AgentType)
		execResult, err := fn(ctx, task)
		if err != nil {
			slog.Error("Executor error", "agent_type", task.AgentType, "error", err)
			result.SetError(err.Error())
			return result
		}
		slog.Debug("Executor returned", "agent_type", task.AgentType, "item_count", len(execResult.Items), "success", execResult.Success)
		return execResult
	}

	// If no local executor, use message sender (for distributed deployment)
	if d.messageSender != nil {
		sessionID := ""
		if task.Context != nil && len(task.Context.Dependencies) > 0 {
			sessionID = task.Context.Dependencies[0]
		}
		msg := ahp.NewTaskMessage(d.getAgentID(), agentAddr, task.TaskID, sessionID, task.Payload)
		if err := d.messageSender.Send(ctx, agentAddr, msg); err != nil {
			result.SetError("failed to send message: " + err.Error())
			return result
		}
		result.SetSuccess(nil, "task dispatched via message queue to "+agentAddr)
		return result
	}

	// No executor and no message sender - return error
	result.SetError("no executor or message sender registered for agent type: " + string(task.AgentType))
	return result
}

func (d *taskDispatcher) getAgentID() string {
	return "leader"
}
