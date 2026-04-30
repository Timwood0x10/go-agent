package leader

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"goagent/internal/agents/base"
	coreerrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/errors"
	"goagent/internal/memory"
	"goagent/internal/protocol/ahp"

	"golang.org/x/sync/errgroup"
)

// Agent represents the Leader Agent interface.
type Agent interface {
	base.Agent
}

// ProfileParser parses user profile from input.
type ProfileParser interface {
	Parse(ctx context.Context, input string) (*models.UserProfile, error)
}

// TaskPlanner plans tasks based on user profile and input text.
type TaskPlanner interface {
	Plan(ctx context.Context, profile *models.UserProfile, inputText string) ([]*models.Task, error)
	// Replan creates new tasks based on previous result and feedback.
	// This is used for iterative refinement when the initial result is insufficient.
	Replan(ctx context.Context, profile *models.UserProfile, inputText string, previousResult *models.RecommendResult, feedback string) ([]*models.Task, error)
}

// TaskDispatcher dispatches tasks to sub-agents.
type TaskDispatcher interface {
	Dispatch(ctx context.Context, tasks []*models.Task) ([]*models.TaskResult, error)
	RegisterExecutor(agentType models.AgentType, fn func(ctx context.Context, task *models.Task) (*models.TaskResult, error))
}

// ResultAggregator aggregates results from sub-agents.
type ResultAggregator interface {
	Aggregate(ctx context.Context, results []*models.TaskResult, tasks []*models.Task) (*models.RecommendResult, error)
}

// leaderAgent implements the Leader Agent.
type leaderAgent struct {
	mu            sync.RWMutex
	id            string
	agentType     models.AgentType
	status        models.AgentStatus
	config        *LeaderAgentConfig
	parser        ProfileParser
	planner       TaskPlanner
	dispatcher    TaskDispatcher
	aggregator    ResultAggregator
	messageQueue  *ahp.MessageQueue
	heartbeatMon  *ahp.HeartbeatMonitor
	memoryManager memory.MemoryManager
	sessionID     string

	// Lifecycle management
	stopCh      chan struct{}  // Channel to signal shutdown
	distillWg   sync.WaitGroup // WaitGroup for distillation goroutines
	cleanupOnce sync.Once      // Ensure cleanup runs only once
}

// LeaderAgentConfig holds configuration for LeaderAgent.
type LeaderAgentConfig struct {
	base.Config
	MaxParallelTasks int
	MaxSteps         int
	EnableCache      bool
	Loop             LoopConfig
}

// LoopConfig holds configuration for agent loop behavior.
type LoopConfig struct {
	// MaxIterations is the maximum number of loop iterations (default: 3).
	MaxIterations int
	// QualityThreshold is the minimum quality score to accept result (default: 0.7).
	QualityThreshold float64
	// EnableReflection enables reflection and re-planning (default: false).
	EnableReflection bool
	// MaxTotalLLMCalls is the maximum total LLM calls across all iterations (default: 50).
	MaxTotalLLMCalls int
	// MaxLoopDuration is the maximum duration for the entire loop (default: 10 minutes).
	MaxLoopDuration time.Duration
}

// New creates a new LeaderAgent instance.
func New(
	id string,
	parser ProfileParser,
	planner TaskPlanner,
	dispatcher TaskDispatcher,
	aggregator ResultAggregator,
	msgQueue *ahp.MessageQueue,
	hbMon *ahp.HeartbeatMonitor,
	memMgr memory.MemoryManager,
	cfg *LeaderAgentConfig,
) Agent {
	if cfg == nil {
		cfg = DefaultLeaderAgentConfig()
	}
	cfg.ID = id
	cfg.Type = models.AgentTypeLeader

	return &leaderAgent{
		id:            id,
		agentType:     models.AgentTypeLeader,
		status:        models.AgentStatusOffline,
		config:        cfg,
		parser:        parser,
		planner:       planner,
		dispatcher:    dispatcher,
		aggregator:    aggregator,
		messageQueue:  msgQueue,
		heartbeatMon:  hbMon,
		memoryManager: memMgr,
	}
}

