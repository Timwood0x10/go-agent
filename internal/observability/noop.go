package observability

import (
	"context"
	"fmt"
	"sync/atomic"
)

// traceIDKey is the context key for trace ID.
type traceIDKey string

const defaultTraceID traceIDKey = "trace_id"

var traceCounter uint64

// NoopTracer is a no-op implementation of Tracer.
// It can be used as a default tracer when observability is not needed.
type NoopTracer struct{}

// NewNoopTracer creates a new NoopTracer.
func NewNoopTracer() Tracer {
	return &NoopTracer{}
}

// RecordLLMCall implements Tracer.
func (t *NoopTracer) RecordLLMCall(ctx context.Context, call *LLMCall) {
	// No-op
}

// RecordToolCall implements Tracer.
func (t *NoopTracer) RecordToolCall(ctx context.Context, call *ToolCall) {
	// No-op
}

// RecordAgentStep implements Tracer.
func (t *NoopTracer) RecordAgentStep(ctx context.Context, step *AgentStep) {
	// No-op
}

// RecordError implements Tracer.
func (t *NoopTracer) RecordError(ctx context.Context, err *AgentError) {
	// No-op
}

// GetTraceID returns the trace ID from context.
func (t *NoopTracer) GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(defaultTraceID).(string); ok {
		return id
	}
	return ""
}

// WithTrace returns a new context with a new trace ID.
func (t *NoopTracer) WithTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, defaultTraceID, generateTraceID())
}

// generateTraceID generates a unique trace ID.
func generateTraceID() string {
	id := atomic.AddUint64(&traceCounter, 1)
	return fmt.Sprintf("trace-%d", id)
}
