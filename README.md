# GoAgent Framework

A lightweight, highly configurable multi-agent framework for building AI applications in Go.

## What is GoAgent?

GoAgent is a **generic multi-agent framework** that allows users to build AI applications through **configuration only** (YAML). Users only need to:

1. Write a YAML configuration file
2. Write a simple startup script (a few lines of code)
3. The framework handles all the complex logic:

- **Profile Parsing** - Extract user preferences from natural language
- **Dynamic Task Planning** - Automatically split and schedule tasks based on triggers
- **Tool Scheduling** - Unified tool management
- **Result Validation** - Ensure output conforms to expected schema
- **Result Aggregation** - Merge results from multiple agents
- **Memory Distillation** - Auto-extract and summarize key info from conversations
- **Storage** - pgvector vector storage for cross-session persistence

## Features

- **Multi-Agent Architecture**: Leader agent orchestrates multiple sub-agents for parallel task execution
- **AHP Protocol**: Custom Agent Heartbeat Protocol for inter-agent communication
- **Workflow Engine**: Dynamic DAG-based workflow orchestration with hot-reload support
- **LLM Integration**: Unified adapters for OpenAI, Ollama, OpenRouter, and other LLM providers
- **Memory System**: Three-tier memory management (session, user, task) with RAG support
- **Graceful Shutdown**: Five-phase shutdown with callback registration
- **Rate Limiting**: Token bucket, sliding window, and semaphore-based limiting
- **Tool System**: Extensible tool registry for agent capabilities
- **Result Validation**: JSON Schema validation with automatic retry

## Quick Start

### Run the Travel Example

```bash
cd /Users/scc/go/src/styleagent

# Set API key
export OPENROUTER_API_KEY="your-api-key"

# Run
go run ./examples/travel/main.go
```

### Try It

```
=== Request 1: 我想去日本东京旅游，5天4晚，预算10000元，喜欢美食和购物 ===
```

## Configuration Reference

All configuration is in YAML. Here's what you can configure:

### LLM Settings

```yaml
llm:
  provider: "openrouter"      # "openai", "ollama", "openrouter"
  api_key: ""                 # Use env var: OPENROUTER_API_KEY
  base_url: "https://openrouter.ai/api/v1"
  model: "meta-llama/llama-3.1-8b-instruct"
  timeout: 60                 # seconds
  max_tokens: 4096           # max response tokens
```

### Agent Settings

```yaml
agents:
  leader:
    id: "leader-travel"
    max_steps: 10
    max_parallel_tasks: 4
    max_validation_retry: 3
    enable_cache: true

  sub:
    - id: "agent-destination"
      type: "destination"
      category: "destination"
      triggers: ["destination"]    # Keywords to trigger this agent
      max_retries: 3
      timeout: 30
      model: "..."               # Optional: per-agent model
      provider: "..."            # Optional: per-agent provider
```

### Prompt Templates

Customize agent behavior through YAML templates:

```yaml
prompts:
  # Profile extraction - parse user input into structured data
  profile_extraction: |
    你是一位旅行助手。请从用户的输入中提取旅行偏好信息。
    用户输入: {{.input}}
    ...

  # Recommendation - generate recommendations
  recommendation: |
    请根据以下信息推荐 {{.Category}}：
    目的地: {{index . "destination"}}
    预算: {{index . "budget"}}
    ...
```

**Template Variables:**

| Variable | Description |
|----------|-------------|
| `{{.input}}` | Raw user input (profile_extraction) |
| `{{.Category}}` | Agent type (recommendation) |
| `{{index . "key"}}` | Access profile fields |

### Output Settings

```yaml
output:
  format: "table"  # "table", "json", "simple"
  item_template: "{{.Name}} - {{.Price}}"
  summary_template: "Got {{.Count}} items"
```

### Validation Settings

Configure result validation with JSON Schema:

```yaml
validation:
  enabled: true           # Enable/disable validation
  schema_type: "travel"  # "fashion", "travel", "custom"
  retry_on_fail: true    # Retry LLM call when validation fails
  max_retries: 3         # Max retry attempts
  strict_mode: false     # If true, return error on validation failure
```

