# Multi-Agent Dialog Example

An interactive multi-agent dialog system demonstrating agent communication, tool usage with ACE (Agent Capability Engine), and conversation history tracking.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.21+
- **LLM Provider**: Ollama (llama3.2) or other OpenAI API-compatible services
- **Configuration Format**: YAML
- **ACE Engine**: Agent Capability Engine for tool filtering
- **Concurrency Control**: errgroup
- **Interactive Interface**: Command Line (CLI)

### Core Components Used

| Component | Purpose | Code Location |
|-----------|---------|---------------|
| **DialogAgent** | Dialog agent main implementation | `examples/multi-agent-dialog/main.go:50-100` |
| **CapabilityEngine** | Capability detection and tool filtering | `internal/tools/resources/core/capability.go` |
| **AgentTools** | Tool registration and management | `internal/tools/resources/core/agent_tools.go` |
| **LLM Client** | LLM interaction and tool calls | `internal/llm/client.go` |
| **Tool Interface** | Unified tool interface | `api/core/types.go:Tool` |
| **Built-in Tools** | Built-in tool collection | `internal/tools/resources/builtin/` |

### Supported Tools and Capabilities

| Tool | Capability | Description | Code Location |
|------|------------|-------------|---------------|
| **calculator** | math | Mathematical calculations | `internal/tools/resources/builtin/calculator.go` |
| **datetime** | time | Date/time operations | `internal/tools/resources/builtin/datetime.go` |
| **http_request** | network | HTTP requests | `internal/tools/resources/builtin/http_request.go` |
| **knowledge_search** | knowledge | Knowledge retrieval | `internal/tools/resources/builtin/knowledge_search.go` |
| **file_read** | file | File reading | `internal/tools/resources/builtin/file_tools.go` |
| **memory_read** | memory | Memory reading | `internal/tools/resources/builtin/memory_tools.go` |

### Key Feature Implementations

**Code Location References**:
- Agent creation and initialization: `examples/multi-agent-dialog/main.go:50-80`
- ACE tool filtering: `examples/multi-agent-dialog/main.go:120-150`
- Tool call handling: `examples/multi-agent-dialog/main.go:200-250`
- Multi-turn dialog management: `examples/multi-agent-dialog/main.go:150-180`
- Capability list display: `examples/multi-agent-dialog/main.go:280-310`

## Features

### ACE (Agent Capability Engine)

Intelligent tool filtering system that dynamically selects relevant tools based on user queries:

- **Detect Capabilities**: Identify needed capability types (math, time, network, etc.) from query
- **Filter Tools**: Show LLM only 2-4 relevant tools instead of all tools
- **Improve Accuracy**: Reduce tool selection errors
- **Lower Cost**: Reduce token usage

**Code Location**: `examples/multi-agent-dialog/main.go:120-150`

### Multi-Turn Dialog

Supports up to 3 rounds of tool call loops:

```
User Input → LLM Analyzes → Tool Call → Tool Result → LLM Generates → Answer
         ↑                                              │
         └──────────────── Add History ──────────────────┘
```

**Code Location**: `examples/multi-agent-dialog/main.go:150-180`

### Capability Query

Support natural language queries about agent capabilities:

```
User: What can you do?
Agent: I can help you with:
  - Math calculations (calculator)
  - Time queries (datetime)
  - HTTP requests (http_request)
  ...
```

**Code Location**: `examples/multi-agent-dialog/main.go:280-310`

## Quick Start

### 1. Configure LLM

Edit `config/server.yaml`:

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
  timeout: 60
  max_tokens: 2048
```

### 2. Run the Example

```bash
cd examples/multi-agent-dialog
go run main.go
```

### 3. Interactive Example

```
=== Multi-Agent Dialog System ===
Available commands:
  - Type your question
  - 'capabilities' - Show all available tools
  - 'exit' - Quit

