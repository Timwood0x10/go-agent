// Package repositories provides data access layer for storage system.
package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"goagent/internal/core/errors"
	"goagent/internal/storage/postgres"
	storage_models "goagent/internal/storage/postgres/models"
)

// ToolRepository provides data access for tool definitions.
// This implements CRUD operations and semantic search for tools.
// It depends on the DBTX interface to support both database connections and transactions.
type ToolRepository struct {
	db postgres.DBTX
}

// NewToolRepository creates a new ToolRepository instance.
// Args:
// db - database connection or transaction implementing DBTX interface.
// Returns new ToolRepository instance.
func NewToolRepository(db postgres.DBTX) *ToolRepository {
	return &ToolRepository{db: db}
}

// Create inserts a new tool into the database.
// Args:
// ctx - database operation context.
// tool - tool to create.
// Returns error if insert operation fails.
func (r *ToolRepository) Create(ctx context.Context, tool *storage_models.Tool) error {
	query := `
		INSERT INTO tools
		(id, tenant_id, name, description, embedding, embedding_model, embedding_version,
		 agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id, name) DO UPDATE SET
			description = EXCLUDED.description,
			embedding = EXCLUDED.embedding,
			embedding_model = EXCLUDED.embedding_model,
			embedding_version = EXCLUDED.embedding_version,
			agent_type = EXCLUDED.agent_type,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query,
		tool.ID, tool.TenantID, tool.Name, tool.Description,
		tool.Embedding, tool.EmbeddingModel, tool.EmbeddingVersion,
		tool.AgentType, tool.Tags, tool.UsageCount, tool.SuccessRate,
		tool.LastUsedAt, tool.Metadata, tool.CreatedAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("create tool: %w", err)
	}

	tool.ID = id
	return nil
}

// GetByID retrieves a tool by ID.
// Args:
// ctx - database operation context.
// id - tool ID, must be non-empty.
// Returns tool or error if not found or invalid argument.
func (r *ToolRepository) GetByID(ctx context.Context, id string) (*storage_models.Tool, error) {
	if id == "" {
		return nil, errors.ErrInvalidArgument
	}

	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at
		FROM tools
		WHERE id = $1
	`

	tool := &storage_models.Tool{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
		&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
		&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
		&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tool by id: %w", err)
	}

	return tool, nil
}