// DefaultLeaderAgentConfig returns default configuration.
func DefaultLeaderAgentConfig() *LeaderAgentConfig {
	return &LeaderAgentConfig{
		Config:           *base.DefaultConfig(models.AgentTypeLeader),
		MaxParallelTasks: DefaultMaxParallelTasks,
		MaxSteps:         DefaultMaxSteps,
		EnableCache:      true,
		Loop: LoopConfig{
			MaxIterations:    3,
			QualityThreshold: 0.7,
			EnableReflection: false,
			MaxTotalLLMCalls: 50,
			MaxLoopDuration:  10 * time.Minute,
		},
	}
}

// ID returns the unique identifier.
func (a *leaderAgent) ID() string {
	return a.id
}

// Type returns the agent type.
func (a *leaderAgent) Type() models.AgentType {
	return a.agentType
}

// Status returns the current status.
func (a *leaderAgent) Status() models.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *leaderAgent) setStatus(status models.AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
}

// Start starts the leader agent.
func (a *leaderAgent) Start(ctx context.Context) error {
	if a.Status() != models.AgentStatusOffline {
		return coreerrors.ErrAgentAlreadyStarted
	}

	a.setStatus(models.AgentStatusStarting)

	// Validate and initialize dependencies
	if a.parser == nil {
		return coreerrors.ErrProfileParserNotInitialized
	}
	if a.planner == nil {
		return coreerrors.ErrTaskPlannerNotInitialized
	}
	if a.dispatcher == nil {
		return coreerrors.ErrDispatchNotInitialized
	}
	if a.aggregator == nil {
		return coreerrors.ErrResultAggNotInitialized
	}

	// Initialize lifecycle channels
	a.stopCh = make(chan struct{})

	// Initialize heartbeat monitor if provided
	if a.heartbeatMon != nil {
		// Start heartbeat monitoring for this agent
		// The heartbeat monitor will track agent health and availability
		a.heartbeatMon.RecordHeartbeat(a.id)

		// In a production environment, you would start a background goroutine
		// to periodically send heartbeats and monitor agent health
		slog.Info("Heartbeat monitor initialized", "agent_id", a.id)
	}

	// Initialize message queue if provided
	if a.messageQueue != nil {
		// Message queue is ready to use for inter-agent communication
		// The queue enables the leader agent to:
		// - Send messages to sub-agents
		// - Receive messages from sub-agents
		// - Coordinate distributed task execution

		slog.Info("Message queue initialized", "agent_id", a.id)
	}

	slog.Info("Leader agent started successfully", "agent_id", a.id)
	a.setStatus(models.AgentStatusReady)
	return nil
}

// Stop stops the leader agent and cleans up resources.
func (a *leaderAgent) Stop(ctx context.Context) error {
	if a.Status() == models.AgentStatusOffline {
		return coreerrors.ErrAgentNotRunning
	}

	a.cleanupOnce.Do(func() {
		// Signal all goroutines to stop
		close(a.stopCh)

		// Wait for distillation goroutines to complete
		a.distillWg.Wait()

		// Note: MessageQueue does not have a Drain method. Messages in the queue
		// will be naturally drained by consumers or discarded when the queue is closed.
		// If needed, consumers should handle remaining messages before stopping.

		// Cleanup heartbeat monitor if provided
		if a.heartbeatMon != nil {
			// Remove agent from heartbeat monitoring
			a.heartbeatMon.RemoveAgent(a.id)
		}

		slog.Info("Leader agent stopped successfully", "agent_id", a.id)
	})

	a.setStatus(models.AgentStatusStopping)
	a.setStatus(models.AgentStatusOffline)
	return nil
}

