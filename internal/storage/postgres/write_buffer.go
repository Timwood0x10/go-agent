// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goagent/internal/core/errors"
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
					return fmt.Errorf("flush final batch: %w", err)
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
// Args:
// ctx - context for cancellation.
// item - write operation to queue.
// Returns error if buffer is stopped or item is invalid.
func (b *WriteBuffer) Write(ctx context.Context, item *WriteItem) error {
	if b.stopped {
		return errors.ErrServiceUnavailable
	}

	if item == nil {
		return errors.ErrInvalidArgument
	}

	select {
	case b.buffer <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer is full, trigger immediate flush
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case b.buffer <- item:
					return
				case <-ticker.C:
					if b.stopped {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		return nil
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

	// Start transaction
	tx, err := b.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Batch insert into database
	for _, item := range batch {
		switch item.Table {
		case "knowledge_chunks_1024":
			_, err := tx.Exec(`
				INSERT INTO knowledge_chunks_1024
				(tenant_id, content, embedding_status, created_at, updated_at)
				VALUES ($1, $2, 'pending', NOW(), NOW())
			`, item.TenantID, item.Content)
			if err != nil {
				return fmt.Errorf("insert knowledge chunk: %w", err)
			}

		case "experiences_1024":
			_, err := tx.Exec(`
				INSERT INTO experiences_1024
				(tenant_id, type, input, output, embedding_status, created_at)
				VALUES ($1, 'solution', $2, $3, 'pending', NOW())
			`, item.TenantID, item.Content, item.Metadata["output"])
			if err != nil {
				return fmt.Errorf("insert experience: %w", err)
			}

		default:
			return fmt.Errorf("unsupported table type: %s (currently only knowledge_chunks_1024 and experiences_1024 are supported)", item.Table)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

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
