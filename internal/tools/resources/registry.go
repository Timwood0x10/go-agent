package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Registry manages tool registration and lookup.
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool == nil {
		return ErrNilTool
	}

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrToolAlreadyRegistered, name)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Execute executes a tool by name.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (Result, error) {
	tool, exists := r.Get(name)
	if !exists {
		return Result{}, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return tool.Execute(ctx, params)
}

// Clear removes all tools.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}

// Registry errors.
var (
	ErrNilTool               = errors.New("tool is nil")
	ErrToolNotFound          = errors.New("tool not found")
	ErrToolAlreadyRegistered = errors.New("tool already registered")
)

// GlobalRegistry is the default tool registry.
var GlobalRegistry = NewRegistry()

// Register registers a tool in the global registry.
func Register(tool Tool) error {
	return GlobalRegistry.Register(tool)
}

// Get retrieves a tool from the global registry.
func Get(name string) (Tool, bool) {
	return GlobalRegistry.Get(name)
}

// List returns all tools from the global registry.
func List() []string {
	return GlobalRegistry.List()
}

// Execute executes a tool from the global registry.
func Execute(ctx context.Context, name string, params map[string]interface{}) (Result, error) {
	return GlobalRegistry.Execute(ctx, name, params)
}

// ToolGroup groups related tools.
type ToolGroup struct {
	name        string
	description string
	registry    *Registry
}

// NewToolGroup creates a new ToolGroup.
func NewToolGroup(name, description string) *ToolGroup {
	return &ToolGroup{
		name:        name,
		description: description,
		registry:    NewRegistry(),
	}
}

// Register registers a tool in the group.
func (g *ToolGroup) Register(tool Tool) error {
	return g.registry.Register(tool)
}

// Get retrieves a tool from the group.
func (g *ToolGroup) Get(name string) (Tool, bool) {
	return g.registry.Get(name)
}

// List returns all tool names in the group.
func (g *ToolGroup) List() []string {
	return g.registry.List()
}

// Name returns the group name.
func (g *ToolGroup) Name() string {
	return g.name
}

// Description returns the group description.
func (g *ToolGroup) Description() string {
	return g.description
}