// initMemoryContext initializes session, records user message, builds context with
// similar tasks, and creates a task record. Returns the enriched input, sessionID, and taskID.
func (a *leaderAgent) initMemoryContext(ctx context.Context, strInput string) (enrichedInput string, sessionID string, taskID string) {
	if a.memoryManager == nil {
		return strInput, "", ""
	}

	// Ensure session exists.
	a.mu.Lock()
	sessionID = a.sessionID
	if sessionID == "" {
		newSessionID, err := a.memoryManager.CreateSession(ctx, "default_user")
		if err != nil {
			slog.Warn("Failed to create session", "error", err)
		} else {
			sessionID = newSessionID
			a.sessionID = sessionID
		}
	}
	a.mu.Unlock()

	// Record user message.
	if err := a.memoryManager.AddMessage(ctx, sessionID, "user", strInput); err != nil {
		slog.Warn("memory operation failed, proceeding without", "operation", "AddMessage", "error", err)
	}

	// Build input with conversation context.
	enrichedInput = strInput
	if inputWithContext, err := a.memoryManager.BuildContext(ctx, strInput, sessionID); err != nil {
		slog.Warn("memory operation failed, proceeding without", "operation", "BuildContext", "error", err)
	} else {
		enrichedInput = inputWithContext
	}

	// Search similar tasks for additional context.
	similarTasks, err := a.memoryManager.SearchSimilarTasks(ctx, enrichedInput, 3)
	if err != nil {
		slog.Warn("memory operation failed, proceeding without", "operation", "SearchSimilarTasks", "error", err)
	} else if len(similarTasks) > 0 {
		slog.Debug("Found similar tasks", "count", len(similarTasks))
		contextStr := "\n\nSimilar previous tasks:\n"
		for _, task := range similarTasks {
			if taskInput, ok := task.Payload["input"].(string); ok {
				contextStr += fmt.Sprintf("- %s\n", taskInput)
			}
		}
		enrichedInput += contextStr
	}

	// Create task record for tracking and distillation.
	if tID, err := a.memoryManager.CreateTask(ctx, sessionID, "default_user", enrichedInput); err != nil {
		slog.Warn("Failed to create task - proceeding without task tracking",
			"error", err, "session_id", sessionID,
			"impact", "task will not be tracked for distillation")
	} else {
		taskID = tID
	}

	return enrichedInput, sessionID, taskID
}

// finalizeMemory updates task output, records assistant message, and triggers
// background distillation. Must be called after aggregation succeeds.
func (a *leaderAgent) finalizeMemory(ctx context.Context, sessionID, taskID string, result *models.RecommendResult) {
	if a.memoryManager == nil {
		return
	}

	resultStr := fmt.Sprintf("Generated %d items", len(result.Items))

	// Update task output.
	if taskID != "" {
		if err := a.memoryManager.UpdateTaskOutput(ctx, taskID, resultStr); err != nil {
			slog.Warn("memory operation failed, proceeding without", "operation", "UpdateTaskOutput", "error", err)
		}
	}

	// Record assistant response.
	if err := a.memoryManager.AddMessage(ctx, sessionID, "assistant", resultStr); err != nil {
		slog.Warn("memory operation failed, proceeding without", "operation", "AddMessage", "error", err)
	}

	// Run distillation in background goroutine with proper lifecycle management.
	// Context is created inside the goroutine to avoid race: defer cancel() in the
	// parent function would cancel the context before the goroutine starts.
	if taskID == "" {
		return
	}
	a.distillWg.Add(1)
	go func() { // #nosec G118 -- Background context needed for async distillation after client disconnects
		defer a.distillWg.Done()

		// Check if agent is stopped before starting.
		select {
		case <-a.stopCh:
			slog.Debug("Distillation skipped: agent stopping", "task_id", taskID)
			return
		default:
		}

		// Create a detached context with its own timeout so distillation
		// continues even if the parent request is cancelled.
		// Must use context.Background() — using ctx as parent would cause
		// distillation to abort when the client disconnects.
		distillCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		g, gCtx := errgroup.WithContext(distillCtx)
		g.Go(func() error {
			distilled, err := a.memoryManager.DistillTask(gCtx, taskID)
			if err != nil {
				slog.Warn("Failed to distill task", "error", err, "task_id", taskID)
				return err
			}
			return a.memoryManager.StoreDistilledTask(gCtx, taskID, distilled)
		})

		if err := g.Wait(); err != nil {
			slog.Error("Error in async distillation", "error", err, "task_id", taskID)
		}
	}()
}

