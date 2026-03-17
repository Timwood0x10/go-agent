// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"context"
	"log/slog"
	"time"
)

// EmbeddingReconciler provides eventual consistency for embedding operations.
// This scans for orphaned tasks where DB write succeeded but queue enqueue failed.
type EmbeddingReconciler struct {
	db               *Pool
	queue            *EmbeddingQueue
	interval         time.Duration
	missingThreshold time.Duration
	stopped          bool
}

// NewEmbeddingReconciler creates a new EmbeddingReconciler instance.
// Args:
// db - database connection pool.
// queue - embedding queue for re-enqueuing tasks.
// interval - time between reconciliation scans.
// missingThreshold - time after which a task is considered orphaned.
// Returns new EmbeddingReconciler instance.
func NewEmbeddingReconciler(db *Pool, queue *EmbeddingQueue, interval, missingThreshold time.Duration) *EmbeddingReconciler {
	return &EmbeddingReconciler{
		db:               db,
		queue:            queue,
		interval:         interval,
		missingThreshold: missingThreshold,
		stopped:          false,
	}
}

// Start begins periodic reconciliation scanning.
// This runs until Stop is called or context is cancelled.
// Args:
// ctx - context for cancellation.
func (r *EmbeddingReconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping embedding reconciler")
			return

		case <-ticker.C:
			if err := r.Reconcile(ctx); err != nil {
				slog.Error("Embedding reconciliation failed", "error", err)
			}
		}
	}
}

// Reconcile scans for orphaned embedding tasks and re-enqueues them.
// This addresses the case where DB write succeeded but queue enqueue failed.
// Args:
// ctx - database operation context.
// Returns error if reconciliation fails.
func (r *EmbeddingReconciler) Reconcile(ctx context.Context) error {
	slog.Debug("Starting embedding reconciliation")

	// Find knowledge chunks with pending embedding status that haven't been processed recently
	query := `
		SELECT id, tenant_id, content
		FROM knowledge_chunks_1024
		WHERE embedding_status = 'pending'
		  AND embedding_queued_at < NOW() - $1
		  AND embedding_processed_at IS NULL
		LIMIT 1000
	`

	rows, err := r.db.Query(ctx, query, r.missingThreshold)
	if err != nil {
		return err
	}
	defer rows.Close()

	reconciledCount := 0
	for rows.Next() {
		var id, tenantID, content string
		if err := rows.Scan(&id, &tenantID, &content); err != nil {
			slog.Error("Failed to scan orphaned task", "error", err)
			continue
		}

		// Re-enqueue the task
		task := &EmbeddingTask{
			TaskID:   id,
			Table:    "knowledge_chunks_1024",
			Content:  content,
			TenantID: tenantID,
			Model:    "intfloat/e5-large",
			Version:  1,
		}

		if err := r.queue.Enqueue(ctx, task); err != nil {
			slog.Error("Failed to re-enqueue orphaned task", "task_id", id, "error", err)
			continue
		}

		// Update queued_at to prevent immediate re-scanning
		_, err = r.db.Exec(ctx, `
			UPDATE knowledge_chunks_1024
			SET embedding_queued_at = NOW()
			WHERE id = $1
		`, id)

		if err != nil {
			slog.Error("Failed to update queued_at for orphaned task", "task_id", id, "error", err)
			continue
		}

		reconciledCount++
	}

	if reconciledCount > 0 {
		slog.Info("Reconciled orphaned embedding tasks", "count", reconciledCount)
	}

	return nil
}

// Stop gracefully shuts down the reconciler.
func (r *EmbeddingReconciler) Stop() {
	r.stopped = true
}
