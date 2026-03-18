// Package repositories provides data access layer for storage system.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"goagent/internal/core/errors"
	storage_models "goagent/internal/storage/postgres/models"
)

// KnowledgeRepository provides data access for knowledge chunks.
// This implements CRUD operations and vector search for RAG knowledge base.
type KnowledgeRepository struct {
	db *sql.DB
}

// NewKnowledgeRepository creates a new KnowledgeRepository instance.
// Args:
// db - database connection.
// Returns new KnowledgeRepository instance.
func NewKnowledgeRepository(db *sql.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

// Create inserts a new knowledge chunk into the database.
// Args:
// ctx - database operation context.
// chunk - knowledge chunk to create.
// Returns error if insert operation fails.
func (r *KnowledgeRepository) Create(ctx context.Context, chunk *storage_models.KnowledgeChunk) error {
	query := `
		INSERT INTO knowledge_chunks_1024
		(id, tenant_id, content, embedding, embedding_model, embedding_version, 
		 embedding_status, source_type, source, metadata, document_id, 
		 chunk_index, content_hash, access_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (content_hash) DO UPDATE SET
			access_count = knowledge_chunks_1024.access_count + 1,
			updated_at = NOW()
		RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query,
		chunk.ID, chunk.TenantID, chunk.Content, chunk.Embedding,
		chunk.EmbeddingModel, chunk.EmbeddingVersion, chunk.EmbeddingStatus,
		chunk.SourceType, chunk.Source, chunk.Metadata, chunk.DocumentID,
		chunk.ChunkIndex, chunk.ContentHash, chunk.AccessCount,
		chunk.CreatedAt, chunk.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("create knowledge chunk: %w", err)
	}

	chunk.ID = id
	return nil
}

// CreateBatch inserts multiple knowledge chunks in a single transaction.
// Args:
// ctx - database operation context.
// chunks - knowledge chunks to create.
// Returns error if any insert operation fails.
func (r *KnowledgeRepository) CreateBatch(ctx context.Context, chunks []*storage_models.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO knowledge_chunks_1024
		(id, tenant_id, content, embedding, embedding_model, embedding_version, 
		 embedding_status, source_type, source, metadata, document_id, 
		 chunk_index, content_hash, access_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (content_hash) DO UPDATE SET
			access_count = knowledge_chunks_1024.access_count + 1,
			updated_at = NOW()
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		_, err := stmt.ExecContext(ctx,
			chunk.ID, chunk.TenantID, chunk.Content, chunk.Embedding,
			chunk.EmbeddingModel, chunk.EmbeddingVersion, chunk.EmbeddingStatus,
			chunk.SourceType, chunk.Source, chunk.Metadata, chunk.DocumentID,
			chunk.ChunkIndex, chunk.ContentHash, chunk.AccessCount,
			chunk.CreatedAt, chunk.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("create knowledge chunk: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a knowledge chunk by its ID.
// Args:
// ctx - database operation context.
// id - knowledge chunk identifier.
// Returns knowledge chunk or error if not found.
func (r *KnowledgeRepository) GetByID(ctx context.Context, id string) (*storage_models.KnowledgeChunk, error) {
	query := `
		SELECT id, tenant_id, content, embedding, embedding_model, embedding_version,
			   embedding_status, source_type, source, metadata, document_id,
			   chunk_index, content_hash, access_count, created_at, updated_at
		FROM knowledge_chunks_1024
		WHERE id = $1
	`

	chunk := &storage_models.KnowledgeChunk{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,
		&chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
		&chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
		&chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
		&chunk.CreatedAt, &chunk.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get knowledge chunk by id: %w", err)
	}

	return chunk, nil
}

// Update updates an existing knowledge chunk.
// Args:
// ctx - database operation context.
// chunk - knowledge chunk with updated values.
// Returns error if update operation fails.
func (r *KnowledgeRepository) Update(ctx context.Context, chunk *storage_models.KnowledgeChunk) error {
	query := `
		UPDATE knowledge_chunks_1024
		SET content = $2, embedding = $3, embedding_status = $4,
			source_type = $5, source = $6, metadata = $7,
			document_id = $8, chunk_index = $9, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		chunk.ID, chunk.Content, chunk.Embedding, chunk.EmbeddingStatus,
		chunk.SourceType, chunk.Source, chunk.Metadata,
		chunk.DocumentID, chunk.ChunkIndex,
	)
	if err != nil {
		return fmt.Errorf("update knowledge chunk: %w", err)
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

// Delete removes a knowledge chunk by its ID.
// Args:
// ctx - database operation context.
// id - knowledge chunk identifier.
// Returns error if delete operation fails.
func (r *KnowledgeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM knowledge_chunks_1024 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete knowledge chunk: %w", err)
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

// SearchByVector performs vector similarity search.
// Args:
// ctx - database operation context.
// embedding - query vector embedding.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of similar knowledge chunks ordered by similarity.
func (r *KnowledgeRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*storage_models.KnowledgeChunk, error) {
	query := `
		SELECT id, tenant_id, content, embedding, embedding_model, embedding_version,
			   embedding_status, source_type, source, metadata, document_id,
			   chunk_index, content_hash, access_count, created_at, updated_at,
			   1 - (embedding <=> $1) as similarity
		FROM knowledge_chunks_1024
		WHERE tenant_id = $2
		  AND embedding_status = 'completed'
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	chunks := make([]*storage_models.KnowledgeChunk, 0)
	for rows.Next() {
		chunk := &storage_models.KnowledgeChunk{}
		var similarity float64
		err := rows.Scan(
			&chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,
			&chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
			&chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
			&chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
			&chunk.CreatedAt, &chunk.UpdatedAt, &similarity,
		)
		if err != nil {
			continue
		}
		// Store similarity in metadata for downstream processing
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["similarity"] = similarity
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// SearchByKeyword performs full-text search using BM25.
// Args:
// ctx - database operation context.
// query - search query text.
// tenantID - tenant identifier for isolation.
// limit - maximum number of results to return.
// Returns list of matching knowledge chunks ordered by relevance.
func (r *KnowledgeRepository) SearchByKeyword(ctx context.Context, query, tenantID string, limit int) ([]*storage_models.KnowledgeChunk, error) {
	sqlQuery := `
		SELECT id, tenant_id, content, embedding, embedding_model, embedding_version,
			   embedding_status, source_type, source, metadata, document_id,
			   chunk_index, content_hash, access_count, created_at, updated_at,
			   ts_rank(tsv, plainto_tsquery('simple', $1)) as score
		FROM knowledge_chunks_1024
		WHERE tsv @@ plainto_tsquery('simple', $1)
		  AND tenant_id = $2
		  AND embedding_status = 'completed'
		ORDER BY ts_rank(tsv, plainto_tsquery('simple', $1)) DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}
	defer rows.Close()

	chunks := make([]*storage_models.KnowledgeChunk, 0)
	for rows.Next() {
		chunk := &storage_models.KnowledgeChunk{}
		var score float64
		err := rows.Scan(
			&chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,
			&chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
			&chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
			&chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
			&chunk.CreatedAt, &chunk.UpdatedAt, &score,
		)
		if err != nil {
			continue
		}
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata["keyword_score"] = score
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// ListByDocument retrieves all chunks for a specific document.
// Args:
// ctx - database operation context.
// documentID - document identifier.
// tenantID - tenant identifier for isolation.
// Returns list of knowledge chunks ordered by chunk index.
func (r *KnowledgeRepository) ListByDocument(ctx context.Context, documentID, tenantID string) ([]*storage_models.KnowledgeChunk, error) {
	query := `
		SELECT id, tenant_id, content, embedding, embedding_model, embedding_version,
			   embedding_status, source_type, source, metadata, document_id,
			   chunk_index, content_hash, access_count, created_at, updated_at
		FROM knowledge_chunks_1024
		WHERE document_id = $1 AND tenant_id = $2
		ORDER BY chunk_index ASC
	`

	rows, err := r.db.QueryContext(ctx, query, documentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list chunks by document: %w", err)
	}
	defer rows.Close()

	chunks := make([]*storage_models.KnowledgeChunk, 0)
	for rows.Next() {
		chunk := &storage_models.KnowledgeChunk{}
		err := rows.Scan(
			&chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,
			&chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
			&chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
			&chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
			&chunk.CreatedAt, &chunk.UpdatedAt,
		)
		if err != nil {
			continue
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// UpdateEmbedding updates the embedding for a knowledge chunk.
// Args:
// ctx - database operation context.
// id - knowledge chunk identifier.
// embedding - vector embedding.
// model - embedding model name.
// version - embedding model version.
// Returns error if update operation fails.
func (r *KnowledgeRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string, version int) error {
	query := `
		UPDATE knowledge_chunks_1024
		SET embedding = $2, embedding_model = $3, embedding_version = $4,
			embedding_status = 'completed', embedding_processed_at = NOW(),
			updated_at = NOW()
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

// UpdateEmbeddingStatus updates the embedding processing status.
// Args:
// ctx - database operation context.
// id - knowledge chunk identifier.
// status - new embedding status.
// error - error message if status is failed.
// Returns error if update operation fails.
func (r *KnowledgeRepository) UpdateEmbeddingStatus(ctx context.Context, id, status, errorMsg string) error {
	query := `
		UPDATE knowledge_chunks_1024
		SET embedding_status = $2, embedding_error = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, errorMsg)
	if err != nil {
		return fmt.Errorf("update embedding status: %w", err)
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

// CleanupExpired removes knowledge chunks that are no longer needed.
// Args:
// ctx - database operation context.
// olderThan - cutoff time for deletion.
// Returns number of deleted chunks or error if operation fails.
func (r *KnowledgeRepository) CleanupExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `
		DELETE FROM knowledge_chunks_1024
		WHERE updated_at < $1
		  AND access_count < 10
	`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired chunks: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}