// Process handles user input and orchestrates the recommendation workflow with automatic memory management.
func (a *leaderAgent) Process(ctx context.Context, input any) (any, error) {
	if a.Status() != models.AgentStatusReady && a.Status() != models.AgentStatusOffline {
		return nil, coreerrors.ErrAgentNotReady
	}

	if a.Status() == models.AgentStatusOffline {
		if err := a.Start(ctx); err != nil {
			return nil, err
		}
	}

	stepCount := 0
	maxSteps := a.config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = DefaultMaxSteps
	}

	a.setStatus(models.AgentStatusBusy)
	defer a.setStatus(models.AgentStatusReady)

	var strInput string
	switch v := input.(type) {
	case string:
		strInput = v
	case []byte:
		strInput = string(v)
	case fmt.Stringer:
		strInput = v.String()
	default:
		return nil, errors.Wrapf(coreerrors.ErrInvalidInput, "expected string, []byte, or fmt.Stringer, got %T", input)
	}

	// Initialize memory context (session, messages, similar tasks, task record).
	strInput, sessionID, taskID := a.initMemoryContext(ctx, strInput)

	// Step 1: Parse profile
	stepCount++
	if stepCount > maxSteps {
		return nil, coreerrors.ErrMaxStepsExceeded
	}

	profile, err := a.parser.Parse(ctx, strInput)
	if err != nil {
		return nil, err
	}

	// Step 2: Plan tasks
	stepCount++
	if stepCount > maxSteps {
		return nil, coreerrors.ErrMaxStepsExceeded
	}

	tasks, err := a.planner.Plan(ctx, profile, strInput)
	if err != nil {
		return nil, err
	}
	slog.Info("Leader tasks created", "module", "leader", "count", len(tasks))

	// Step 3: Dispatch tasks
	stepCount++
	if stepCount > maxSteps {
		return nil, coreerrors.ErrMaxStepsExceeded
	}

	slog.Info("Leader dispatching tasks", "module", "leader")
	results, err := a.dispatcher.Dispatch(ctx, tasks)
	if err != nil {
		return nil, err
	}
	slog.Info("Leader dispatch completed", "module", "leader", "result_count", len(results))
	for i, r := range results {
		slog.Info("Leader task result", "module", "leader", "index", i, "success", r.Success, "items", len(r.Items), "error", r.Error)
	}

	// Step 4: Aggregate results
	stepCount++
	if stepCount > maxSteps {
		return nil, coreerrors.ErrMaxStepsExceeded
	}

	result, err := a.aggregator.Aggregate(ctx, results, tasks)
	if err != nil {
		return nil, err
	}

	// Finalize memory (update task, record assistant message, distill).
	a.finalizeMemory(ctx, sessionID, taskID, result)

	return result, nil
}

// SendMessage sends a message to another agent.
func (a *leaderAgent) SendMessage(ctx context.Context, msg *ahp.AHPMessage) error {
	if a.messageQueue == nil {
		return coreerrors.ErrQueueNotInitialized
	}
	return a.messageQueue.Enqueue(ctx, msg)
}

// ReceiveMessage receives a message from the message queue.
func (a *leaderAgent) ReceiveMessage(ctx context.Context) (*ahp.AHPMessage, error) {
	if a.messageQueue == nil {
		return nil, coreerrors.ErrQueueNotInitialized
	}
	return a.messageQueue.Dequeue(ctx)
}

// Heartbeat sends a heartbeat signal.
func (a *leaderAgent) Heartbeat(ctx context.Context) error {
	if a.heartbeatMon == nil {
		return nil
	}
	a.heartbeatMon.RecordHeartbeat(a.id)
	return nil
}

