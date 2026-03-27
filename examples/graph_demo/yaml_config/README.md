# YAML Configuration for Graph System

This directory demonstrates how to use YAML configuration files to define and execute graph workflows.

## Overview

The YAML configuration system allows you to define complex graph workflows without writing code. You can specify:

- Graph structure (nodes and edges)
- Node types (function, agent, tool)
- Execution flow
- Conditional branching

## Quick Start

### 1. Create a YAML Configuration

Create a `workflow.yaml` file:

```yaml
graph:
  id: "my-workflow"
  start_node: "validate"

  nodes:
    - id: "validate"
      type: "function"
      description: "Validate input data"

    - id: "process"
      type: "function"
      description: "Process the data"

    - id: "save"
      type: "function"
      description: "Save results"

  edges:
    - from: "validate"
      to: "process"

    - from: "process"
      to: "save"
```

### 2. Run the Workflow

```bash
go run yaml_example.go workflow.yaml
```

## Configuration Reference

### Graph Structure

```yaml
graph:
  id: "workflow-id"           # Required: unique graph identifier
  start_node: "node-id"       # Required: entry point node
  nodes: [...]                # Required: list of nodes
  edges: [...]                # Required: list of edges
  agents: [...]               # Optional: list of agents
```

### Node Types

#### Function Node

```yaml
- id: "node-id"
  type: "function"
  description: "Node description"
```

Function nodes execute a simple function that logs execution and stores status in the state.

#### Agent Node

```yaml
- id: "node-id"
  type: "agent"
  description: "Agent node"
  config:
    agent_id: "agent-id"  # ID of registered agent
```

Agent nodes wrap a registered agent and execute its `Process` method.

#### Tool Node

```yaml
- id: "node-id"
  type: "tool"
  description: "Tool node"
  config:
    tool_id: "tool-id"  # ID of registered tool
```

Tool nodes wrap a registered tool for execution.

### Edges

```yaml
- from: "source-node"
  to: "target-node"
  condition: "condition-expression"  # Optional
```

Edges define the execution flow between nodes. Conditional edges allow dynamic routing based on state.

### Agents

```yaml
agents:
  - id: "agent-id"
    type: "agent-type"
    name: "Agent Name"
    config: {}  # Agent-specific configuration
```

## Examples

### Simple Linear Workflow

See `simple_workflow.yaml` for a basic example:

```bash
go run yaml_example.go simple_workflow.yaml
```

### Conditional Workflow

See `conditional_workflow.yaml` for branching logic:

```bash
go run yaml_example.go conditional_workflow.yaml
```

## Advanced Usage

### Registering Agents

For agent nodes, you need to register agents before building the graph:

```go
builder := graph.NewGraphBuilder()
builder.RegisterAgent(myAgent)

config, _ := graph.ParseGraphConfig(yamlData)
g, _ := builder.Build(config)
```

### Registering Tools

For tool nodes, register tools similarly:

```go
builder.RegisterTool("my-tool", myTool)
```

### Custom Service Configuration

```go
service, _ := graph.NewService(&graph.Config{
    RequestTimeout: 30 * time.Second,
    Tracer:         observability.NewLogTracer(nil),
})
```

## Validation

The configuration parser validates:

- Graph ID is present and non-empty
- Start node exists
- All nodes have unique IDs
- All edges reference valid nodes
- Node types are valid (function, agent, tool)

Invalid configurations will produce an error before execution.

## State Management

Nodes can read and write to the shared state:

```go
// Read from state
value, exists := state.Get("key")

// Write to state
state.Set("key", value)
```

Function nodes automatically store execution status:

```yaml
node.<node-id>.timestamp: "executed"
node.<node-id>.status: "success"
```

## Error Handling

If node execution fails, the error is propagated and the graph execution stops. Check the error response for details:

```go
response, err := service.Execute(ctx, g, request)
if err != nil {
    log.Fatalf("Execution failed: %v", err)
}
```

## TODO

- Implement condition parsing from string expressions
- Implement ToolNode with full tool interface
- Add support for custom function node configurations
- Add validation for agent and tool configurations

## Best Practices

1. Use descriptive node IDs
2. Add descriptions for complex nodes
3. Keep YAML files organized with comments
4. Test configurations with simple cases first
5. Validate configurations before deployment

## Troubleshooting

### "agent not registered"

Ensure you've registered all agents referenced in the configuration before building the graph.

### "tool not registered"

Ensure you've registered all tools referenced in the configuration.

### "node type not supported"

Check that node type is one of: `function`, `agent`, `tool`.

### "start node does not exist"

Ensure the `start_node` ID matches an existing node ID.

## See Also

- [Graph System README](../README.md)
- [API Documentation](../../../api/service/graph/)
- [Core Graph Implementation](../../../internal/workflow/graph/)