package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"goagent/internal/core/errors"
)

// VectorSearcher handles vector similarity search.
type VectorSearcher struct {
	pool *Pool
}

// NewVectorSearcher creates a new VectorSearcher.
func NewVectorSearcher(pool *Pool) *VectorSearcher {
	return &VectorSearcher{pool: pool}
}

// SearchResult represents a vector search result.
type SearchResult struct {
	ID       string
	Score    float64
	Metadata map[string]any
}

// Search performs a vector similarity search.
// This is a simplified implementation that uses pgvector if available.
func (v *VectorSearcher) Search(ctx context.Context, table string, embedding []float64, limit int) ([]*SearchResult, error) {
	// Simplified implementation - in production, use pgvector
	// SELECT id, 1 - (embedding <=> $1) as distance, metadata FROM table
	// ORDER BY embedding <=> $1 LIMIT $2

	query := fmt.Sprintf(`
		SELECT id, 1 - (embedding <=> $1) as distance, metadata
		FROM %s
		ORDER BY embedding <=> $1
		LIMIT $2
	`, table)

	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding: %w", err)
	}

	rows, err := v.pool.Query(ctx, query, embeddingJSON, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var result SearchResult
		var metadataJSON []byte

		if err := rows.Scan(&result.ID, &result.Score, &metadataJSON); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &result.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		results = append(results, &result)
	}

	if len(results) == 0 {
		return nil, errors.ErrRecordNotFound
	}

	return results, nil
}

// AddEmbedding adds a vector embedding to the specified table.
func (v *VectorSearcher) AddEmbedding(ctx context.Context, table, id string, embedding []float64, metadata map[string]any) error {
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, embedding, metadata)
		VALUES ($1, $2, $3)
	`, table)

	_, err = v.pool.Exec(ctx, query, id, embeddingJSON, metadataJSON)
	if err != nil {
		return fmt.Errorf("add embedding: %w", err)
	}

	return nil
}

// DeleteEmbedding deletes a vector embedding.
func (v *VectorSearcher) DeleteEmbedding(ctx context.Context, table, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table)

	_, err := v.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}

	return nil
}

// CreateVectorTable creates a table with vector support.
// This is a simplified implementation - in production use proper pgvector setup.
func (v *VectorSearcher) CreateVectorTable(ctx context.Context, table string, metadataSchema string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			embedding VECTOR(%d),
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_cosine_ops);
	`, table, 1536, table, table) // Default dimension for common embedding models

	_, err := v.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("create vector table: %w", err)
	}

	return nil
}
