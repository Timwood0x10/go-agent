# Simple Agent Framework Example

A basic example demonstrating the GoAgent framework's core functionality with Leader Agent and Sub Agents configuration.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.26+
- **LLM Provider**: Configurable (supports OpenAI, Ollama, OpenRouter)
- **Configuration Format**: YAML
- **Concurrency Control**: errgroup
- **Template Engine**: Go text/template

### Core Components Used

| Component | Purpose | Code Location |
|-----------|---------|---------------|
| **Leader Agent** | Task analysis, config parsing, sub-agent coordination | `internal/agents/leader/` |
| **Sub Agents** | Parallel execution of specific tasks | `internal/agents/sub/` |
| **AHP Protocol** | Inter-agent communication (message queue) | `internal/protocol/ahp/` |
| **LLM Client** | LLM service interaction | `internal/llm/client.go` |
| **Configuration Management** | YAML config file parsing | `internal/config/config.go` |
| **Template Engine** | Prompt template rendering | `internal/llm/template.go` |
| **Memory System** | Session memory management | `internal/memory/context/` |

### Key Feature Implementations

**Code Location References**:
- Configuration loading: `examples/simple/main.go:30-50`
- Leader Agent creation: `examples/simple/main.go:100-120`
- Sub Agents creation: `examples/simple/main.go:125-145`
- Agent startup: `examples/simple/main.go:150-165`
- Graceful shutdown: `examples/simple/main.go:170-185`

## Features

### Configurable Options

Configure via `config/server.yaml`:

1. **LLM Configuration**
   - Provider selection (openai/ollama/openrouter)
   - API key and endpoint
   - Model name and timeout settings

2. **Agent Configuration**
   - Leader Agent parameters (max steps, parallel tasks)
   - Sub Agents list (types, triggers, retry counts)

3. **Prompt Templates**
   - Profile extraction template
   - Recommendation generation template
   - Custom template variables

4. **Output Configuration**
   - Output format (table/json/simple)
   - Summary template

## Quick Start

### 1. Configure LLM

Edit `config/server.yaml`:

```yaml
llm:
  provider: "ollama"  # or "openai", "openrouter"
  base_url: "http://localhost:11434"
  model: "llama3.2"
```

### 2. Run the Example

```bash
cd examples/simple
go run main.go
```

### 3. Example Output

```
=== Processing Sample Request ===
Input: I want to travel to Tokyo, Japan for 5 days and 4 nights, with a budget of 10,000 yuan

[Leader Agent] Parsing profile...
Profile: {"destination": "Tokyo", "duration": "5 days and 4 nights", "budget": 10000}

[Leader Agent] Dispatching tasks to 3 sub-agents...

[Agent: destination] Processing...
[Agent: food] Processing...
[Agent: hotel] Processing...

=== Aggregated Results ===
Destination Recommendations: ...
Food Recommendations: ...
Hotel Recommendations: ...
```

## Architecture

```
User Input
    │
    ▼
┌─────────────────┐
│  Leader Agent   │
│  - Parse config │
│  - Analyze intent│
│  - Dispatch tasks│
└────────┬────────┘
         │
         ├────────────┬────────────┬────────────
         │            │            │
         ▼            ▼            ▼
    ┌────────┐  ┌────────┐  ┌────────┐
    │Agent 1 │  │Agent 2 │  │Agent 3 │
    └────────┘  └────────┘  └────────┘
         │            │            │
         └────────────┴────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │  Result Aggregation │
              └─────────────────┘
```

## Configuration Example

### config/server.yaml

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
  timeout: 60
  max_tokens: 2048

agents:
  leader:
    id: "leader-simple"
    max_steps: 5
    max_parallel_tasks: 3

  sub:
    - id: "agent-destination"
      type: "destination"
      triggers: ["destination", "go", "travel"]
      max_retries: 2

prompts:
  profile_extraction: |
    Extract travel information from user input: {{.input}}
```

## Extending

### Add New Sub Agent

1. Add configuration in `config/server.yaml`:

```yaml
sub:
  - id: "agent-transport"
    type: "transport"
    triggers: ["transport", "flight", "train"]
```

2. Define corresponding prompt template in config file

### Customize Output Format

Modify `output.format` configuration:

```yaml
output:
  format: "json"  # or "table", "simple"
```

## Troubleshooting

### Issue 1: Config File Not Found

```
Error: Failed to load config: no such file
```

**Solution**:
- Check `CONFIG_PATH` environment variable
- Ensure config file is at correct path: `./examples/simple/config/server.yaml`

### Issue 2: LLM Connection Failed

```
Error: Failed to initialize LLM client: connection refused
```

**Solution**:
- Check if LLM service is running
- Verify `base_url` and `model` configuration

### Issue 3: Agent Startup Failed

```
Error: Failed to start sub agent
```

**Solution**:
- Check if sub agents configuration format is correct
- View logs for detailed error information

## References

- [Main README](../../README.md)
- [Quick Start](../../docs/quick_start_en.md)
- [Architecture Documentation](../../docs/arch.md)
- [Agent Documentation](../../docs/agents/)

## License

MIT License

---

**Created**: 2026-03-23  
**Example Type**: Basic Framework Demonstration  
**Code Location**: `examples/simple/main.go:1-409`