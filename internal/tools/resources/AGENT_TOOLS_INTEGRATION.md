# Agent Tools Integration Guide

This guide explains how to integrate tools with agents in the GoAgent framework.

## Overview

The GoAgent framework provides a unified interface for agent tool registration and management. Agents can load tools during initialization based on their configuration.

## Tool Categories

Tools are organized into the following categories:

- **CategorySystem**: System-level tools (file operations, ID generation, etc.)
- **CategoryCore**: Core general-purpose tools (HTTP, calculator, datetime, text processor, etc.)
- **CategoryData**: Data processing tools (JSON, validation, etc.)
- **CategoryKnowledge**: Knowledge base tools
- **CategoryMemory**: Memory-related tools
- **CategoryDomain**: Domain-specific tools (fashion, weather, etc.)

## Agent Initialization

### Step 1: Register Builtin Tools

```go
import "goagent/internal/tools/resources"

// Register all builtin tools
if err := resources.RegisterBuiltinToolsForAgent(); err != nil {
    return fmt.Errorf("failed to register builtin tools: %w", err)
}
```

### Step 2: Create Agent Tools Configuration

```go
// Option 1: Use predefined configuration
config := resources.CreateAgentToolConfigs.Leader()

// Option 2: Custom configuration
config := &resources.AgentToolConfig{
    Enabled: []string{
        "calculator",
        "http_request",
    },
    Categories: []resources.ToolCategory{
        resources.CategoryCore,
        resources.CategoryData,
    },
}
```

### Step 3: Create Agent Tools Instance

```go
agentTools := resources.NewAgentTools(config)
agentTools.LogTools("my_agent")
```

### Step 4: Integrate with Agent

```go
type MyAgent struct {
    name  string
    tools *resources.AgentTools
}

func NewMyAgent(name string) (*MyAgent, error) {
    config := resources.CreateAgentToolConfigs.Leader()
    agentTools := resources.NewAgentTools(config)

    return &MyAgent{
        name:  name,
        tools: agentTools,
    }, nil
}
```

## Using Tools in Agent

### Execute a Tool

```go
result, err := agent.tools.Execute(ctx, "calculator", map[string]interface{}{
    "operation": "add",
    "operands":  []interface{}{1, 2, 3},
})

if err != nil {
    return err
}

if !result.Success {
    return fmt.Errorf("tool failed: %s", result.Error)
}

fmt.Println("Result:", result.Data)
```

### Get Tool Schemas

```go
schemas := agent.tools.GetSchemas()
for _, schema := range schemas {
    fmt.Printf("%s (%s): %s\n", schema.Name, schema.Category, schema.Description)
}
```

### Generate Tool Prompt

```go
toolPrompt := agent.tools.GenerateToolPrompt()
systemPrompt := fmt.Sprintf("You are an AI assistant.\n\n%s", toolPrompt)
```

### Export Capabilities

```go
capabilityExport := agent.tools.GetCapabilityExport("my_agent")
// {
//   "agent_name": "my_agent",
//   "tools": ["calculator", "http_request", ...],
//   "categories": ["core", "data"],
//   "tool_count": 5
// }
```

## Predefined Agent Configurations

### Leader Agent (Orchestration Focused)

```go
config := resources.CreateAgentToolConfigs.Leader()
// Categories: Core, Knowledge, Memory
// Focus: Planning, coordination, decision-making
```

### Worker Agent (Task Execution Focused)

```go
config := resources.CreateAgentToolConfigs.Worker()
// Categories: Core, Data, System
// Focus: Task execution, data processing
```

### Research Agent

```go
config := resources.CreateAgentToolConfigs.Research()
// Tools: http_request, knowledge_search, text_processor, json_tools
// Focus: Information gathering and analysis
```

### All Tools

```go
config := resources.CreateAgentToolConfigs.All()
// All tools enabled
```

## Tool Filtering

### By Tool Name

```go
config := &resources.AgentToolConfig{
    Enabled: []string{"calculator", "http_request"},
}
```

### By Category

