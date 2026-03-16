// Package memory provides unified memory management for the StyleAgent framework.
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	memctx "goagent/internal/memory/context"
	"goagent/internal/core/models"
)

// memoryManager implements MemoryManager interface.
// It coordinates session memory, task memory, and local vector storage.
type memoryManager struct {
	sessionMemory     *memctx.SessionMemory
	taskMemory        *memctx.TaskMemory
	distilledTasks    map[string]*DistilledTaskData
	mu                sync.RWMutex
	config            *MemoryConfig
	started           bool
	stopped           bool
	vectorDim         int
}

// DistilledTaskData holds distilled task information with local vector.
type DistilledTaskData struct {
	TaskID    string
	Input     string
	Output    string
	Context   map[string]interface{}
	Vector    []float64
	CreatedAt time.Time
}

// NewMemoryManager creates a new MemoryManager with the given configuration.
func NewMemoryManager(config *MemoryConfig) (MemoryManager, error) {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	sessionMemory := memctx.NewSessionMemory(
		config.MaxSessions,
		config.SessionTTL,
	)

	taskMemory := memctx.NewTaskMemory(
		config.MaxTasks,
		config.TaskTTL,
	)

	return &memoryManager{
		sessionMemory:  sessionMemory,
		taskMemory:     taskMemory,
		distilledTasks: make(map[string]*DistilledTaskData),
		config:         config,
		vectorDim:      config.VectorDim,
	}, nil
}

// Start starts the memory manager and background workers.
func (m *memoryManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	m.sessionMemory.StartCleanup()
	m.started = true

	slog.Info("Memory manager started")
	return nil
}

// Stop stops the memory manager and cleans up resources.
func (m *memoryManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return nil
	}

	if err := m.sessionMemory.Close(ctx); err != nil {
		slog.Warn("Failed to close session memory", "error", err)
	}

	m.stopped = true
	slog.Info("Memory manager stopped")
	return nil
}

// CreateSession creates a new session and returns the session ID.
func (m *memoryManager) CreateSession(ctx context.Context, userID string) (string, error) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	messages := []memctx.Message{
		{
			Role:    "system",
			Content: "New session started",
			Time:    time.Now(),
		},
	}

	if err := m.sessionMemory.Set(ctx, sessionID, userID, messages); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	slog.Debug("Session created", "session_id", sessionID, "user_id", userID)
	return sessionID, nil
}

// AddMessage adds a message to the session.
func (m *memoryManager) AddMessage(ctx context.Context, sessionID, role, content string) error {
	msg := memctx.Message{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	}

	if err := m.sessionMemory.AddMessage(ctx, sessionID, msg); err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	slog.Debug("Message added", "session_id", sessionID, "role", role)
	return nil
}

// GetMessages retrieves all messages from the session.
func (m *memoryManager) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	sessionMemMessages, err := m.sessionMemory.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	messages := make([]Message, len(sessionMemMessages))
	for i, msg := range sessionMemMessages {
		messages[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
			Time:    msg.Time,
		}
	}

	return messages, nil
}

// BuildContext builds input with conversation history context.
func (m *memoryManager) BuildContext(ctx context.Context, input string, sessionID string) (string, error) {
	messages, err := m.GetMessages(ctx, sessionID)
	if err != nil {
		slog.Warn("Failed to get messages, using raw input", "error", err)
		return input, nil
	}

	// Keep only last N messages to avoid long context.
	maxHistory := m.config.MaxHistory
	if len(messages) > maxHistory {
		messages = messages[len(messages)-maxHistory:]
	}

	// Build context string.
	var contextBuilder string
	if len(messages) > 0 {
		contextBuilder = "Previous conversation history:\n\n"
		for _, msg := range messages {
			if msg.Role == "user" {
				contextBuilder += fmt.Sprintf("User: %s\n", truncate(msg.Content, 100))
			} else if msg.Role == "assistant" {
				contextBuilder += fmt.Sprintf("Assistant: %s\n", truncate(msg.Content, 100))
			}
		}
		contextBuilder += "\nCurrent request:\n"
	}
	contextBuilder += input

	slog.Debug("Context built", "session_id", sessionID, "history_length", len(messages))
	return contextBuilder, nil
}

