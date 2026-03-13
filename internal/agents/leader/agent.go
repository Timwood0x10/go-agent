package leader

import (
	"context"
	"sync"

	"goagent/internal/agents/base"
	"goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/protocol/ahp"
)

// Agent represents the Leader Agent interface.
type Agent interface {
	base.Agent
}

// ProfileParser parses user profile from input.
type ProfileParser interface {
	Parse(ctx context.Context, input string) (*models.UserProfile, error)
}

// TaskPlanner plans tasks based on user profile.
type TaskPlanner interface {
	Plan(ctx context.Context, profile *models.UserProfile) ([]*models.Task, error)
}

// TaskDispatcher dispatches tasks to sub-agents.
type TaskDispatcher interface {
	Dispatch(ctx context.Context, tasks []*models.Task) ([]*models.TaskResult, error)
}

// ResultAggregator aggregates results from sub-agents.
type ResultAggregator interface {
	Aggregate(ctx context.Context, results []*models.TaskResult) (*models.RecommendResult, error)
}

// leaderAgent implements the Leader Agent.
type leaderAgent struct {
	mu           sync.RWMutex
	id           string
	agentType    models.AgentType
	status       models.AgentStatus
	config       *LeaderAgentConfig
	parser       ProfileParser
	planner      TaskPlanner
	dispatcher   TaskDispatcher
	aggregator   ResultAggregator
	messageQueue *ahp.MessageQueue
	heartbeatMon *ahp.HeartbeatMonitor
}

// LeaderAgentConfig holds configuration for LeaderAgent.
type LeaderAgentConfig struct {
	base.Config
	MaxParallelTasks int
	EnableCache      bool
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
	cfg *LeaderAgentConfig,
) Agent {
	if cfg == nil {
		cfg = DefaultLeaderAgentConfig()
	}
	cfg.ID = id
	cfg.Type = models.AgentTypeLeader

	return &leaderAgent{
		id:           id,
		agentType:    models.AgentTypeLeader,
		status:       models.AgentStatusOffline,
		config:       cfg,
		parser:       parser,
		planner:      planner,
		dispatcher:   dispatcher,
		aggregator:   aggregator,
		messageQueue: msgQueue,
		heartbeatMon: hbMon,
	}
}

// DefaultLeaderAgentConfig returns default configuration.
func DefaultLeaderAgentConfig() *LeaderAgentConfig {
	return &LeaderAgentConfig{
		Config:           *base.DefaultConfig(models.AgentTypeLeader),
		MaxParallelTasks: 10,
		EnableCache:      true,
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
		return errors.ErrAgentAlreadyStarted
	}

	a.setStatus(models.AgentStatusStarting)

	a.setStatus(models.AgentStatusReady)
	return nil
}

// Stop stops the leader agent.
func (a *leaderAgent) Stop(ctx context.Context) error {
	if a.Status() == models.AgentStatusOffline {
		return errors.ErrAgentNotRunning
	}

	a.setStatus(models.AgentStatusStopping)
	a.setStatus(models.AgentStatusOffline)
	return nil
}

// Process handles user input and orchestrates the recommendation workflow.
func (a *leaderAgent) Process(ctx context.Context, input any) (any, error) {
	if a.Status() != models.AgentStatusReady && a.Status() != models.AgentStatusOffline {
		return nil, errors.ErrAgentNotReady
	}

	if a.Status() == models.AgentStatusOffline {
		if err := a.Start(ctx); err != nil {
			return nil, err
		}
	}

	a.setStatus(models.AgentStatusBusy)
	defer a.setStatus(models.AgentStatusReady)

	strInput, ok := input.(string)
	if !ok {
		return nil, errors.ErrInvalidInput
	}

	profile, err := a.parser.Parse(ctx, strInput)
	if err != nil {
		return nil, err
	}

	tasks, err := a.planner.Plan(ctx, profile)
	if err != nil {
		return nil, err
	}

	results, err := a.dispatcher.Dispatch(ctx, tasks)
	if err != nil {
		return nil, err
	}

	result, err := a.aggregator.Aggregate(ctx, results)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendMessage sends a message to another agent.
func (a *leaderAgent) SendMessage(ctx context.Context, msg *ahp.AHPMessage) error {
	if a.messageQueue == nil {
		return errors.ErrQueueNotInitialized
	}
	return a.messageQueue.Enqueue(ctx, msg)
}

// ReceiveMessage receives a message from the message queue.
func (a *leaderAgent) ReceiveMessage(ctx context.Context) (*ahp.AHPMessage, error) {
	if a.messageQueue == nil {
		return nil, errors.ErrQueueNotInitialized
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
