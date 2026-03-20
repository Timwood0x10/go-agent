// Package models defines data structures for the storage system.
package models

import "time"

// Tool represents an agent tool with semantic embedding for intelligent selection.
// Tools are stored with embedding to enable semantic matching beyond keyword search.
type Tool struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Embedding        []float64              `json:"embedding"`
	EmbeddingModel   string                 `json:"embedding_model"`
	EmbeddingVersion int                    `json:"embedding_version"`
	AgentType        string                 `json:"agent_type"`
	Tags             []string               `json:"tags"`
	UsageCount       int                    `json:"usage_count"`
	SuccessRate      float64                `json:"success_rate"`
	LastUsedAt       time.Time              `json:"last_used_at"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

// TableName returns the table name for this model.
func (t *Tool) TableName() string {
	return "tools"
}

// UpdateUsage updates the usage statistics after tool execution.
func (t *Tool) UpdateUsage(success bool) {
	t.UsageCount++
	t.LastUsedAt = time.Now()

	// Update success rate using moving average
	if t.UsageCount == 1 {
		if success {
			t.SuccessRate = 1.0
		} else {
			t.SuccessRate = 0.0
		}
	} else {
		alpha := 0.1 // smoothing factor
		currentRate := 0.0
		if success {
			currentRate = 1.0
		}
		t.SuccessRate = alpha*currentRate + (1-alpha)*t.SuccessRate
	}
}

// IsAvailable checks if the tool is available for use based on success rate.
func (t *Tool) IsAvailable() bool {
	return t.SuccessRate >= 0.5 // Minimum success rate threshold
}