// CreateTask creates a new task and returns the task ID.
func (m *memoryManager) CreateTask(ctx context.Context, sessionID, userID, input string) (string, error) {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	if err := m.taskMemory.Set(ctx, taskID, sessionID, userID, input); err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	slog.Debug("Task created", "task_id", taskID, "session_id", sessionID)
	return taskID, nil
}

// UpdateTaskOutput updates the task output.
func (m *memoryManager) UpdateTaskOutput(ctx context.Context, taskID, output string) error {
	if err := m.taskMemory.UpdateOutput(ctx, taskID, output); err != nil {
		return fmt.Errorf("update task output: %w", err)
	}

	slog.Debug("Task output updated", "task_id", taskID)
	return nil
}

// DistillTask extracts key information from task for future reference.
func (m *memoryManager) DistillTask(ctx context.Context, taskID string) (*models.Task, error) {
	task, err := m.taskMemory.Distill(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("distill task: %w", err)
	}

	slog.Debug("Task distilled", "task_id", taskID)
	return task, nil
}

// StoreDistilledTask stores a distilled task with local vector embedding.
func (m *memoryManager) StoreDistilledTask(ctx context.Context, taskID string, distilled *models.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract input string from payload
	inputStr, ok := distilled.Payload["input"].(string)
	if !ok {
		inputStr = ""
	}

	// Generate local vector using simple hash-based approach
	vector := m.generateHashVector(inputStr)

	data := &DistilledTaskData{
		TaskID:    taskID,
		Input:     inputStr,
		Output:    fmt.Sprintf("%v", distilled.Payload["output"]),
		Context:   distilled.Payload["context"].(map[string]interface{}),
		Vector:    vector,
		CreatedAt: time.Now(),
	}

	m.distilledTasks[taskID] = data
	slog.Debug("Distilled task stored", "task_id", taskID)

	return nil
}

// generateHashVector generates a simple hash-based vector from text.
func (m *memoryManager) generateHashVector(text string) []float64 {
	vector := make([]float64, m.vectorDim)

	if len(text) == 0 {
		return vector
	}

	// Simple hash-based vector generation
	hash := uint64(0)
	for i, c := range text {
		hash = hash*31 + uint64(c)
		if i >= len(text)-1 {
			break
		}
	}

	// Spread hash across vector dimensions
	for i := range vector {
		vector[i] = float64((hash >> (i * 5)) % 1000) / 1000.0
	}

	// Normalize vector
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

// SearchSimilarTasks searches for similar tasks using local cosine similarity.
func (m *memoryManager) SearchSimilarTasks(ctx context.Context, query string, limit int) ([]*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Generate vector for query
	queryVector := m.generateHashVector(query)

	// Calculate cosine similarity for all tasks
	type similarityResult struct {
		task  *models.Task
		score float64
	}

	results := []similarityResult{}
	for _, data := range m.distilledTasks {
		score := m.cosineSimilarity(queryVector, data.Vector)
		if score > 0.5 { // Only return tasks with similarity > 0.5
			results = append(results, similarityResult{
				task: &models.Task{
					TaskID: data.TaskID,
					Payload: map[string]any{
						"input":   data.Input,
						"output":  data.Output,
						"context": data.Context,
					},
				},
				score: score,
			})
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	tasks := make([]*models.Task, 0, len(results))
	for _, result := range results {
		tasks = append(tasks, result.task)
	}

	slog.Debug("Similar tasks found", "count", len(tasks))
	return tasks, nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func (m *memoryManager) cosineSimilarity(v1, v2 []float64) float64 {
	if len(v1) != len(v2) {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := range v1 {
		dotProduct += v1[i] * v2[i]
		norm1 += v1[i] * v1[i]
		norm2 += v2[i] * v2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// truncate truncates a string to the maximum length and adds "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
