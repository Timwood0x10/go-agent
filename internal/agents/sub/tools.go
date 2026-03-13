package sub

import (
	"context"
	"sync"

	"goagent/internal/core/errors"
)

// toolBinder binds and calls tools.
type toolBinder struct {
	mu    sync.RWMutex
	tools map[string]func(ctx context.Context, args map[string]any) (any, error)
}

// NewToolBinder creates a new ToolBinder.
func NewToolBinder() ToolBinder {
	return &toolBinder{
		tools: make(map[string]func(ctx context.Context, args map[string]any) (any, error)),
	}
}

// BindTool binds a tool function to the agent.
func (b *toolBinder) BindTool(name string, toolFunc func(ctx context.Context, args map[string]any) (any, error)) {
	if name == "" || toolFunc == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tools[name] = toolFunc
}

// CallTool calls a bound tool by name.
func (b *toolBinder) CallTool(ctx context.Context, name string, args map[string]any) (any, error) {
	b.mu.RLock()
	toolFunc, ok := b.tools[name]
	b.mu.RUnlock()

	if !ok {
		return nil, errors.ErrToolNotFound
	}

	return toolFunc(ctx, args)
}

// ListTools returns all available tool names.
func (b *toolBinder) ListTools() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	names := make([]string, 0, len(b.tools))
	for name := range b.tools {
		names = append(names, name)
	}
	return names
}
