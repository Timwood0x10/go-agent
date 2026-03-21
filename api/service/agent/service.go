// Package agent provides agent service implementation.
package agent

import (
	"context"
	"fmt"
	"time"

	"goagent/api/core"
	"goagent/internal/memory"
)

// Service provides agent management operations.
type Service struct {
	repo      core.AgentRepository
	memoryMgr memory.MemoryManager
	config    *core.BaseConfig
}

// Config represents service configuration.
type Config struct {
	// BaseConfig is the base configuration.
	BaseConfig *core.BaseConfig
	// MemoryMgr is the memory manager instance.
	MemoryMgr memory.MemoryManager
	// Repo is the agent repository.
	Repo core.AgentRepository
}

// NewService creates a new agent service instance.
// Args:
// config - service configuration.
// Returns new agent service instance or error.
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	if config.BaseConfig == nil {
		config.BaseConfig = &core.BaseConfig{
			RequestTimeout: 30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
		}
	}

	return &Service{
		repo:      config.Repo,
		memoryMgr: config.MemoryMgr,
		config:    config.BaseConfig,
	}, nil
}

// CreateAgent creates a new agent with the given configuration.
// Args:
// ctx - operation context.
// agentConfig - the agent configuration.
// Returns the created agent or error.
func (s *Service) CreateAgent(ctx context.Context, agentConfig *core.AgentConfig) (*core.Agent, error) {
	if agentConfig == nil {
		return nil, ErrInvalidConfig
	}

	if agentConfig.ID == "" {
		return nil, ErrInvalidAgentID
	}

	// Check if agent already exists (only if repo is configured)
	if s.repo != nil {
		existing, err := s.repo.Get(ctx, agentConfig.ID)
		if err == nil && existing != nil {
			return nil, ErrAgentAlreadyExists
		}
	}

	// Create session for the agent
	var sessionID string
	var err error
	if s.memoryMgr != nil {
		sessionID, err = s.memoryMgr.CreateSession(ctx, agentConfig.ID)
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
	}

	now := time.Now().Unix()
	agent := &core.Agent{
		ID:        agentConfig.ID,
		Name:      agentConfig.Name,
		Type:      agentConfig.Type,
		Status:    core.AgentStatusReady,
		SessionID: sessionID,
		Config:    agentConfig.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Persist agent only if repo is configured
	if s.repo != nil {
		if err := s.repo.Create(ctx, agent); err != nil {
			return nil, fmt.Errorf("create agent: %w", err)
		}
	}

	return agent, nil
}

// GetAgent retrieves an agent by ID.
// Args:
// ctx - operation context.
// agentID - the agent identifier.
// Returns the agent or error if not found.
func (s *Service) GetAgent(ctx context.Context, agentID string) (*core.Agent, error) {
	if agentID == "" {
		return nil, ErrInvalidAgentID
	}

	// If repo is not configured, return not found
	if s.repo == nil {
		return nil, ErrAgentNotFound
	}

	agent, err := s.repo.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	if agent == nil {
		return nil, ErrAgentNotFound
	}

	return agent, nil
}

// UpdateAgent updates an existing agent.
// Args:
// ctx - operation context.
// agentID - the agent identifier.
// updates - the fields to update.
// Returns the updated agent or error.
func (s *Service) UpdateAgent(ctx context.Context, agentID string, updates map[string]interface{}) (*core.Agent, error) {
	if agentID == "" {
		return nil, ErrInvalidAgentID
	}

	agent, err := s.repo.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		agent.Name = name
	}
	if status, ok := updates["status"].(core.AgentStatus); ok {
		agent.Status = status
	}
	// TODO: Apply other updates

	agent.UpdatedAt = time.Now().Unix()

	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}

	return agent, nil
}

// DeleteAgent deletes an agent and its associated data.
// Args:
// ctx - operation context.
// agentID - the agent identifier.
// Returns error if deletion fails.
func (s *Service) DeleteAgent(ctx context.Context, agentID string) error {
	if agentID == "" {
		return ErrInvalidAgentID
	}

	// Get agent to retrieve session ID
	agent, err := s.repo.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	if agent == nil {
		return ErrAgentNotFound
	}

	// Delete session if exists
	if agent.SessionID != "" && s.memoryMgr != nil {
		// TODO: Implement session deletion
	}

	// Delete agent
	if err := s.repo.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}

	return nil
}

// ListAgents lists agents with optional filtering.

// Args:

// ctx - operation context.

// filter - optional filter criteria.

// Returns list of agents and pagination info, or error.

func (s *Service) ListAgents(ctx context.Context, filter *core.AgentFilter) ([]*core.Agent, *core.PaginationResponse, error) {

	if filter == nil {

		filter = &core.AgentFilter{}

	}



	// If repo is not configured, return empty list

	if s.repo == nil {

		pagination := &core.PaginationResponse{

			Total:      0,

			Page:       1,

			PageSize:   0,

			TotalPages: 1,

			HasMore:    false,

		}

		return []*core.Agent{}, pagination, nil

	}



	agents, err := s.repo.List(ctx, filter)

	if err != nil {

		return nil, nil, fmt.Errorf("list agents: %w", err)

	}



	// TODO: Calculate pagination info

	pagination := &core.PaginationResponse{

		Total:      int64(len(agents)),

		Page:       1,

		PageSize:   len(agents),

		TotalPages: 1,

		HasMore:    false,

	}



	return agents, pagination, nil

}

// ExecuteTask executes a task on an agent.
// Args:
// ctx - operation context.
// task - the task to execute.
// Returns the task result or error.
func (s *Service) ExecuteTask(ctx context.Context, task *core.Task) (*core.TaskResult, error) {
	if task == nil {
		return nil, ErrInvalidConfig
	}

	if task.AgentID == "" {
		return nil, ErrInvalidAgentID
	}

	// Get agent
	agent, err := s.repo.Get(ctx, task.AgentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// Update agent status
	agent.Status = core.AgentStatusRunning
	agent.UpdatedAt = time.Now().Unix()
	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("update agent status: %w", err)
	}

	// TODO: Implement task execution logic
	now := time.Now().Unix()
	result := &core.TaskResult{
		TaskID:      task.ID,
		AgentID:     task.AgentID,
		Success:     true,
		Data:        make(map[string]interface{}),
		CompletedAt: now,
	}

	// Update agent status back to ready
	agent.Status = core.AgentStatusReady
	agent.UpdatedAt = time.Now().Unix()
	if err := s.repo.Update(ctx, agent); err != nil {
		// Log error but don't fail the task
		fmt.Printf("warning: failed to update agent status: %v\n", err)
	}

	return result, nil
}

// GetTaskResult retrieves the result of a task.
// Args:
// ctx - operation context.
// taskID - the task identifier.
// Returns the task result or error if not found.
func (s *Service) GetTaskResult(ctx context.Context, taskID string) (*core.TaskResult, error) {
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	// TODO: Implement task result retrieval
	return &core.TaskResult{
		TaskID:  taskID,
		Success: true,
		Data:    make(map[string]interface{}),
	}, nil
}