// IsAlive checks if the agent is alive.
func (a *leaderAgent) IsAlive() bool {
	return a.Status() == models.AgentStatusReady || a.Status() == models.AgentStatusBusy
}

// ProcessStream handles user input and returns a stream of events.
// It follows the same workflow as Process but emits events at each phase.
func (a *leaderAgent) ProcessStream(ctx context.Context, input any) (<-chan base.AgentEvent, error) {
	if a.Status() != models.AgentStatusReady && a.Status() != models.AgentStatusOffline {
		return nil, coreerrors.ErrAgentNotReady
	}

	if a.Status() == models.AgentStatusOffline {
		if err := a.Start(ctx); err != nil {
			return nil, err
		}
	}

	var strInput string
	switch v := input.(type) {
	case string:
		strInput = v
	case []byte:
		strInput = string(v)
	case fmt.Stringer:
		strInput = v.String()
	default:
		return nil, errors.Wrapf(coreerrors.ErrInvalidInput, "expected string, []byte, or fmt.Stringer, got %T", input)
	}

	// Initialize memory context (session, messages, similar tasks, task record).
	strInput, sessionID, taskID := a.initMemoryContext(ctx, strInput)

	ch := make(chan base.AgentEvent, 64)

	go func() {
		defer close(ch)

		a.setStatus(models.AgentStatusBusy)
		defer a.setStatus(models.AgentStatusReady)

		// Send planning event
		select {
		case ch <- base.AgentEvent{Type: base.EventPlanning, Source: a.id, Data: strInput}:
		case <-ctx.Done():
			return
		}

		// Parse profile
		profile, err := a.parser.Parse(ctx, strInput)
		if err != nil {
			select {
			case ch <- base.AgentEvent{Type: base.EventComplete, Source: a.id, Err: err}:
			case <-ctx.Done():
			}
			return
		}

		// Plan tasks
		tasks, err := a.planner.Plan(ctx, profile, strInput)
		if err != nil {
			select {
			case ch <- base.AgentEvent{Type: base.EventComplete, Source: a.id, Err: err}:
			case <-ctx.Done():
			}
			return
		}
		slog.Info("Leader tasks created", "module", "leader", "count", len(tasks))

		// Dispatch tasks and emit events
		var allResults []*models.TaskResult
		for i, task := range tasks {
			select {
			case ch <- base.AgentEvent{Type: base.EventTaskStart, Source: a.id, Data: task}:
			case <-ctx.Done():
				return
			}

			// Execute task
			result, err := a.dispatcher.Dispatch(ctx, []*models.Task{task})
			if err != nil {
				select {
				case ch <- base.AgentEvent{Type: base.EventTaskComplete, Source: a.id, Data: &models.TaskResult{TaskID: task.TaskID, Success: false, Error: err.Error()}}:
				case <-ctx.Done():
					return
				}
				continue
			}

			if len(result) > 0 {
				select {
				case ch <- base.AgentEvent{Type: base.EventTaskComplete, Source: a.id, Data: result[0]}:
				case <-ctx.Done():
					return
				}
				allResults = append(allResults, result...)
			}
			slog.Debug("Task completed", "index", i, "task_id", task.TaskID)
		}

		// Aggregate results
		select {
		case ch <- base.AgentEvent{Type: base.EventAggregating, Source: a.id}:
		case <-ctx.Done():
			return
		}

		result, err := a.aggregator.Aggregate(ctx, allResults, tasks)
		if err != nil {
			select {
			case ch <- base.AgentEvent{Type: base.EventComplete, Source: a.id, Err: err}:
			case <-ctx.Done():
			}
			return
		}

		// Finalize memory (update task, record assistant message, distill).
		a.finalizeMemory(ctx, sessionID, taskID, result)

		// Send final result
		select {
		case ch <- base.AgentEvent{Type: base.EventComplete, Source: a.id, Data: result}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}
