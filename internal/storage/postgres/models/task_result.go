// Package models defines data structures for the storage system.
package models

import "time"

// TaskResult represents the execution result of an agent task.
// This stores task outputs with vector embedding for future reference and learning.
type TaskResult struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	SessionID        string                 `json:"session_id"`
	TaskType         string                 `json:"task_type"`
	AgentID          string                 `json:"agent_id"`
	Input            map[string]interface{} `json:"input"`
	Output           map[string]interface{} `json:"output"`
	Embedding        []float64              `json:"embedding"`
	EmbeddingModel   string                 `json:"embedding_model"`
	EmbeddingVersion int                    `json:"embedding_version"`
	Status           string                 `json:"status"`
	Error            string                 `json:"error"`
	LatencyMs        int                    `json:"latency_ms"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

// TableName returns the table name for this model.
func (t *TaskResult) TableName() string {
	return "task_results_1024"
}

// TaskStatus constants.
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)

// IsSuccessful checks if the task execution was successful.
func (t *TaskResult) IsSuccessful() bool {
	return t.Status == TaskStatusCompleted
}

// IsFailed checks if the task execution failed.
func (t *TaskResult) IsFailed() bool {
	return t.Status == TaskStatusFailed
}
