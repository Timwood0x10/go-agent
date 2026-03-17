// Package models defines data structures for the storage system.
package models

import "time"

// Experience represents an agent experience with learning capability.
// This includes successful solutions, failed cases, and patterns for future reference.
type Experience struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	Type             string                 `json:"type"`
	Input            string                 `json:"input"`
	Output           string                 `json:"output"`
	Embedding        []float64              `json:"embedding"`
	EmbeddingModel   string                 `json:"embedding_model"`
	EmbeddingVersion int                    `json:"embedding_version"`
	Score            float64                `json:"score"`
	Success          bool                   `json:"success"`
	AgentID          string                 `json:"agent_id"`
	Metadata         map[string]interface{} `json:"metadata"`
	DecayAt          time.Time              `json:"decay_at"`
	CreatedAt        time.Time              `json:"created_at"`
}

// TableName returns the table name for this model.
func (e *Experience) TableName() string {
	return "experiences_1024"
}

// ExperienceType constants.
const (
	ExperienceTypeQuery     = "query"
	ExperienceTypeSolution  = "solution"
	ExperienceTypeFailure   = "failure"
	ExperienceTypePattern   = "pattern"
	ExperienceTypeDistilled = "distilled"
)

// IsExpired checks if the experience has decayed and should be excluded from search.
func (e *Experience) IsExpired() bool {
	return !e.DecayAt.IsZero() && time.Now().After(e.DecayAt)
}
