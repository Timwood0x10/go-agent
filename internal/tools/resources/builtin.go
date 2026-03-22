package resources

import (
	"context"
	"fmt"
	"log/slog"
)

// RegisterBuiltinTools registers all built-in tools to the global registry.
// This includes tools for knowledge base operations, HTTP requests, calculations, etc.
func RegisterBuiltinTools() error {
	// Register knowledge base tools (requires RetrievalService)
	// Note: These need to be registered with a service instance
	slog.Info("Built-in tools ready for registration")
	slog.Info("Available built-in tools:")
	slog.Info("  - http_request: Perform HTTP requests")
	slog.Info("  - calculator: Mathematical calculations")
	slog.Info("  - datetime: Date and time operations")
	slog.Info("  - text_processor: Text processing operations")
	slog.Info("  - knowledge_search: Search knowledge base (requires service)")
	slog.Info("  - knowledge_add: Add knowledge (requires service)")
	slog.Info("  - knowledge_update: Update knowledge (requires service)")
	slog.Info("  - knowledge_delete: Delete knowledge (requires service)")
	slog.Info("  - weather_check: Check weather (requires provider)")
	slog.Info("  - fashion_search: Search fashion items (requires searcher)")
	slog.Info("  - style_recommend: Get style recommendations (requires recommender)")

	return nil
}

// RegisterGeneralTools registers general-purpose tools that don't require external dependencies.
func RegisterGeneralTools() error {
	// Register HTTP request tool
	httpTool := NewHTTPRequest()
	if err := Register(httpTool); err != nil {
		return fmt.Errorf("failed to register http_request: %w", err)
	}

	// Register calculator tool
	calcTool := NewCalculator()
	if err := Register(calcTool); err != nil {
		return fmt.Errorf("failed to register calculator: %w", err)
	}

	// Register datetime tool
	dtTool := NewDateTime()
	if err := Register(dtTool); err != nil {
		return fmt.Errorf("failed to register datetime: %w", err)
	}

	// Register text processor tool
	tpTool := NewTextProcessor()
	if err := Register(tpTool); err != nil {
		return fmt.Errorf("failed to register text_processor: %w", err)
	}

	slog.Info("General purpose tools registered successfully")
	return nil
}

// RegisterKnowledgeTools registers knowledge base tools with the given service.
func RegisterKnowledgeTools(service interface{}) error {
	// This is a placeholder - knowledge tools need the actual service
	// They should be registered by the application code
	slog.Info("Knowledge tools ready to be registered with service")
	return nil
}

// ToolExecutor provides a high-level interface for executing tools.
type ToolExecutor struct {
	registry *Registry
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		registry: GlobalRegistry,
	}
}

// Execute executes a tool by name.
func (e *ToolExecutor) Execute(ctx context.Context, name string, params map[string]interface{}) (Result, error) {
	return e.registry.Execute(ctx, name, params)
}

// ListTools returns all available tool names.
func (e *ToolExecutor) ListTools() []string {
	return e.registry.List()
}

// GetTool retrieves a tool by name.
func (e *ToolExecutor) GetTool(name string) (Tool, bool) {
	return e.registry.Get(name)
}

// RegisterTool registers a custom tool.
func (e *ToolExecutor) RegisterTool(tool Tool) error {
	return e.registry.Register(tool)
}

// GetToolInfo returns information about a tool.
func (e *ToolExecutor) GetToolInfo(name string) map[string]interface{} {
	tool, exists := e.registry.Get(name)
	if !exists {
		return nil
	}

	return map[string]interface{}{
		"name":        tool.Name(),
		"description": tool.Description(),
		"parameters":  tool.Parameters(),
	}
}

// GetAllToolInfo returns information about all registered tools.
func (e *ToolExecutor) GetAllToolInfo() []map[string]interface{} {
	tools := e.registry.List()
	info := make([]map[string]interface{}, 0, len(tools))

	for _, name := range tools {
		if toolInfo := e.GetToolInfo(name); toolInfo != nil {
			info = append(info, toolInfo)
		}
	}

	return info
}
