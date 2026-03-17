package output

import (
	"context"
	"time"
)

// DefaultTimeouts defines default timeout values for different operations.
var DefaultTimeouts = struct {
	LLMRequest          time.Duration
	LLMStructuredOutput time.Duration
	LLMStream           time.Duration
	DatabaseQuery       time.Duration
	DatabaseTransaction time.Duration
	VectorSearch        time.Duration
}{
	LLMRequest:          120 * time.Second,
	LLMStructuredOutput: 180 * time.Second,
	LLMStream:           300 * time.Second,
	DatabaseQuery:       30 * time.Second,
	DatabaseTransaction: 60 * time.Second,
	VectorSearch:        10 * time.Second,
}

// WithDefaultTimeout ensures the context has a timeout.
// If the context already has a deadline, it uses the existing deadline.
// Otherwise, it adds the default timeout.
func WithDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		// Context already has a deadline, return it as-is
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// WithLLMTimeout ensures the context has a timeout suitable for LLM requests.
func WithLLMTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithDefaultTimeout(ctx, DefaultTimeouts.LLMRequest)
}

// WithLLMStructuredTimeout ensures the context has a timeout suitable for structured LLM output.
func WithLLMStructuredTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithDefaultTimeout(ctx, DefaultTimeouts.LLMStructuredOutput)
}

// WithDatabaseTimeout ensures the context has a timeout suitable for database operations.
func WithDatabaseTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithDefaultTimeout(ctx, DefaultTimeouts.DatabaseQuery)
}

// WithDatabaseTransactionTimeout ensures the context has a timeout suitable for database transactions.
func WithDatabaseTransactionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithDefaultTimeout(ctx, DefaultTimeouts.DatabaseTransaction)
}

// WithVectorSearchTimeout ensures the context has a timeout suitable for vector search operations.
func WithVectorSearchTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return WithDefaultTimeout(ctx, DefaultTimeouts.VectorSearch)
}
