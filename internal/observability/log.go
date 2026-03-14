package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// LogTracer is a tracer that logs to standard output.
// It can be used for development and debugging.
type LogTracer struct {
	mu         sync.RWMutex
	enableJSON bool
}

// LogTracerConfig holds configuration for LogTracer.
type LogTracerConfig struct {
	EnableJSON bool
}

// NewLogTracer creates a new LogTracer.
func NewLogTracer(cfg *LogTracerConfig) Tracer {
	if cfg == nil {
		cfg = &LogTracerConfig{}
	}
	return &LogTracer{
		enableJSON: cfg.EnableJSON,
	}
}

// RecordLLMCall implements Tracer.
func (t *LogTracer) RecordLLMCall(ctx context.Context, call *LLMCall) {
	t.log("llm_call", map[string]any{
		"trace_id":     call.TraceID,
		"model":        call.Model,
		"prompt_len":   len(call.Prompt),
		"response_len": len(call.Response),
		"tokens":       call.TokensUsed,
		"duration_ms":  call.Duration.Milliseconds(),
		"error":        callErrorToString(call.Error),
	})
}

// RecordToolCall implements Tracer.
func (t *LogTracer) RecordToolCall(ctx context.Context, call *ToolCall) {
	t.log("tool_call", map[string]any{
		"trace_id":    call.TraceID,
		"tool_name":   call.ToolName,
		"input":       call.Input,
		"output":      call.Output,
		"duration_ms": call.Duration.Milliseconds(),
		"error":       callErrorToString(call.Error),
	})
}

// RecordAgentStep implements Tracer.
func (t *LogTracer) RecordAgentStep(ctx context.Context, step *AgentStep) {
	t.log("agent_step", map[string]any{
		"trace_id":    step.TraceID,
		"agent_id":    step.AgentID,
		"step_name":   step.StepName,
		"metadata":    step.Metadata,
		"duration_ms": step.Duration.Milliseconds(),
		"error":       callErrorToString(step.Error),
	})
}

// RecordError implements Tracer.
func (t *LogTracer) RecordError(ctx context.Context, err *AgentError) {
	t.log("error", map[string]any{
		"trace_id":   err.TraceID,
		"agent_id":   err.AgentID,
		"error_type": err.ErrorType,
		"message":    err.Message,
		"metadata":   err.Metadata,
	})
}

// GetTraceID returns the trace ID from context.
func (t *LogTracer) GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(defaultTraceID).(string); ok {
		return id
	}
	return ""
}

// WithTrace returns a new context with a new trace ID.
func (t *LogTracer) WithTrace(ctx context.Context) context.Context {
	traceID := generateTraceID()
	return context.WithValue(ctx, defaultTraceID, traceID)
}

// log writes a log entry.
func (t *LogTracer) log(eventType string, data map[string]any) {
	if t.enableJSON {
		data["event_type"] = eventType
		data["timestamp"] = time.Now().Format(time.RFC3339)
		jsonData, _ := json.Marshal(data)
		log.Println(string(jsonData))
	} else {
		log.Printf("[%s] %s", eventType, formatLog(data))
	}
}

// formatLog formats log data as a human-readable string.
func formatLog(data map[string]any) string {
	result := ""
	for k, v := range data {
		result += fmt.Sprintf("%s=%v ", k, v)
	}
	return result
}

// callErrorToString converts an error to string.
func callErrorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
