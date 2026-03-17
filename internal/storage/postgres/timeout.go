package postgres

import (
	"context"
	"time"
)

// DefaultTimeouts defines default timeout values for database operations.
var DefaultTimeouts = struct {
	Query      time.Duration
	Insert     time.Duration
	Update     time.Duration
	Delete     time.Duration
	Transaction time.Duration
	VectorSearch time.Duration
}{
	Query:      30 * time.Second,
	Insert:     20 * time.Second,
	Update:     20 * time.Second,
	Delete:     20 * time.Second,
	Transaction: 60 * time.Second,
	VectorSearch: 10 * time.Second,
}

// WithQueryTimeout ensures the context has a timeout suitable for query operations.
func WithQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.Query)
}

// WithInsertTimeout ensures the context has a timeout suitable for insert operations.
func WithInsertTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.Insert)
}

// WithUpdateTimeout ensures the context has a timeout suitable for update operations.
func WithUpdateTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.Update)
}

// WithDeleteTimeout ensures the context has a timeout suitable for delete operations.
func WithDeleteTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.Delete)
}

// WithTransactionTimeout ensures the context has a timeout suitable for transaction operations.
func WithTransactionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.Transaction)
}

// WithVectorSearchTimeout ensures the context has a timeout suitable for vector search operations.
func WithVectorSearchTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return withTimeout(ctx, DefaultTimeouts.VectorSearch)
}

// withTimeout ensures the context has a timeout.
// If the context already has a deadline, it uses the existing deadline.
// Otherwise, it adds the default timeout.
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		// Context already has a deadline, return it as-is
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}