# GoAgent Architecture Design

**Last Updated**: 2026-03-24

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Input                                    │
│                    "Help me plan a trip..."                            │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Leader Agent                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                │
│  │ Parse Input  │  │  Task Plan   │  │  Aggregate   │                │
│  │   (LLM)      │  │  (LLM)       │  │   Results    │                │
│  └──────────────┘  └──────────────┘  └──────────────┘                │
│         │                 │                 │                           │
│         └─────────────────┼─────────────────┘                           │
│                           │                                              │
│                    dispatch tasks (parallel)                            │
└───────────────────────────┬─────────────────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  Agent-destination │ │ Agent-food    │ │  Agent-hotel    │
│  (Sub Agent)     │ │  (Sub Agent)   │ │  (Sub Agent)    │
└────────┬────────┘ └────────┬────────┘ └────────┬────────┘
         │                  │                  │
         └──────────────────┼──────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Protocol Layer                                     │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  AHP Protocol (Agent Handshake Protocol)                       │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐                          │    │
│  │  │  TASK   │ │ RESULT  │ │ PROGRESS│                          │    │
│  │  └─────────┘ └─────────┘ └─────────┘                          │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  Message Types: TASK | RESULT | PROGRESS | ACK                          │
└─────────────────────────────────────────────────────────────────────────┘
│
│         ┌──────────────────┼──────────────────┐
│         │                  │                  │
│         ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│     Tools       │ │     LLM         │ │    Storage      │
│                 │ │                  │ │                 │
│ • calculator    │ │ • Ollama        │ │  • PostgreSQL   │
│ • datetime      │ │ • OpenRouter    │ │  • pgvector     │
│ • http_request  │ │                  │ │                 │
└─────────────────┘ └─────────────────┘ └────────┬────────┘
         │                                    │
         │                                    ▼
         │                        ┌─────────────────────────┐
         │                        │  PostgreSQL + pgvector   │
         │                        │  ┌───────────────────┐  │
         │                        │  │ • Connections    │  │
         │                        │  │   Pool (25/10)   │  │
         │                        │  │ • Transactions  │  │
         │                        │  └───────────────────┘  │
         │                        │  ┌───────────────────┐  │
         │                        │  │ • Vector Index   │  │
         │                        │  │   (ivfflat)      │  │
         │                        │  │ • Cosine Search  │  │
         │                        │  └───────────────────┘  │
         │                        └─────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Memory System                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                    │
│  │ Session     │  │ User Memory  │  │ Task Memory  │                    │
│  │  (Short-term)│  │  (Long-term) │  │  (Distilled) │                    │
│  └─────────────┘  └─────────────┘  └─────────────┘                    │
│                    │                                                 │
│  ProductionMemoryManager (PG + pgvector persistence)                    │
└─────────────────────────────────────────────────────────────────────────┘
```

**Code Locations**:
- Leader Agent: `internal/agents/leader/agent.go`
- Sub Agent: `internal/agents/sub/agent.go`
- Protocol: `internal/protocol/ahp/`
- LLM Client: `internal/llm/client.go`
- Storage Pool: `internal/storage/postgres/pool.go`
- Memory Manager: `internal/memory/production_manager.go`

---

## Core Component Implementation

### Leader Agent

Leader Agent is responsible for task decomposition, distribution, and result aggregation.

**Code Location**: `internal/agents/leader/agent.go`

```go
type LeaderAgent struct {
    id               string
    maxSteps         int
    maxParallelTasks int
    subAgents        map[string]*SubAgent
    llmClient        *llm.Client
}

func (l *LeaderAgent) Process(ctx context.Context, input string) (string, error) {
    // 1. Parse user input
    parsed, err := l.parseInput(ctx, input)
    if err != nil {
        return "", err
    }

    // 2. Generate task plan
    tasks, err := l.planTasks(ctx, parsed)
    if err != nil {
        return "", err
    }

    // 3. Execute tasks in parallel
    results, err := l.executeTasks(ctx, tasks)
    if err != nil {
        return "", err
    }

    // 4. Aggregate results
    return l.aggregateResults(ctx, results)
}
```

### Sub Agent

Sub Agent is responsible for executing specific tasks.

**Code Location**: `internal/agents/sub/agent.go`

```go
type SubAgent struct {
    id        string
    agentType string
    triggers  []string
    llmClient *llm.Client
    tools     []Tool
}