> What can you do?
I can help you with:
  - Math calculations (calculator)
  - Time queries (datetime)
  - HTTP requests (http_request)
  - Knowledge retrieval (knowledge_search)
  - File operations (file_read)
  - Memory operations (memory_read)

> Calculate sum from 1 to 100
[TOOL:calculator {"expression":"1+2+...+100"}]
The result is: 5050

> What time is it?
[TOOL:datetime {"operation":"current_time"}]
Current time: 2026-03-23 14:30:45

> Request Baidu homepage
[TOOL:http_request {"url":"https://www.baidu.com"}]
HTTP request successful, status code: 200
Response size: 15,234 bytes
```

## Configuration

### config/server.yaml

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
  timeout: 60
  max_tokens: 2048

tools:
  enabled: true
  capabilities:
    - math
    - time
    - network
    - knowledge
    - file
    - memory
```

## Architecture

```
User Input
    │
    ▼
┌─────────────────────┐
│  Capability Query  │
│  Detection          │
└────────┬────────────┘
         │
    ┌────┴────┐
    │ Yes     │ No
    ▼         ▼
┌────────┐ ┌─────────────────────┐
│Show    │ │ ACE Capability      │
│List    │ │ Detection           │
└────────┘ │ - Analyze query     │
           │ - Match capability   │
           └────────┬────────────┘
                    │
                    ▼
           ┌─────────────────────┐
           │  LLM Generate       │
           │  Response           │
           │  - Use tool schemas │
           │  - Generate calls   │
           └────────┬────────────┘
                    │
                    ▼
           ┌─────────────────────┐
           │  Tool Call Handling │
           │  - Parse [TOOL:]    │
           │  - Execute tools    │
           │  - Return results   │
           └────────┬────────────┘
                    │
                    ▼
           ┌─────────────────────┐
           │  Generate Final     │
           │  Answer             │
           └─────────────────────┘
```

## ACE Workflow

```
User Query: "Calculate 1+1 and tell me the time"
    │
    ▼
Capability Detection: [math, time]
    │
    ▼
Tool Filtering: [calculator, datetime]
    │
    ▼
LLM Prompt: "Available tools: calculator, datetime..."
    │
    ▼
Tool Calls: [TOOL:calculator, TOOL:datetime]
    │
    ▼
Result Integration: "1+1=2, current time is 14:30"
```

**Code Location**: `examples/multi-agent-dialog/main.go:120-250`

## Extending

### Add New Tool

1. Implement tool interface:

```go
type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Execute(ctx, params) (Result, error) {
    // Implementation
}
```

2. Register tool:

```go
agentTools.RegisterTool(&MyTool{})
```

### Customize Capability Mapping

Modify capability keywords in `examples/multi-agent-dialog/main.go`:

```go
capabilities := map[string]string{
    "math": "calculate,sum,multiply,divide",
    "time": "time,date,current time",
    // Add more
}
```

## Troubleshooting

### Issue 1: Tool Not Recognized

```
No tools available for query
```

**Solution**:
- Check if query contains capability keywords
- View capability detection results in logs
- Try more explicit query

### Issue 2: Tool Call Failed

```
Error: tool execution failed
```

**Solution**:
- Check if tool parameters are correct
- View detailed error logs for tool execution
- Verify tool-dependent services are available

### Issue 3: LLM Not Calling Tools

```
LLM response doesn't contain [TOOL:] marker
```

**Solution**:
- Check if LLM model supports tool calls
- Adjust prompt template
- Try different LLM model

## References

- [Main README](../../README.md)
- [ACE Documentation](../../plan/CapabilityLayer.md)
- [Tools System Documentation](../../docs/tools/)
- [LLM Documentation](../../docs/llm/)

## License

MIT License

---

**Created**: 2026-03-23  
**Example Type**: Multi-Agent Dialog Demonstration  
**Code Location**: `examples/multi-agent-dialog/main.go:1-358`