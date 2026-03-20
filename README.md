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
- **Vector Storage**: PostgreSQL + pgvector for semantic search and RAG

## System Requirements

### Minimum Requirements
- Go 1.26.1 or higher
- LLM API access (OpenAI, Ollama, or OpenRouter)

### Optional Requirements (for advanced features)
- PostgreSQL 16+ with pgvector extension (for vector storage)
- Redis (for caching)
- golangci-lint (for development)

### Dependencies

The framework uses minimal external dependencies:
- `github.com/fsnotify/fsnotify` - File system watcher
- `github.com/google/uuid` - UUID generation
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/stretchr/testify` - Testing framework
- `golang.org/x/*` - Standard Go extension libraries
- `gopkg.in/yaml.v3` - YAML parsing

No heavy third-party framework dependencies.

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

### Run the Travel Planning Example

```bash
cd /goagent

# Set API key
export OPENROUTER_API_KEY="your-api-key"

# Run
go run ./examples/travel/main.go
```

### Try Knowledge Base Example

```bash
cd goagent

# Start PostgreSQL + pgvector
docker run -d \
  --name postgres-pgvector \
  -p 5433:5432 \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goagent \
  pgvector/pgvector:pg16

# Import a document
cd examples/knowledge-base
go run main.go --save example.md

# Ask questions
go run main.go --chat
```

### Sample Output

Travel Example:
```
=== Request: 我想去日本东京旅游，5天4晚，预算10000元，喜欢美食和购物 ===
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

### Storage Settings (Optional)

```yaml
storage:
  enabled: false            # Enable PostgreSQL storage
  type: "postgres"
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "postgres"
  database: "goagent"
  ssl_mode: "disable"
  pgvector:
    enabled: false          # Enable pgvector for vector search
    dimension: 1536         # Embedding dimension
    table_name: "embeddings"
```

### Memory Settings (Optional)

```yaml
memory:
  enabled: false            # Enable memory system
  session:
    enabled: true
    max_history: 50         # Max conversation turns
  user_profile:
    enabled: false          # Enable persistent user profile
    storage: "memory"       # "memory" or "postgres"
    vector_db: false         # Store profile as vectors
  task_distillation:
    enabled: false          # Enable task distillation
    storage: "memory"       # "memory" or "postgres"
    vector_store: false     # Store distilled results in pgvector
    prompt: "请简洁总结以下任务的关键信息，包括：用户需求、偏好、预算范围。"
```

### Retrieval Strategies (Optional)

The framework provides two retrieval services for different use cases:

| Use Case | Recommended Service | Description |
|----------|---------------------|-------------|
| **Single Knowledge Base Retrieval** (RAG, Q&A, Document Search) | ✅ SimpleRetrievalService | Pure vector similarity search without complex weights. Best for single-source semantic search scenarios. |
| **Exact Match Queries** (e.g., "a = x", configuration lookups) | ✅ SimpleRetrievalService | Precision mode with Exact Match → Keyword → Vector pipeline. Ideal for precise queries requiring deterministic matching. |
| **Multi-Source Fusion Retrieval** (Knowledge + Experience + Tools) | ✅ RetrievalService | Hybrid search with multi-source fusion, query rewriting, and time decay. For complex enterprise systems. |
| **Complex Enterprise Systems** (time decay, weight control) | ✅ RetrievalService | Advanced features including query weights, source weights, time-based scoring, and result reranking. |

**SimpleRetrievalService Features:**
- Pure vector similarity search (1 - cosine_distance)
- Precision mode: Exact Match → Keyword → Vector (early return)
- No complex weight calculations
- No time decay
- No query rewrites
- Simple and effective for single knowledge base scenarios

**RetrievalService Features:**
- Multi-source search (Knowledge + Experience + Tools)
- Query rewriting with weight control (original=1.0, rule=0.7, llm=0.5)
- Source weight configuration
- Time-based score decay
- Result merging and reranking
- Complex enterprise-grade features

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

│   ├── server/          # Main server application

│   ├── migrate_goagent/ # Database migration tool

│   └── setup_test_db/   # Test database setup

├── configs/              # Configuration files

├── docs/                 # Architecture documentation

├── examples/

│   ├── travel/          # Travel planning example

│   ├── simple/           # Simple example

│   ├── knowledge-base/   # Knowledge base example

│   ├── openrouter/       # OpenRouter example

│   └── devagent/         # Development agent

├── internal/

│   ├── agents/

│   │   ├── base/        # Base interfaces

│   │   ├── leader/      # Leader agent

│   │   └── sub/          # Sub agents

│   ├── config/          # Configuration management

│   ├── core/

│   │   ├── errors/       # Error handling

│   │   ├── models/       # Data models

│   │   └── registry/     # Component registry

│   ├── llm/

│   │   └── output/       # LLM adapters

│   ├── memory/           # Memory system

│   ├── observability/    # Logging and tracing

│   ├── protocol/          # AHP protocol

│   ├── ratelimit/        # Rate limiting

│   ├── security/         # Security utilities

│   ├── shutdown/          # Graceful shutdown

│   ├── storage/

│   │   └── postgres/     # PostgreSQL + pgvector

│   ├── tools/            # Tool system

│   └── workflow/         # Workflow engine

├── knowledge/            # Knowledge base data (Python scripts)

├── services/             # Service configurations

│   └── embedding/        # Embedding service

└── pkg/                  # Utilities

```

## Examples

### 1. Travel Planning Agent (`examples/travel/`)
Multi-agent travel assistant demonstrating:
- Profile parsing from natural language
- Dynamic task planning based on triggers
- Parallel sub-agent execution
- Result aggregation

**Run:**
```bash
export OPENROUTER_API_KEY="your-api-key"
go run ./examples/travel/main.go
```

### 2. Knowledge Base (`examples/knowledge-base/`)
Local document retrieval and Q&A system demonstrating:
- Document import with chunking
- Vector similarity search (pgvector)
- Multi-tenant isolation
- Interactive chat interface

**Run:**
```bash
cd examples/knowledge-base
go run main.go --save example.md
go run main.go --chat
```

### 3. Simple Agent (`examples/simple/`)
Basic multi-agent example with fashion recommendations.

**Run:**
```bash
go run ./examples/simple/main.go
```

See individual example READMEs for detailed configuration.

## Development

### Prerequisites
- Go 1.26.1+
- golangci-lint: `brew install golangci-lint`
- staticcheck: `go install honnef.co/go/tools/cmd/staticcheck@latest`
- goimports: `go install golang.org/x/tools/cmd/goimports@latest`

### Commands

```bash
# Install dependencies
make install

# Format code
make fmt

# Run all checks (lint + test)
make check

# Run tests with coverage
make test

# Run tests with race detection
make test-race

# Run linting
make lint

# Run CI checks (install, fmt, lint, test-race)
make ci

# Build binaries
make build

# Build all binaries
make build-all

# Clean build artifacts
make clean

# Show help
make help
```

Run `make check-all` to verify all coverage requirements.

## Contributing

Contributions are welcome! Please follow these guidelines:

1. **Code Style**
   - Run `make fmt` before committing
   - Pass `make lint` checks
   - Add tests for new features

2. **Testing**
   - All tests must pass: `make test`
   - Maintain coverage requirements
   - Add integration tests for new features

3. **Documentation**
   - Update READMEs for new examples
   - Add inline comments for complex logic
   - Update architecture docs for structural changes

4. **Pull Requests**
   - Describe changes in PR description
   - Reference related issues
   - Ensure CI checks pass

## License

MIT License

## Documentation

- [Architecture](docs/arch.md) - System architecture overview
- [Agents](docs/agents/) - Agent design and definitions
- [Core](docs/core/) - Core components (errors, models, registry)
- [Engine](docs/engine/) - Workflow engine design
- [LLM](docs/llm/) - LLM integration and query rewriting
- [Memory](docs/memory/) - Memory system design
- [Protocol](docs/protocol/) - AHP protocol specification
- [Rate Limiting](docs/ratelimit/) - Rate limiting strategies
- [Shutdown](docs/shutdown/) - Graceful shutdown mechanism
- [Storage](docs/storage/) - PostgreSQL storage with pgvector
- [Tools](docs/tools/) - Tool system design
- [Retrieval Strategy](docs/retrieval-strategy.md) - Knowledge retrieval strategies

## License

MIT License