func (s *SubAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 1. Check trigger conditions
    if !s.shouldExecute(task) {
        return nil, nil
    }

    // 2. Execute tools
    toolResults, err := s.executeTools(ctx, task)
    if err != nil {
        return nil, err
    }

    // 3. LLM generates response
    response, err := s.llmClient.Generate(ctx, s.buildPrompt(task, toolResults))
    if err != nil {
        return nil, err
    }

    return &TaskResult{
        AgentID: s.id,
        Result:  response,
    }, nil
}
```

### LLM Client

Unified client supporting multiple LLM providers.

**Code Location**: `internal/llm/client.go`

```go
type Client struct {
    config     *Config
    httpClient *http.Client
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
    switch ProviderType(c.config.Provider) {
    case ProviderOpenRouter:
        return c.generateOpenRouter(ctx, prompt)
    case ProviderOllama:
        return c.generateOllama(ctx, prompt)
    default:
        return "", fmt.Errorf("unsupported provider: %s", c.config.Provider)
    }
}
```

### Storage Pool

PostgreSQL connection pool implementing "Get-Use-Release" pattern.

**Code Location**: `internal/storage/postgres/pool.go`

```go
type Pool struct {
    cfg          *Config
    db           *sql.DB
    mu           sync.RWMutex
    openCount    int
    idleCount    int
    waitCount    int
    waitDuration time.Duration
}

func (p *Pool) WithConnection(ctx context.Context, fn func(*sql.Conn) error) error {
    conn, err := p.Get(ctx)
    if err != nil {
        return err
    }
    defer p.Release(conn)

    return fn(conn)
}
```

---

## Tech Stack

| Layer | Tech Stack | Code Location |
|-------|-----------|---------------|
| Language | Go 1.21+ | - |
| LLM | Ollama / OpenRouter | `internal/llm/client.go` |
| Protocol | AHP (Agent Handshake Protocol) | `internal/protocol/ahp/` |
| Storage | PostgreSQL 15+ with pgvector | `internal/storage/postgres/` |
| Concurrency | errgroup, sync | - |
| Tools | Built-in tools | `internal/tools/` |
| Embedding | FastAPI + Ollama/SentenceTransformers | `services/embedding/` |

---

## Message Format (AHP Protocol)

**Code Location**: `internal/protocol/ahp/message.go`

```go
type Message struct {
    MessageID  string    `json:"message_id"`
    Method     Method    `json:"method"`     // TASK, RESULT, PROGRESS, ACK
    AgentID    string    `json:"agent_id"`
    TargetID   string    `json:"target_id"`
    TaskID     string    `json:"task_id"`
    SessionID  string    `json:"session_id"`
    Payload    []byte    `json:"payload"`
    Timestamp  time.Time `json:"timestamp"`
}

type Method string

const (
    MethodTask     Method = "TASK"
    MethodResult   Method = "RESULT"
    MethodProgress Method = "PROGRESS"
    MethodAck      Method = "ACK"
)
```

---

## Directory Structure

```
goagent/
├── internal/                # Core implementation
│   ├── agents/              # Agent system
│   │   ├── base/            # Agent base interfaces
│   │   ├── leader/          # Leader Agent
│   │   └── sub/             # Sub Agent
│   ├── protocol/            # AHP protocol
│   │   └── ahp/             # Protocol implementation
│   ├── storage/             # Storage layer
│   │   └── postgres/        # PostgreSQL + pgvector
│   │       ├── pool.go      # Connection pool
│   │       ├── repositories/ # Data repositories
│   │       └── migrations/   # Database migrations
│   ├── memory/              # Memory system
│   │   └── production_manager.go
│   ├── llm/                 # LLM client
│   │   └── client.go
│   ├── tools/               # Tool system
│   │   └── resources/
│   ├── core/                # Core types
│   │   ├── errors/          # Error definitions
│   │   └── types.go
│   ├── config/              # Configuration management
│   ├── workflow/            # Workflow engine
│   ├── ratelimit/           # Rate limiting
│   ├── shutdown/            # Graceful shutdown
│   └── observability/       # Observability
├── api/                     # API layer
│   ├── service/             # Service interfaces
│   │   ├── agent/           # Agent service
│   │   ├── llm/             # LLM service
│   │   ├── memory/          # Memory service
│   │   └── retrieval/       # Retrieval service
│   └── client/              # Client
├── examples/                # Example applications
│   ├── travel/              # Travel planning
│   ├── knowledge-base/      # Knowledge base Q&A
│   ├── simple/              # Simple example
│   └── capability-demo/     # Feature demonstration
├── services/                # Standalone services
│   └── embedding/           # Embedding service
│       ├── app.py           # FastAPI service
│       └── config.py
├── cmd/                     # Command line tools
│   └── server/              # Server startup
├── docs/                    # Documentation
└── configs/                 # Configuration files
```

**Code Location**: Project root directory

---

## Key Design Points

| Feature | Implementation |
|---------|----------------|
| **Concurrency Model** | Worker pool dispatches tasks to multiple Sub Agents |
| **Communication Protocol** | In-Memory Message Queue + AHP custom protocol |
| **State Management** | SessionMemory short-term + TaskMemory distillation |
| **Fault Tolerance** | DLQ stores failed messages, supports retry |
| **Task Coordination** | Phase 1 (parallel) → Phase 2 (dependency-aware) |
| **Scalability** | Dynamic registration of new Agent types |
| **Agent Definition** | Markdown file configuration, supports hot reload |
| **Workflow Orchestration** | YAML/JSON DSL, user-defined workflows |
| **LLM Output** | Four-layer guarantee mechanism ensures output consistency |

---

## Agent Definition (Markdown Configuration)

Agents are defined using Markdown files, allowing non-developers to adjust Agent behavior by editing configuration files.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Agent Definition (Markdown)                         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ # agent_top.md                                               │
├─────────────────────────────────────────────────────────────┤
│ ## Metadata                                                 │
│ name: agent_top                                             │
│ version: 1.0.0                                             │
│                                                             │
│ ## Role                                                     │
│ You are a professional fashion consultant, specializing    │
│ in top wear recommendations.                                │
│                                                             │
│ ## Profile                                                  │
│ expertise: top wear                                         │
│ category: tops                                              │
│ style_tags: [casual, formal, street]                        │
│                                                             │
│ ## Tools                                                    │
│ - fashion_search                                            │
│ - weather_check                                             │
│ - style_recomm                                              │
│                                                             │
│ ## Instructions                                             │
│ 1. Recommend suitable styles based on user preferences      │
│ 2. Consider local weather conditions                       │
│ 3. Match user budget range                                 │
│                                                             │
│ ## Output Format                                            │
│ ```json                                                     │
│ { "items": [...], "reason": "..." }                        │
│ ```                                                         │
└─────────────────────────────────────────────────────────────┘
```

