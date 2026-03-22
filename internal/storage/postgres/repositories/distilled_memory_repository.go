// Package repositories provides data access for distilled memories.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goagent/internal/storage/postgres"
)

// DistilledMemory represents a distilled memory from conversation history.
type DistilledMemory struct {
	ID              string
	TenantID        string
	UserID          string
	SessionID       string
	Content         string
	Embedding       []float64
	EmbeddingModel  string
	EmbeddingVersion int
	MemoryType      string
	Importance      float64
	Metadata        map[string]interface{}
	AccessCount     int
	LastAccessedAt  *time.Time
	ExpiresAt       time.Time
	CreatedAt       time.Time
}

// DistilledMemoryRepository provides data access for distilled memories.
type DistilledMemoryRepository struct {
	db     postgres.DBTX
	dbPool *sql.DB
}

// Ensure DistilledMemoryRepository implements DistilledMemoryRepositoryInterface.
var _ DistilledMemoryRepositoryInterface = (*DistilledMemoryRepository)(nil)

// NewDistilledMemoryRepository creates a new DistilledMemoryRepository.
func NewDistilledMemoryRepository(db postgres.DBTX, dbPool *sql.DB) *DistilledMemoryRepository {
	return &DistilledMemoryRepository{
		db:     db,
		dbPool: dbPool,
	}
}

// parseVectorString converts pgvector string format to []float64.
// Note: This function is also defined in knowledge_repository.go to avoid import cycles.
func parseDistilledVectorString(vecStr string) ([]float64, error) {
	if len(vecStr) == 0 {
		return []float64{}, nil
	}

	vecStr = strings.Trim(vecStr, "[]")
	if vecStr == "" {
		return []float64{}, nil
	}

	parts := strings.Split(vecStr, ",")
	result := make([]float64, len(parts))
	for i, part := range parts {
		val, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &result[i])
		if err != nil || val != 1 {
			return nil, fmt.Errorf("failed to parse vector component: %w", err)
		}
	}

	return result, nil
}

// Create creates a new distilled memory.
func (r *DistilledMemoryRepository) Create(ctx context.Context, memory *DistilledMemory) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO distilled_memories
		(id, tenant_id, user_id, session_id, content, embedding, embedding_model,
		 embedding_version, memory_type, importance, metadata, access_count,
		 last_accessed_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT DO NOTHING
	`

	_, err = r.db.ExecContext(ctx, query,
		memory.ID, memory.TenantID, memory.UserID, memory.SessionID, memory.Content,
		postgres.FormatVector(memory.Embedding), memory.EmbeddingModel, memory.EmbeddingVersion,
		memory.MemoryType, memory.Importance, metadataJSON, memory.AccessCount,
		memory.LastAccessedAt, memory.ExpiresAt, memory.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("create distilled memory: %w", err)
	}

	return nil
}

// SearchByVector searches for memories using vector similarity.
func (r *DistilledMemoryRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*DistilledMemory, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, content, embedding::text,
			   embedding_model, embedding_version, memory_type, importance,
			   metadata, access_count, last_accessed_at, expires_at, created_at,
			   1 - (embedding <=> $1::vector) as similarity
		FROM distilled_memories
		WHERE tenant_id = $2
		  AND expires_at > NOW()
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	vectorStr := postgres.FormatVector(embedding)
	rows, err := r.db.QueryContext(ctx, query, vectorStr, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("search distilled memories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	memories := make([]*DistilledMemory, 0)
	for rows.Next() {
		memory := &DistilledMemory{}
		var similarity float64
		var embeddingStr string

		err := rows.Scan(
			&memory.ID, &memory.TenantID, &memory.UserID, &memory.SessionID,
			&memory.Content, &embeddingStr, &memory.EmbeddingModel,
			&memory.EmbeddingVersion, &memory.MemoryType, &memory.Importance,
			&memory.Metadata, &memory.AccessCount, &memory.LastAccessedAt,
			&memory.ExpiresAt, &memory.CreatedAt, &similarity,
		)
		if err != nil {
			continue
		}

		memory.Embedding, err = parseDistilledVectorString(embeddingStr)
		if err != nil {
			continue
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// GetByUserID retrieves memories for a specific user.
func (r *DistilledMemoryRepository) GetByUserID(ctx context.Context, tenantID, userID string, limit int) ([]*DistilledMemory, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, content, embedding::text,
			   embedding_model, embedding_version, memory_type, importance,
			   metadata, access_count, last_accessed_at, expires_at, created_at
		FROM distilled_memories
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND expires_at > NOW()
		ORDER BY importance DESC, created_at DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get memories by user: %w", err)
	}
	defer func() { _ = rows.Close() }()

	memories := make([]*DistilledMemory, 0)
	for rows.Next() {
		memory := &DistilledMemory{}
		var embeddingStr string

		err := rows.Scan(
			&memory.ID, &memory.TenantID, &memory.UserID, &memory.SessionID,
			&memory.Content, &embeddingStr, &memory.EmbeddingModel,
			&memory.EmbeddingVersion, &memory.MemoryType, &memory.Importance,
			&memory.Metadata, &memory.AccessCount, &memory.LastAccessedAt,
			&memory.ExpiresAt, &memory.CreatedAt,
		)
		if err != nil {
			continue
		}

		memory.Embedding, err = parseDistilledVectorString(embeddingStr)
		if err != nil {
			continue
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// UpdateAccessCount updates the access count and last accessed time.
func (r *DistilledMemoryRepository) UpdateAccessCount(ctx context.Context, id string) error {
	query := `
		UPDATE distilled_memories
		SET access_count = access_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("update access count: %w", err)
	}

	return nil
}

// DeleteExpired removes expired memories.
func (r *DistilledMemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM distilled_memories WHERE expires_at <= NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired memories: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}