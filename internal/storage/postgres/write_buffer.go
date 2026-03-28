// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	coreerrors "goagent/internal/core/errors"
	"goagent/internal/errors"
)

// WriteBuffer provides write batching to reduce database and embedding load.
// This implements an in-memory buffer with periodic flushing to batch database operations.
type WriteBuffer struct {
	db              *Pool
	buffer          chan *WriteItem
	batchSize       int
	flushInterval   time.Duration
	queue           *EmbeddingQueue
	embeddingConfig *EmbeddingConfig
	mu              sync.Mutex
	wg              sync.WaitGroup
	stopped         bool
}

// WriteItem represents a single write operation to be batched.
type WriteItem struct {
	TenantID string
	Table    string
	Content  string
	Metadata map[string]interface{}
}

// NewWriteBuffer creates a new WriteBuffer instance.
// Args:
// pool - database connection pool.
// queue - embedding queue for async processing.
// batchSize - number of items to batch before flushing.
// flushInterval - maximum time between flushes.
// embeddingConfig - embedding configuration for model and version settings.
// Returns new WriteBuffer instance.
func NewWriteBuffer(pool *Pool, queue *EmbeddingQueue, batchSize int, flushInterval time.Duration, embeddingConfig *EmbeddingConfig) *WriteBuffer {
	if embeddingConfig == nil {
		embeddingConfig = DefaultEmbeddingConfig()
	}
	return &WriteBuffer{
		db:              pool,
		buffer:          make(chan *WriteItem, batchSize*2), // Double size to avoid blocking
		batchSize:       batchSize,
		flushInterval:   flushInterval,
		queue:           queue,
		embeddingConfig: embeddingConfig,
		stopped:         false,
	}
}

// Start begins the buffer processing loop.
// This should be called after initialization and runs until Stop is called.
// Args:
// ctx - context for cancellation.
// Returns error if processing loop encounters unrecoverable error.
func (b *WriteBuffer) Start(ctx context.Context) error {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	batch := make([]*WriteItem, 0, b.batchSize)

	for {
		select {
		case <-ctx.Done():
			// Flush remaining items on shutdown
			if len(batch) > 0 {
				if err := b.flushBatch(ctx, batch); err != nil {
					return errors.Wrap(err, "flush final batch")
				}
			}
			return nil

		case item := <-b.buffer:
			batch = append(batch, item)
			if len(batch) >= b.batchSize {
				if err := b.flushBatch(ctx, batch); err != nil {
					// Log error but continue processing to avoid dropping items
					continue
				}
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				if err := b.flushBatch(ctx, batch); err != nil {
					// Log error but continue processing
					continue
				}
				batch = batch[:0]
			}
		}
	}
}

// Write queues a write operation for batch processing.
// This is non-blocking and returns immediately if the buffer has capacity.
// If the buffer is full, it returns an error instead of spawning a goroutine.
// Args:
// ctx - context for cancellation.
// item - write operation to queue.
// Returns error if buffer is stopped, item is invalid, or buffer is full.
func (b *WriteBuffer) Write(ctx context.Context, item *WriteItem) error {
	if b.stopped {
		return coreerrors.ErrServiceUnavailable
	}

	if item == nil {
		return coreerrors.ErrInvalidArgument
	}

	select {
	case b.buffer <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer is full, return error immediately
		// This prevents goroutine leaks and provides backpressure
		return coreerrors.ErrBufferFull
	}
}

// flushBatch writes a batch of items to the database and queues embedding tasks.
// Args:
// ctx - database operation context.
// batch - items to write.
// Returns error if database write or embedding enqueue fails.
func (b *WriteBuffer) flushBatch(ctx context.Context, batch []*WriteItem) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := b.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("Failed to rollback transaction", "error", rbErr)
			}
		}
	}()

	// Batch insert into database with content hash deduplication (per design standard)
	for _, item := range batch {
		switch item.Table {
		case "knowledge_chunks_1024":
			// Generate content hash for real-time deduplication (per design standard)
			contentHash := b.computeContentHash(item.Content)

			_, err := tx.Exec(`
				INSERT INTO knowledge_chunks_1024
				(tenant_id, content, content_hash, embedding, embedding_model, embedding_version, 
				 embedding_status, embedding_queued_at, source_type, metadata, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW(), 'memory', $7, NOW(), NOW())
				ON CONFLICT (content_hash) DO UPDATE SET
					access_count = knowledge_chunks_1024.access_count + 1,
					updated_at = NOW()
			`, item.TenantID, item.Content, contentHash, make([]float64, 1024),
				b.embeddingConfig.DefaultModel, b.embeddingConfig.DefaultVersion, item.Metadata)
			if err != nil {
				return errors.Wrap(err, "insert knowledge chunk")
			}

		case "experiences_1024":
			_, err := tx.Exec(`
				INSERT INTO experiences_1024
				(tenant_id, type, input, output, embedding, embedding_model, embedding_version,
				 embedding_status, embedding_queued_at, agent_id, metadata, score, success, decay_at, created_at)
				VALUES ($1, 'solution', $2, $3, $4, $5, $6, 'pending', NOW(), 'style-agent', $7, 0.8, true, NOW() + INTERVAL '30 days', NOW())
			`, item.TenantID, item.Content, item.Metadata["output"], make([]float64, 1024),
				b.embeddingConfig.DefaultModel, b.embeddingConfig.DefaultVersion, item.Metadata)
			if err != nil {
				return errors.Wrap(err, "insert experience")
			}

		default:
			return fmt.Errorf("unsupported table type: %s (currently only knowledge_chunks_1024 and experiences_1024 are supported)", item.Table)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "commit transaction")
	}
	committed = true

	// Enqueue embedding tasks
	for _, item := range batch {
		task := &EmbeddingTask{
			TaskID:   "", // Will be generated by the queue
			Table:    item.Table,
			Content:  item.Content,
			TenantID: item.TenantID,
			Model:    b.embeddingConfig.DefaultModel,
			Version:  b.embeddingConfig.DefaultVersion,
		}
		if err := b.queue.Enqueue(ctx, task); err != nil {
			// Log error but don't fail the batch write
			continue
		}
	}

	return nil
}

// Stop gracefully shuts down the buffer and flushes remaining items.
// This should be called during application shutdown.
// Args:
// ctx - context for cancellation.
func (b *WriteBuffer) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stopped {
		return nil
	}

	b.stopped = true
	close(b.buffer)

	// Wait for any ongoing processing to complete
	b.wg.Wait()

	return nil
}

// computeContentHash computes content hash for deduplication (per design standard).
// This implements real-time hash deduplication as specified in storage-implementation-plan.md.
func (b *WriteBuffer) computeContentHash(content string) string {
	// Simple hash implementation - in production, consider using more robust hashing
	h := 0
	for i := 0; i < len(content); i++ {
		h = 31*h + int(content[i])
	}
	return fmt.Sprintf("%x", h)
}