### Built-in Variables

| Variable | Description |
|----------|-------------|
| {{.UserProfile}} | User profile |
| {{.SessionID}} | Session ID |
| {{.Context}} | Context information |
| {{.Input}} | User input |
| {{.Results}} | Upstream results |

---

## Workflow Engine (Workflow Orchestration)

Users can customize workflows through YAML/JSON files to achieve flexible Agent orchestration.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Workflow Engine                                     │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  workflow.yaml                                                       │
├─────────────────────────────────────────────────────────────────────┤
│  name: "Travel Planning Workflow"                                  │
│                                                                     │
│  agents:                                                            │
│    - id: leader                                                     │
│      type: leader                                                  │
│      prompt_file: ./agents/agent_leader.md                         │
│                                                                     │
│    - id: agent_destination                                         │
│      type: sub                                                     │
│      prompt_file: ./agents/agent_destination.md                    │
│      depends_on: [leader]                                          │
│                                                                     │
│    - id: agent_hotel                                               │
│      type: sub                                                     │
│      prompt_file: ./agents/agent_hotel.md                          │
│      depends_on: [leader, agent_destination]                       │
│                                                                     │
│  execution:                                                         │
│    phase1: [agent_destination, agent_food]                        │
│    phase2: [agent_hotel]                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
workflows/
├── default.yaml          # Default workflow
├── summer.yaml           # Summer recommendation
├── winter.yaml           # Winter recommendation
│
├── agents/               # Agent Markdown definitions
│   ├── agent_leader.md
│   ├── agent_destination.md
│   ├── agent_food.md
│   ├── agent_hotel.md
│   └── agent_accessory.md
│
└── templates/           # Templates
```

---

## LLM Output Standardization

Multi-LLM output is ensured through a four-layer guarantee mechanism.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LLM Output Standardization                            │
└─────────────────────────────────────────────────────────────────────────┘

Layer 1: Prompt Template
┌────────────────────────────────────────────────────────────────┐
│ {{.Instructions}}                                                │
│ Output Format:                                                   │
│ ```json                                                          │
│ { "items": [...], "reason": "..." }                            │
│ ```                                                              │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
Layer 2: JSON Schema / Tool Calling
┌────────────────────────────────────────────────────────────────┐
│ {                                                                │
│   "type": "object",                                             │
│   "properties": {                                                │
│     "items": { "type": "array" },                              │
│     "reason": { "type": "string" }                             │
│   }                                                              │
│ }                                                                │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
Layer 3: Output Parser & Validator
┌────────────────────┐    ┌────────────────────┐
│  Parser (Parse)   │───▶│  Validator (Validate)  │
│  - Extract JSON   │    │  - Schema validation   │
│  - Fix broken JSON│    │  - Auto retry       │
└────────────────────┘    └────────────────────┘
                              │
                              ▼
Layer 4: LLM Adapter Layer
┌────────────────────────────────────────────────────────────────┐
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                        │
│  │ Ollama  │  │ OpenAI  │  │ Anthropic│                        │
│  │ Adapter │  │ Adapter │  │ Adapter  │                        │
│  └─────────┘  └─────────┘  └─────────┘                        │
│                         (Unified abstraction, upper layer doesn't  │
│                          depend on specific model)                │
└────────────────────────────────────────────────────────────────┘
```

### Complete Call Flow

```
LLM Output
    │
    ▼
Parser Parse ── Extract JSON
    │ Failed
    ▼
Fix JSON ── Fix broken JSON
    │ Failed
    ▼
Validator Validate ── Schema validation
    │ Failed
    ▼
Retry (3 times) ── Auto retry
    │
    ▼
Return structured result
```

---

**Version**: 1.0  
**Last Updated**: 2026-03-24  
**Maintainer**: GoAgent Team