// GetByName retrieves a tool by its name within a tenant.
// Args:
// ctx - database operation context.
// name - tool name.
// tenantID - tenant identifier.
// Returns tool or error if not found.
func (r *ToolRepository) GetByName(ctx context.Context, name, tenantID string) (*storage_models.Tool, error) {
	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at
		FROM tools
		WHERE name = $1 AND tenant_id = $2
	`

	tool := &storage_models.Tool{}
	err := r.db.QueryRowContext(ctx, query, name, tenantID).Scan(
		&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
		&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
		&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
		&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tool by name: %w", err)
	}

	return tool, nil
}

// Update updates an existing tool.
// Args:
// ctx - database operation context.
// tool - tool with updated values.
// Returns error if update operation fails.
func (r *ToolRepository) Update(ctx context.Context, tool *storage_models.Tool) error {
	query := `
		UPDATE tools
		SET name = $2, description = $3, embedding = $4, embedding_model = $5,
			embedding_version = $6, agent_type = $7, tags = $8, metadata = $9
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		tool.ID, tool.Name, tool.Description, tool.Embedding,
		tool.EmbeddingModel, tool.EmbeddingVersion, tool.AgentType,
		tool.Tags, tool.Metadata,
	)
	if err != nil {
		return fmt.Errorf("update tool: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// Delete removes a tool by its ID.
// Args:
// ctx - database operation context.
// id - tool identifier.
// Returns error if delete operation fails.
func (r *ToolRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tools WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// SearchByVector performs semantic search for tools using vector embedding.
// Args:
// ctx - database operation context.
// embedding - query vector embedding.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of similar tools ordered by similarity.
func (r *ToolRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*storage_models.Tool, error) {
	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at,
			   1 - (embedding <=> $1) as similarity
		FROM tools
		WHERE tenant_id = $2
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tools := make([]*storage_models.Tool, 0)
	for rows.Next() {
		tool := &storage_models.Tool{}
		var similarity float64
		err := rows.Scan(
			&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
			&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
			&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
			&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt, &similarity,
		)
		if err != nil {
			continue
		}
		if tool.Metadata == nil {
			tool.Metadata = make(map[string]interface{})
		}
		tool.Metadata["similarity"] = similarity
		tools = append(tools, tool)
	}

	return tools, nil
}

// SearchByKeyword performs keyword-based search for tools.
// Args:
// ctx - database operation context.
// query - search query text.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of matching tools ordered by relevance.
func (r *ToolRepository) SearchByKeyword(ctx context.Context, query, tenantID string, limit int) ([]*storage_models.Tool, error) {
	sqlQuery := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at,
			   ts_rank(tsv, plainto_tsquery('simple', $1)) as score
		FROM tools
		WHERE tsv @@ plainto_tsquery('simple', $1)
		  AND tenant_id = $2
		ORDER BY ts_rank(tsv, plainto_tsquery('simple', $1)) DESC, usage_count DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tools := make([]*storage_models.Tool, 0)
	for rows.Next() {
		tool := &storage_models.Tool{}
		var score float64
		err := rows.Scan(
			&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
			&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
			&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
			&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt, &score,
		)
		if err != nil {
			continue
		}
		if tool.Metadata == nil {
			tool.Metadata = make(map[string]interface{})
		}
		tool.Metadata["keyword_score"] = score
		tools = append(tools, tool)
	}

	return tools, nil
}

// ListAll retrieves all tools for a tenant.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of tools ordered by usage count (descending).
func (r *ToolRepository) ListAll(ctx context.Context, tenantID string, limit int) ([]*storage_models.Tool, error) {
	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at
		FROM tools
		WHERE tenant_id = $1
		ORDER BY usage_count DESC, success_rate DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tools := make([]*storage_models.Tool, 0)
	for rows.Next() {
		tool := &storage_models.Tool{}
		err := rows.Scan(
			&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
			&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
			&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
			&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt,
		)
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// ListByAgentType retrieves tools by agent type.
// Args:
// ctx - database operation context.
// agentType - agent type filter.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of tools ordered by usage count (descending).
func (r *ToolRepository) ListByAgentType(ctx context.Context, agentType, tenantID string, limit int) ([]*storage_models.Tool, error) {
	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at
		FROM tools
		WHERE agent_type = $1 AND tenant_id = $2
		ORDER BY usage_count DESC, success_rate DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, agentType, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tools by agent type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tools := make([]*storage_models.Tool, 0)
	for rows.Next() {
		tool := &storage_models.Tool{}
		err := rows.Scan(
			&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
			&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
			&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
			&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt,
		)
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// UpdateUsage updates the usage statistics for a tool.
// Args:
// ctx - database operation context.
// id - tool identifier.
// success - whether the tool execution was successful.
// Returns error if update operation fails.
func (r *ToolRepository) UpdateUsage(ctx context.Context, id string, success bool) error {
	query := `
		UPDATE tools
		SET usage_count = usage_count + 1,
			success_rate = CASE 
				WHEN $2 THEN success_rate * 0.9 + 1.0 * 0.1
				ELSE success_rate * 0.9 + 0.0 * 0.1
			END,
			last_used_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, success)
	if err != nil {
		return fmt.Errorf("update tool usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// UpdateEmbedding updates the embedding for a tool.
// Args:
// ctx - database operation context.
// id - tool identifier.
// embedding - vector embedding.
// model - embedding model name.
// version - embedding model version.
// Returns error if update operation fails.
func (r *ToolRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string, version int) error {
	query := `
		UPDATE tools
		SET embedding = $2, embedding_model = $3, embedding_version = $4, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, embedding, model, version)
	if err != nil {
		return fmt.Errorf("update embedding: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// ListByTags retrieves tools by tags.
// Args:
// ctx - database operation context.
// tags - tags to filter by.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of tools that match any of the tags.
func (r *ToolRepository) ListByTags(ctx context.Context, tags []string, tenantID string, limit int) ([]*storage_models.Tool, error) {
	query := `
		SELECT id, tenant_id, name, description, embedding, embedding_model, embedding_version,
			   agent_type, tags, usage_count, success_rate, last_used_at, metadata, created_at
		FROM tools
		WHERE tenant_id = $1
		  AND tags && $2
		ORDER BY usage_count DESC, success_rate DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, tags, limit)
	if err != nil {
		return nil, fmt.Errorf("list tools by tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tools := make([]*storage_models.Tool, 0)
	for rows.Next() {
		tool := &storage_models.Tool{}
		err := rows.Scan(
			&tool.ID, &tool.TenantID, &tool.Name, &tool.Description,
			&tool.Embedding, &tool.EmbeddingModel, &tool.EmbeddingVersion,
			&tool.AgentType, &tool.Tags, &tool.UsageCount, &tool.SuccessRate,
			&tool.LastUsedAt, &tool.Metadata, &tool.CreatedAt,
		)
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}

	return tools, nil
}