```go
config := &resources.AgentToolConfig{
    Categories: []resources.ToolCategory{
        resources.CategoryCore,
        resources.CategoryData,
    },
}
```

### Disable Specific Tools

```go
config := &resources.AgentToolConfig{
    Disabled: []string{"file_write", "file_delete"},
}
```

## Multi-Agent Coordination

Agents can export their capabilities for coordination:

```go
leaderExport := leaderAgent.tools.GetCapabilityExport("leader_agent")
workerExport := workerAgent.tools.GetCapabilityExport("worker_agent")

// Use capabilities for task delegation
if containsTool(workerExport, "calculator") {
    delegateTask(workerAgent, "calculate", params)
}
```

## Example: Complete Agent Setup

```go
package main

import (
    "context"
    "fmt"
    "log/slog"

    "goagent/internal/tools/resources"
)

type ResearchAgent struct {
    name  string
    tools *resources.AgentTools
    llm   LLMClient // Your LLM client
}

func NewResearchAgent(name string) (*ResearchAgent, error) {
    // Register builtin tools
    if err := resources.RegisterBuiltinToolsForAgent(); err != nil {
        return nil, err
    }

    // Configure tools for research agent
    config := resources.CreateAgentToolConfigs.Research()
    agentTools := resources.NewAgentTools(config)
    agentTools.LogTools(name)

    // Generate tool prompt
    toolPrompt := agentTools.GenerateToolPrompt()

    // Initialize LLM with tools
    llm := NewLLMClient(WithTools(agentTools.GetSchemas()))

    return &ResearchAgent{
        name:  name,
        tools: agentTools,
        llm:   llm,
    }, nil
}

func (a *ResearchAgent) Process(ctx context.Context, query string) (string, error) {
    // Process query with LLM
    response, err := a.llm.Chat(ctx, query)
    if err != nil {
        return "", err
    }

    // If LLM wants to use tools, execute them
    if response.ToolCall != nil {
        result, err := a.tools.Execute(ctx, response.ToolCall.Name, response.ToolCall.Params)
        if err != nil {
            return "", err
        }

        // Feed result back to LLM
        return a.llm.Chat(ctx, query + "\n\nTool Result: " + fmt.Sprint(result.Data))
    }

    return response.Content, nil
}
```

## Best Practices

1. **Register tools early**: Register builtin tools during agent initialization
2. **Filter appropriately**: Use category or name filtering to give agents only the tools they need
3. **Log tool loading**: Always log which tools are loaded for debugging
4. **Generate prompts**: Include tool descriptions in system prompts for better tool selection
5. **Export capabilities**: Use capability exports for multi-agent coordination

## Available Tools

### Core Tools
- `http_request`: Perform HTTP requests
- `calculator`: Mathematical calculations
- `datetime`: Date and time operations
- `text_processor`: Text processing operations

### Data Tools
- `json_tools`: Parse, extract, merge, and pretty-print JSON
- `data_validation`: Validate JSON, email, URL, or schema

### System Tools
- `file_tools`: Read, write, and list files and directories
- `id_generator`: Generate unique identifiers (UUID or short ID)

### Knowledge Tools
- `knowledge_search`: Search knowledge base
- `knowledge_add`: Add knowledge
- `knowledge_update`: Update knowledge
- `knowledge_delete`: Delete knowledge
- `correct_knowledge`: Correct knowledge

### Memory Tools
- `memory_search`: Search distilled memories
- `user_profile`: Retrieve user profile

### Domain Tools
- `fashion_search`: Search fashion items
- `style_recommend`: Get style recommendations
- `weather_check`: Check weather

## Troubleshooting

### Tool Not Found

If a tool is not found, check:
1. The tool is registered in `RegisterGeneralTools()`
2. The tool is enabled in the agent configuration
3. The tool name matches exactly

### Permission Errors

For file operations, ensure:
1. The file paths are absolute
2. The agent has read/write permissions
3. Directories exist before writing

### Execution Failures

Check:
1. Tool parameters are valid
2. Required parameters are provided
3. Parameter types match the schema