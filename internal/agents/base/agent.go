package base

import (
	"context"
	"time"

	"goagent/internal/core/models"
	"goagent/internal/protocol/ahp"
)

// EventType represents the type of agent event.
type EventType int

const (
	// EventPlanning indicates the agent is planning.
	EventPlanning EventType = iota
	// EventTaskStart indicates a task has started.
	EventTaskStart
	// EventTaskProgress indicates progress on a task.
	EventTaskProgress
	// EventTaskComplete indicates a task has completed.
	EventTaskComplete
	// EventAggregating indicates the agent is aggregating results.
	EventAggregating
	// EventComplete indicates the agent has completed processing.
	EventComplete
)

// AgentEvent represents an event emitted during agent processing.
type AgentEvent struct {
	// Type is the type of event.
	Type EventType
	// Source is the agent ID that emitted this event.
	Source string
	// Data is the event payload. Type depends on the event type.
	Data any
	// Err contains any error that occurred. Non-nil only for error events.
	Err error
}

// Agent represents the base interface for all agents.
type Agent interface {
	// ID returns the unique identifier of the agent.
	ID() string
	// Type returns the type of the agent.
	Type() models.AgentType
	// Status returns the current status of the agent.
	Status() models.AgentStatus
	// Start starts the agent.
	Start(ctx context.Context) error
	// Stop stops the agent.
	Stop(ctx context.Context) error
	// Process handles input and returns result.
	Process(ctx context.Context, input any) (any, error)
	// ProcessStream handles input and returns a stream of events.
	// The returned channel is closed when processing completes.
	ProcessStream(ctx context.Context, input any) (<-chan AgentEvent, error)
}

// Messenger defines message passing capabilities.
type Messenger interface {
	// SendMessage sends a message to another agent.
	SendMessage(ctx context.Context, msg *ahp.AHPMessage) error
	// ReceiveMessage receives a message from the message queue.
	ReceiveMessage(ctx context.Context) (*ahp.AHPMessage, error)
}

// Heartbeater defines heartbeat capabilities.
type Heartbeater interface {
	// Heartbeat sends a heartbeat signal.
	Heartbeat(ctx context.Context) error
	// IsAlive checks if the agent is alive.
	IsAlive() bool
}

// Config holds common agent configuration.
type Config struct {
	ID                string
	Type              models.AgentType
	HeartbeatInterval time.Duration
	MaxRetries        int
	Timeout           time.Duration
}

// DefaultConfig returns default agent configuration.
func DefaultConfig(agentType models.AgentType) *Config {
	return &Config{
		Type:              agentType,
		HeartbeatInterval: 30 * time.Second,
		MaxRetries:        3,
		Timeout:           5 * time.Minute,
	}
}
