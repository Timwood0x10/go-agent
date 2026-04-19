# Capability Demo - Agent Capability Engine (ACE)

This example demonstrates the **Agent Capability Engine (ACE)** workflow, which provides a structured approach to tool selection in agent systems.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.26+
- **LLM Provider**: Ollama (llama3.2) or other OpenAI API-compatible services
- **Configuration Format**: YAML
- **Capability Detection**: Keyword matching + LLM intent analysis

### Core Components Used

| Component | Purpose | Code Location |
|-----------|---------|---------------|
| **CapabilityEngine** | Capability detection and tool matching | `internal/tools/resources/core/capability.go` |
| **AgentTools** | Tool registration and management | `internal/tools/resources/core/agent_tools.go` |
| **Tool Interface** | Unified tool interface | `api/core/types.go:Tool` |
| **Built-in Tools** | Built-in tool collection | `internal/tools/resources/builtin/` |
| **LLM Client** | Intent analysis | `internal/llm/client.go` |

### Supported Capability Types

| Capability | Description | Keywords | Corresponding Tools |
|------------|-------------|----------|-------------------|
| **math** | Mathematical calculations | calculate, sum, multiply, divide, compute | calculator |
| **knowledge** | Knowledge retrieval | what, who, explain, search, find | knowledge_search |
| **memory** | Memory access/storage | remember, store, recall, history | memory_read, memory_write |
| **text** | Text processing | parse, format, validate, transform | text_parse, text_format |
| **network** | Network/API requests | api, request, fetch, http, url | http_request |
| **time** | Date/time operations | time, date, schedule, timestamp | datetime |
| **file** | File system operations | file, read, write, delete, list | file_read, file_write |
| **external** | External system interaction | execute, run, command, script | shell_execute |

### Key Feature Implementations

**Code Location References**:
- Capability definitions: `internal/tools/resources/core/capability.go:20-50`
- Tool-to-capability mapping: `internal/tools/resources/core/agent_tools.go:80-120`
- Built-in tool implementations: `internal/tools/resources/builtin/calculator.go`
- LLM intent analysis: `examples/capability-demo/main.go:100-150`
- Tool filtering and ranking: `internal/tools/resources/core/agent_tools.go:200-250`

## ACE Workflow

```
User Query → LLM analyzes intent → Identify Capability → Match Tools → Execute → Return Result
```

### The Problem

Without ACE:
- LLM sees all available tools (e.g., 12 tools)
- Tool selection becomes unstable and inaccurate
- Higher token usage and slower responses

With ACE:
- LLM sees only relevant tools (2-4 tools)
- Better tool selection accuracy
- Reduced token usage and faster responses

## Key Concepts

### Capabilities

Capabilities are high-level abstractions that tools provide. The system supports 8 core capabilities:

| Capability | Description | Keywords |
|------------|-------------|----------|
| `math` | Mathematical calculations | calculate, sum, multiply, divide, compute |
| `knowledge` | Knowledge retrieval | what, who, explain, search, find |
| `memory` | Memory access/storage | remember, store, recall, history |
| `text` | Text processing | parse, format, validate, transform |
| `network` | Network/API requests | api, request, fetch, http, url |
| `time` | Date/time operations | time, date, schedule, timestamp |
| `file` | File system operations | file, read, write, delete, list |
| `external` | External system interaction | execute, run, command, script |

### ACE Components

1. **Capability Detection**: Analyzes query to identify needed capabilities
2. **Tool Filtering**: Returns tools matching detected capabilities
3. **Tool Ranking**: Prioritizes relevant tools for LLM

## Running the Demo

### Prerequisites

- Go 1.21+
- Ollama with llama3.2 model (or modify config for other providers)

### Start the Demo

```bash
cd examples/capability-demo
go run main.go
```

### Interactive Commands

```bash
# Show all available capabilities and tools
capabilities

# Analyze ACE workflow for a specific query
analyze Calculate 1 to 100 sum

# Query the agent
Calculate 1 to 100 sum
What time is it?
Search for information
```

## Example Interactions

### Math Capability

```
> Calculate 1 to 100 sum
2026/03/22 22:19:54 INFO ACE: Capabilities detected query="Calculate 1 to 100 sum" capabilities=[math]
2026/03/22 22:19:54 INFO ACE: Tools matched count=1 tools=[calculator]
[Tools Specialist]: 最终结果是：5,050
```

