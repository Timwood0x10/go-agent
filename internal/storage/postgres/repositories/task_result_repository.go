// Package repositories provides data access layer for storage system.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"goagent/internal/core/errors"
	storage_models "goagent/internal/storage/postgres/models"
)

// TaskResultRepository provides data access for task execution results.
// This implements CRUD operations and vector search for task results.
type TaskResultRepository struct {
	db *sql.DB
}

// NewTaskResultRepository creates a new TaskResultRepository instance.
// Args:
// db - database connection.
// Returns new TaskResultRepository instance.
func NewTaskResultRepository(db *sql.DB) *TaskResultRepository {
	return &TaskResultRepository{db: db}
}

// Create inserts a new task result into the database.
// Args:
// ctx - database operation context.
// result - task result to create.
// Returns error if insert operation fails.
func (r *TaskResultRepository) Create(ctx context.Context, result *storage_models.TaskResult) error {
	inputJSON, err := json.Marshal(result.Input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}

	var outputJSON []byte
	if result.Output != nil {
		outputJSON, err = json.Marshal(result.Output)
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}
	}

	query := `
		INSERT INTO task_results_1024
		(id, tenant_id, session_id, task_type, agent_id, input, output, embedding,
		 embedding_model, embedding_version, status, error, latency_ms, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var id string
	err = r.db.QueryRowContext(ctx, query,
		result.ID, result.TenantID, result.SessionID, result.TaskType,
		result.AgentID, inputJSON, outputJSON, result.Embedding,
		result.EmbeddingModel, result.EmbeddingVersion, result.Status,
		result.Error, result.LatencyMs, result.Metadata, result.CreatedAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("create task result: %w", err)
	}

	result.ID = id
	return nil
}

// GetByID retrieves a task result by its ID.
// Args:
// ctx - database operation context.
// id - task result identifier.
// Returns task result or error if not found.
func (r *TaskResultRepository) GetByID(ctx context.Context, id string) (*storage_models.TaskResult, error) {
	query := `
		SELECT id, tenant_id, session_id, task_type, agent_id, input, output, embedding,
			   embedding_model, embedding_version, status, error, latency_ms, metadata, created_at
		FROM task_results_1024
		WHERE id = $1
	`

	result := &storage_models.TaskResult{}
	var inputJSON, outputJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&result.ID, &result.TenantID, &result.SessionID, &result.TaskType,
		&result.AgentID, &inputJSON, &outputJSON, &result.Embedding,
		&result.EmbeddingModel, &result.EmbeddingVersion, &result.Status,
		&result.Error, &result.LatencyMs, &result.Metadata, &result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task result by id: %w", err)
	}

	if err := json.Unmarshal(inputJSON, &result.Input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}

	if outputJSON != nil {
		if err := json.Unmarshal(outputJSON, &result.Output); err != nil {
			return nil, fmt.Errorf("unmarshal output: %w", err)
		}
	}

	return result, nil
}

// Update updates an existing task result.
// Args:
// ctx - database operation context.
// result - task result with updated values.
// Returns error if update operation fails.
func (r *TaskResultRepository) Update(ctx context.Context, result *storage_models.TaskResult) error {
	inputJSON, err := json.Marshal(result.Input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}

	var outputJSON []byte
	if result.Output != nil {
		outputJSON, err = json.Marshal(result.Output)
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}
	}

	query := `
		UPDATE task_results_1024
		SET task_type = $2, agent_id = $3, input = $4, output = $5, embedding = $6,
			embedding_model = $7, embedding_version = $8, status = $9, error = $10,
			latency_ms = $11, metadata = $12
		WHERE id = $1
	`

	resultSQL, err := r.db.ExecContext(ctx, query,
		result.ID, result.TaskType, result.AgentID, inputJSON, outputJSON,
		result.Embedding, result.EmbeddingModel, result.EmbeddingVersion,
		result.Status, result.Error, result.LatencyMs, result.Metadata,
	)
	if err != nil {
		return fmt.Errorf("update task result: %w", err)
	}

	rows, err := resultSQL.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// Delete removes a task result by its ID.
// Args:
// ctx - database operation context.
// id - task result identifier.
// Returns error if delete operation fails.
func (r *TaskResultRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM task_results_1024 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task result: %w", err)
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

// SearchByVector performs vector similarity search for task results.
// Args:
// ctx - database operation context.
// embedding - query vector embedding.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of similar task results ordered by similarity.
func (r *TaskResultRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*storage_models.TaskResult, error) {
	query := `
		SELECT id, tenant_id, session_id, task_type, agent_id, input, output, embedding,
			   embedding_model, embedding_version, status, error, latency_ms, metadata, created_at,
			   1 - (embedding <=> $1) as similarity
		FROM task_results_1024
		WHERE tenant_id = $2
		  AND embedding IS NOT NULL
		  AND status = 'completed'
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	results := make([]*storage_models.TaskResult, 0)
	for rows.Next() {
		result := &storage_models.TaskResult{}
		var inputJSON, outputJSON []byte
		var similarity float64

		err := rows.Scan(
			&result.ID, &result.TenantID, &result.SessionID, &result.TaskType,
			&result.AgentID, &inputJSON, &outputJSON, &result.Embedding,
			&result.EmbeddingModel, &result.EmbeddingVersion, &result.Status,
			&result.Error, &result.LatencyMs, &result.Metadata, &result.CreatedAt, &similarity,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(inputJSON, &result.Input); err != nil {
			continue
		}

		if outputJSON != nil {
			if err := json.Unmarshal(outputJSON, &result.Output); err != nil {
				continue
			}
		}

		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["similarity"] = similarity
		results = append(results, result)
	}

	return results, nil
}

