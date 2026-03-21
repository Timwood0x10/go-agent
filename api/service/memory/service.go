// Package memory provides memory service implementation.
package memory

import (
	"context"
	"fmt"
	"time"

	"goagent/api/core"
	"goagent/internal/memory"
)

// Service provides memory management operations.
type Service struct {
	repo      core.MemoryRepository
	memoryMgr memory.MemoryManager
	config    *core.BaseConfig
}

// Config represents service configuration.
type Config struct {
	// BaseConfig is the base configuration.
	BaseConfig *core.BaseConfig
	// MemoryMgr is the internal memory manager instance.
	MemoryMgr memory.MemoryManager
	// Repo is the memory repository.
	Repo core.MemoryRepository
}

// NewService creates a new memory service instance.
// Args:
// config - service configuration.
// Returns new memory service instance or error.
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

// CreateSession creates a new session with the given configuration.
// Args:
// ctx - operation context.
// sessionConfig - the session configuration.
// Returns the session ID or error.
func (s *Service) CreateSession(ctx context.Context, sessionConfig *core.SessionConfig) (string, error) {
	if sessionConfig == nil {
		return "", ErrInvalidConfig
	}

	if sessionConfig.UserID == "" {
		return "", ErrInvalidUserID
	}

	// Generate session ID
	sessionID := generateSessionID()

	now := time.Now()
	var expiresAt *time.Time
	if sessionConfig.ExpiresIn > 0 {
		expired := now.Add(sessionConfig.ExpiresIn)
		expiresAt = &expired
	}

	session := &core.Session{
		ID:        sessionID,
		UserID:    sessionConfig.UserID,
		TenantID:  sessionConfig.TenantID,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
		Metadata:  sessionConfig.Metadata,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return sessionID, nil
}

// GetSession retrieves a session by ID.
// Args:
// ctx - operation context.
// sessionID - the session identifier.
// Returns the session or error if not found.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*core.Session, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session == nil {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// DeleteSession deletes a session and all its messages.
// Args:
// ctx - operation context.
// sessionID - the session identifier.
// Returns error if deletion fails.
func (s *Service) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	// Verify session exists
	_, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// AddMessage adds a message to a session.
// Args:
// ctx - operation context.
// sessionID - the session identifier.
// role - the message role.
// content - the message content.
// Returns error if addition fails.
func (s *Service) AddMessage(ctx context.Context, sessionID string, role core.MessageRole, content string) error {
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	if role == "" {
		return ErrInvalidRole
	}

	if content == "" {
		return ErrInvalidContent
	}

	// Verify session exists
	_, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	message := &core.Message{
		ID:        generateMessageID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Time:      time.Now(),
		Metadata:  make(core.Metadata),
	}

	if err := s.repo.AddMessage(ctx, message); err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	return nil
}

// GetMessages retrieves messages from a session.
// Args:
// ctx - operation context.
// sessionID - the session identifier.
// pagination - pagination parameters.
// Returns list of messages or error.
func (s *Service) GetMessages(ctx context.Context, sessionID string, pagination *core.PaginationRequest) ([]*core.Message, error) {
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	// Verify session exists
	_, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if pagination == nil {
		pagination = &core.PaginationRequest{
			Page:     1,
			PageSize: 100,
		}
	}

	messages, err := s.repo.GetMessages(ctx, sessionID, pagination)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	return messages, nil
}

// DistillTask distills a task for future reference.
// Args:
// ctx - operation context.
// taskID - the task identifier.
// Returns the distilled task or error.
func (s *Service) DistillTask(ctx context.Context, taskID string) (*core.DistilledTask, error) {
	if taskID == "" {
		return nil, ErrInvalidTaskID
	}

	// TODO: Implement task distillation logic
	task := &core.DistilledTask{
		TaskID:    taskID,
		Input:     "example input",
		Output:    "example output",
		Context:   "example context",
		Summary:   "example summary",
		Tags:      []string{"example"},
		Embedding: make([]float32, 0),
		CreatedAt: time.Now(),
	}

	if err := s.repo.StoreDistilledTask(ctx, task); err != nil {
		return nil, fmt.Errorf("store distilled task: %w", err)
	}

	return task, nil
}

// SearchSimilarTasks searches for similar tasks.
// Args:
// ctx - operation context.
// query - the search query.
// Returns list of search results or error.
func (s *Service) SearchSimilarTasks(ctx context.Context, query *core.SearchQuery) ([]*core.SearchResult, error) {
	if query == nil {
		return nil, ErrInvalidConfig
	}

	if query.Query == "" {
		return nil, ErrInvalidQuery
	}

	if query.Limit <= 0 {
		query.Limit = 10
	}

	results, err := s.repo.SearchSimilarTasks(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search similar tasks: %w", err)
	}

	return results, nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	// TODO: Implement proper ID generation
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	// TODO: Implement proper ID generation
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