### Time Capability

```
> What time is it?
2026/03/22 22:19:56 INFO ACE: Capabilities detected query="What time is it?" capabilities=[time]
2026/03/22 22:19:56 INFO ACE: Tools matched count=1 tools=[datetime]
[Tools Specialist]: Current time: 2026-03-22 22:19:56
```

### Multiple Capabilities

```
> Send HTTP request and calculate response time
2026/03/22 22:20:00 INFO ACE: Capabilities detected query="Send HTTP request and calculate response time" capabilities=[network math]
2026/03/22 22:20:00 INFO ACE: Tools matched count=2 tools=[http_request calculator]
[Tools Specialist]: [TOOL:http_request {"url": "https://api.example.com"}]... [TOOL:calculator {...}]
```

## Architecture

```
+----------------+
|   User Query   |
+--------+-------+
         |
         v
+-----------------------+
|  LLM Intent Analysis  |
+----------+------------+
           |
           v
+---------------------+
| Capability Detection|
| - keyword matching  |
+----------+----------+
           |
           v
+---------------------+
|   Tool Filtering    |
| - cap → tool map    |
+----------+----------+
           |
           v
+---------------------+
|   Tool Ranking      |
| - relevance score   |
+----------+----------+
           |
           v
+---------------------+
|  LLM with 2-4 tools |
+----------+----------+
           |
           v
+---------------------+
|  Tool Execution     |
+---------------------+
```

## Expected Output

When running `go run main.go`, you should see:

```
=== Capability Demo Agent ===
This demo shows the ACE workflow:
  1. Query → LLM analyzes intent
  2. Intent → Detect Capability
  3. Capability → Match Tools (2-4 tools)
  4. Tools → Execute → Return Result

Commands:
  capabilities - Show all capabilities and tools
  analyze <query> - Show ACE workflow analysis for a query
  exit - Quit

Try queries like:
  - 'Calculate 1 to 100 sum' (math capability)
  - 'What time is it?' (time capability)
  - 'Search for information' (knowledge capability)
  - 'Send HTTP request' (network capability)

Start... (type 'exit' to quit)
> calculate 1 to 1000 sum
2026/03/22 22:30:00 INFO ACE: Capabilities detected query="calculate 1 to 1000 sum" capabilities=[math]
2026/03/22 22:30:00 INFO ACE: Tools matched count=1 tools=[calculator]
[Capability Demo Agent]: The sum of numbers from 1 to 1000 is 500,500. (2.1s)
```

## Key Features

1. **Automatic Capability Detection**: Keywords in queries are matched to capabilities
2. **Dynamic Tool Filtering**: Only relevant tools are shown to LLM
3. **Reduced Token Usage**: 2-4 tools instead of all tools
4. **Better Accuracy**: Focused tool selection improves reliability
5. **Extensible**: Easy to add new capabilities and tools

## Code Structure

```
examples/capability-demo/
├── main.go           # Demo agent implementation
└── config/
    └── server.yaml   # Configuration file
```

## Implementation Details

### Tool Interface

Tools implement the `core.Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Category() ToolCategory
    Capabilities() []Capability  // Key method for ACE
    Execute(ctx, params) (Result, error)
    Parameters() *ParameterSchema
}
```

### Capability Engine

The `CapabilityEngine` provides:

- `Detect(query) []Capability`: Identify capabilities from query
- `ToolsFor(cap) []Tool`: Get tools for a capability
- `Match(query) []Tool`: Full workflow (detect + filter)

### Agent Integration

Agents use ACE through `AgentTools`:

```go
// Detect capabilities
capabilities := agentTools.DetectCapabilities(query)

// Match tools
tools := agentTools.MatchToolsByQuery(query)

// Get tool schemas for LLM
schemas := agentTools.MatchToolSchemasByQuery(query)
```

## Benefits of ACE

1. **Stability**: Consistent tool selection across queries
2. **Efficiency**: Reduced token usage and faster responses
3. **Accuracy**: Better tool matching and execution
4. **Maintainability**: Clear separation of concerns
5. **Scalability**: Easy to add new tools and capabilities

## References

- Design Document: `/plan/CapabilityLayer.md`
- Code Rules: `/plan/code_rules.md`
- Core Implementation: `/internal/tools/resources/core/capability.go`
- Tool Implementation: `/internal/tools/resources/builtin/`