// ListByType retrieves task results by type.
// Args:
// ctx - database operation context.
// taskType - task type filter.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of task results ordered by created time (descending).
func (r *TaskResultRepository) ListByType(ctx context.Context, taskType, tenantID string, limit int) ([]*storage_models.TaskResult, error) {
	query := `
		SELECT id, tenant_id, session_id, task_type, agent_id, input, output, embedding,
			   embedding_model, embedding_version, status, error, latency_ms, metadata, created_at
		FROM task_results_1024
		WHERE task_type = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, taskType, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list task results by type: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	results := make([]*storage_models.TaskResult, 0)
	for rows.Next() {
		result := &storage_models.TaskResult{}
		var inputJSON, outputJSON []byte

		err := rows.Scan(
			&result.ID, &result.TenantID, &result.SessionID, &result.TaskType,
			&result.AgentID, &inputJSON, &outputJSON, &result.Embedding,
			&result.EmbeddingModel, &result.EmbeddingVersion, &result.Status,
			&result.Error, &result.LatencyMs, &result.Metadata, &result.CreatedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(inputJSON, &result.Input); err != nil {
			continue
		}

		if outputJSON != nil {
			if err := json.Unmarshal(outputJSON, &result.Output); err != nil {
				continue
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// ListBySession retrieves task results for a specific session.
// Args:
// ctx - database operation context.
// sessionID - session identifier.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of task results ordered by created time (descending).
func (r *TaskResultRepository) ListBySession(ctx context.Context, sessionID, tenantID string, limit int) ([]*storage_models.TaskResult, error) {
	query := `
		SELECT id, tenant_id, session_id, task_type, agent_id, input, output, embedding,
			   embedding_model, embedding_version, status, error, latency_ms, metadata, created_at
		FROM task_results_1024
		WHERE session_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list task results by session: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err)
		}
	}()

	results := make([]*storage_models.TaskResult, 0)
	for rows.Next() {
		result := &storage_models.TaskResult{}
		var inputJSON, outputJSON []byte

		err := rows.Scan(
			&result.ID, &result.TenantID, &result.SessionID, &result.TaskType,
			&result.AgentID, &inputJSON, &outputJSON, &result.Embedding,
			&result.EmbeddingModel, &result.EmbeddingVersion, &result.Status,
			&result.Error, &result.LatencyMs, &result.Metadata, &result.CreatedAt,
		)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(inputJSON, &result.Input); err != nil {
			continue
		}

		if outputJSON != nil {
			if err := json.Unmarshal(outputJSON, &result.Output); err != nil {
				continue
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// UpdateEmbedding updates the embedding for a task result.
// Args:
// ctx - database operation context.
// id - task result identifier.
// embedding - vector embedding.
// model - embedding model name.
// version - embedding model version.
// Returns error if update operation fails.
func (r *TaskResultRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string, version int) error {
	query := `
		UPDATE task_results_1024
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

// UpdateStatus updates the status of a task result.
// Args:
// ctx - database operation context.
// id - task result identifier.
// status - new status value.
// errorMsg - error message if status is failed.
// latencyMs - execution latency in milliseconds.
// Returns error if update operation fails.
func (r *TaskResultRepository) UpdateStatus(ctx context.Context, id, status, errorMsg string, latencyMs int) error {
	query := `
		UPDATE task_results_1024
		SET status = $2, error = $3, latency_ms = $4, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, errorMsg, latencyMs)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
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

// GetStatistics returns statistics about task results.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier for isolation.
// Returns task result statistics or error if query fails.
func (r *TaskResultRepository) GetStatistics(ctx context.Context, tenantID string) (map[string]int64, error) {
	query := `
		SELECT
			task_type,
			status,
			COUNT(*) as count
		FROM task_results_1024
		WHERE tenant_id = $1
		GROUP BY task_type, status
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get task result statistics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := make(map[string]int64)
	for rows.Next() {
		var taskType, status string
		var count int64
		if err := rows.Scan(&taskType, &status, &count); err != nil {
			continue
		}
		key := fmt.Sprintf("%s:%s", taskType, status)
		stats[key] = count
	}

	return stats, nil
}
