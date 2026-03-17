package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"goagent/internal/core/errors"
)

// VectorSearcher handles vector similarity search.
type VectorSearcher struct {
	db DBTX
}

// NewVectorSearcher creates a new VectorSearcher.
func NewVectorSearcher(pool *Pool) *VectorSearcher {
	return &VectorSearcher{db: pool.db}
}

// NewVectorSearcherWithDB creates a new VectorSearcher with a transaction or connection.
func NewVectorSearcherWithDB(db DBTX) *VectorSearcher {
	return &VectorSearcher{db: db}
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
	// Validate table name to prevent SQL injection
	if err := sanitizeSQLTable(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	// Validate limit to prevent excessive results
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid limit: %d (must be 1-1000)", limit)
	}

	query := fmt.Sprintf(`
		SELECT id, 1 - (embedding <=> $1) as distance, metadata
		FROM %s
		ORDER BY embedding <=> $1
		LIMIT $2
	`, safeFormatTable(table))

	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding: %w", err)
	}

	rows, err := v.db.QueryContext(ctx, query, embeddingJSON, limit)
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
	// Validate table name to prevent SQL injection
	if err := sanitizeSQLTable(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	// Validate embedding dimensions
	if len(embedding) == 0 {
		return fmt.Errorf("embedding cannot be empty")
	}

	if len(embedding) > 2000 { // Reasonable upper limit
		return fmt.Errorf("embedding dimension too large: %d (max 2000)", len(embedding))
	}

	// Validate id
	if err := validateSQLIdentifier(id); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

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
	`, safeFormatTable(table))

	_, err = v.db.ExecContext(ctx, query, id, embeddingJSON, metadataJSON)
	if err != nil {
		return fmt.Errorf("add embedding: %w", err)
	}

	return nil
}

// DeleteEmbedding deletes a vector embedding.
func (v *VectorSearcher) DeleteEmbedding(ctx context.Context, table, id string) error {
	// Validate table name to prevent SQL injection
	if err := sanitizeSQLTable(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	// Validate id
	if err := validateSQLIdentifier(id); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, safeFormatTable(table))

	_, err := v.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}

	return nil
}

// CreateVectorTable creates a table with vector support.
// This is a simplified implementation - in production use proper pgvector setup.
func (v *VectorSearcher) CreateVectorTable(ctx context.Context, table string, metadataSchema string) error {
	// Validate table name to prevent SQL injection
	if err := sanitizeSQLTable(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	// Validate dimension (should be between 1 and 2000)
	dim := 1536 // Default dimension for common embedding models
	if dim < 1 || dim > 2000 {
		return fmt.Errorf("invalid dimension: %d (must be 1-2000)", dim)
	}

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			embedding VECTOR(%d),
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_cosine_ops);
	`, safeFormatTable(table), dim, safeFormatTable(table), safeFormatTable(table)) // Default dimension for common embedding models

	_, err := v.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("create vector table: %w", err)
	}

	return nil
}
