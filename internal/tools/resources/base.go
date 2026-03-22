package resources

import (
	"context"
)

// Tool represents an executable tool.
type Tool interface {
	// Name returns the tool name.
	Name() string
	// Description returns the tool description.
	Description() string
	// Execute executes the tool with given parameters.
	Execute(ctx context.Context, params map[string]interface{}) (Result, error)
	// Parameters returns the parameter schema.
	Parameters() *ParameterSchema
}

// ParameterSchema defines tool parameters.
type ParameterSchema struct {
	Type       string                `json:"type"`
	Properties map[string]*Parameter `json:"properties"`
	Required   []string              `json:"required"`
}

// GetType returns the parameter type.
func (p *ParameterSchema) GetType() string {
	return p.Type
}

// GetProperties returns the parameter properties.
func (p *ParameterSchema) GetProperties() map[string]*Parameter {
	if p == nil {
		return nil
	}
	return p.Properties
}

// GetRequired returns the required parameters.
func (p *ParameterSchema) GetRequired() []string {
	if p == nil {
		return nil
	}
	return p.Required
}

// Parameter defines a single parameter.
type Parameter struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Min         *float64      `json:"min,omitempty"`
	Max         *float64      `json:"max,omitempty"`
}

// BaseTool provides common tool functionality.
type BaseTool struct {
	name        string
	description string
	parameters  *ParameterSchema
}

// NewBaseTool creates a new BaseTool.
func NewBaseTool(name, description string, params *ParameterSchema) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		parameters:  params,
	}
}

// Name returns the tool name.
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *BaseTool) Description() string {
	return t.description
}

// Parameters returns the parameter schema.
func (t *BaseTool) Parameters() *ParameterSchema {
	return t.parameters
}

// ToolFunc is a function-based tool.
type ToolFunc struct {
	BaseTool
	fn func(ctx context.Context, params map[string]interface{}) (Result, error)
}

// NewToolFunc creates a new ToolFunc.
func NewToolFunc(name, description string, params *ParameterSchema, fn func(ctx context.Context, params map[string]interface{}) (Result, error)) *ToolFunc {
	return &ToolFunc{
		BaseTool: *NewBaseTool(name, description, params),
		fn:       fn,
	}
}

// Execute executes the tool function.
func (t *ToolFunc) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	return t.fn(ctx, params)
}

// ToolMetadata holds additional tool metadata.
type ToolMetadata struct {
	Version     string
	Author      string
	Tags        []string
	Examples    []string
	Deprecated  bool
	Deprecation string
}

// WithMetadata adds metadata to a tool.
func WithMetadata(tool Tool, metadata ToolMetadata) Tool {
	return &metadataTool{
		Tool:     tool,
		Metadata: metadata,
	}
}

// metadataTool wraps a tool with metadata.
type metadataTool struct {
	Tool
	Metadata ToolMetadata
}

// IsDeprecated returns true if tool is deprecated.
func (m *metadataTool) IsDeprecated() bool {
	return m.Metadata.Deprecated
}