**Validation Fields by Schema Type:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **Travel Schema (`schema_type: "travel"`)** |
| item_id | string | Yes | Unique identifier |
| name | string | Yes | Item name |
| category | string | Yes | destination/food/hotel/itinerary/transport/activity |
| description | string | No | Item description |
| price | number | No | Price (>= 0) |
| url | string | No | URL (uri format) |
| image_url | string | No | Image URL (uri format) |
| style | array | No | Style tags |
| colors | array | No | Color list |
| match_reason | string | No | Why recommended |
| brand | string | No | Brand name |
| metadata | object | No | Additional metadata |
| **Result Level Fields** |
| session_id | string | No | Session identifier |
| user_id | string | No | User identifier |
| items | array | Yes | Array of items (min 1) |
| reason | string | No | Recommendation reason |
| total_price | number | No | Total price (>= 0) |
| match_score | number | No | Match score (0-1) |
| **Fashion Schema (`schema_type: "fashion"`)** |
| item_id | string | Yes | Unique identifier |
| category | string | Yes | top/bottom/dress/outerwear/shoes/accessory/bag/hat |
| name | string | Yes | Item name |
| brand | string | No | Brand name |
| price | number | Yes | Price (>= 0) |
| url | string | No | URL (uri format) |
| image_url | string | No | Image URL (uri format) |

**Validation Behavior:**
- `retry_on_fail: true` - Automatically retry LLM call when validation fails
- `strict_mode: true` - Return error on validation failure; otherwise log and continue with unvalidated result

### Storage Settings (Future)

```yaml
storage:
  enabled: false
  type: "postgres"
  host: "localhost"
  port: 5432
  pgvector:
    enabled: false
    dimension: 1536
```

### Memory Settings (Future)

```yaml
memory:
  enabled: false
  session:
    enabled: true
    max_history: 50
  user_profile:
    enabled: false
  task_distillation:
    enabled: false
```

## Architecture

```
User Input
    │
    ▼
┌─────────────────┐
│ Leader Agent   │ ── Parse Profile (LLM)
│                │ ── Plan Tasks (trigger-based)
└────────┬────────┘
         │ Parallel dispatch
         ▼
┌────────┴────────┐
│ Sub Agents       │
│ (Parallel)       │
└────────┬────────┘
         │ Results
         ▼
┌─────────────────┐
│ Validation      │ ── JSON Schema Check
│ (Schema)        │ ── Auto-retry on fail (optional)
└────────┬────────┘
         │ Validated
         ▼
┌─────────────────┐
│ Aggregation     │
└─────────────────┘
```

## Project Structure

```
goagent/
├── cmd/                  # Application entry points
├── configs/             # Configuration files
├── docs/                # Architecture documentation
├── examples/
│   ├── travel/          # Travel planning example
│   └── simple/           # Simple example
├── internal/
│   ├── agents/
│   │   ├── base/        # Base interfaces
│   │   ├── leader/      # Leader agent
│   │   └── sub/          # Sub agents
│   ├── core/
│   │   ├── errors/      # Error handling
│   │   └── models/       # Data models
│   ├── llm/
│   │   └── output/       # LLM adapters
│   ├── memory/           # Memory system
│   ├── protocol/         # AHP protocol
│   ├── ratelimit/        # Rate limiting
│   ├── shutdown/         # Graceful shutdown
│   ├── storage/          # PostgreSQL storage
│   ├── tools/            # Tool system
│   └── workflow/         # Workflow engine
└── pkg/                  # Utilities
```

## Examples

See `examples/travel/README.md` for a complete example with detailed configuration.

## Development

```bash
# Run tests
make test

# Run with race detection
make test-race

# Linting
make lint

# Build
make build
```

## Documentation

- [Architecture](docs/arch.md)
- [Agent Definitions](docs/agents/)
- [LLM Integration](docs/llm/)
- [Storage](docs/storage/)
- [Memory](docs/memory/)

## License

MIT License