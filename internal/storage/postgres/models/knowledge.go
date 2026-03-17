// Package models defines data structures for the storage system.
// This package contains domain models for knowledge chunks, experiences, tools, conversations, task results, and secrets.
package models

import (
	"time"
)

// KnowledgeChunk represents a RAG knowledge base entry with vector embedding.
// This is the core data structure for semantic search and retrieval.
type KnowledgeChunk struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	Content          string                 `json:"content"`
	Embedding        []float64              `json:"embedding"`
	EmbeddingModel   string                 `json:"embedding_model"`
	EmbeddingVersion int                    `json:"embedding_version"`
	EmbeddingStatus  string                 `json:"embedding_status"`
	ChunkIndex       int                    `json:"chunk_index"`
	DocumentID       string                 `json:"document_id"`
	SourceType       string                 `json:"source_type"`
	Source           string                 `json:"source"`
	Metadata         map[string]interface{} `json:"metadata"`
	ContentHash      string                 `json:"content_hash"`
	AccessCount      int                    `json:"access_count"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// TableName returns the table name for this model.
// Different dimensions use different tables to avoid mixing vector spaces.
func (k *KnowledgeChunk) TableName() string {
	return "knowledge_chunks_1024"
}

// EmbeddingStatus constants.
const (
	EmbeddingStatusPending    = "pending"
	EmbeddingStatusProcessing = "processing"
	EmbeddingStatusCompleted  = "completed"
	EmbeddingStatusFailed     = "failed"
)
