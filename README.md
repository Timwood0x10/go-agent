# GoAgent

GoAgent is a generic multi-agent framework implemented in Go, supporting multi-agent collaboration, memory management, and tool invocation.

## Architecture Diagram

```mermaid
graph TB

%% =======================
%% User Layer
%% =======================

User[用户输入<br/>我想去东京旅行5天<br/>预算10000]


%% =======================
%% Agent Runtime
%% =======================

subgraph AgentRuntime["Agent Runtime"]

    %% Leader
    subgraph LeaderAgent["Leader Agent"]
        MemoryManager[MemoryManager<br/>记忆管理]
        ParseProfile[ParseProfile<br/>LLM解析]
        TaskPlanner[TaskPlanner<br/>LLM任务规划]
        Dispatcher[Dispatcher<br/>errgroup并发]

        MemoryManager --> TaskPlanner
        ParseProfile --> TaskPlanner
        TaskPlanner --> Dispatcher
    end


    %% Sub Agents
    subgraph SubAgents["Sub Agents (并行执行)"]
        DestinationAgent[Destination Agent]
        FoodAgent[Food Agent]
        HotelAgent[Hotel Agent]
        ItineraryAgent[Itinerary Agent]
    end


    %% AHP Protocol
    subgraph AHP["AHP Protocol (Agent Communication)"]

        MessageQueue[In-Memory Message Queue<br/>channel]

        subgraph Queues["Agent Queues"]
            LeaderQueue[leader queue]
            AgentDstQueue[agent_destination]
            AgentFoodQueue[agent_food]
            AgentHotelQueue[agent_hotel]
        end

        MessageQueue --> LeaderQueue
        MessageQueue --> AgentDstQueue
        MessageQueue --> AgentFoodQueue
        MessageQueue --> AgentHotelQueue
    end

end


%% =======================
%% Core Services
%% =======================

subgraph CoreServices["Core Services"]

    %% LLM
    subgraph LLMSystem["LLM System"]
        OpenAI[OpenAI]
        Ollama[Ollama]
        OpenRouter[OpenRouter]
    end

    %% Tools
    subgraph ToolsSystem["Tools System"]

        subgraph Tools["Built-in Tools"]
            Calculator[calculator]
            DateTime[datetime]
            FileTools[file_tools]
            HTTPRequest[http_request]
            WebScraper[web_scraper]
            KnowledgeSearch[knowledge_search]
        end

    end


    %% Workflow
    subgraph WorkflowEngine["Workflow Engine"]

        DAGEngine[DAG 执行引擎<br/>步骤依赖 + 并行控制]

        WorkflowDef[Workflow Definition<br/>YAML steps<br/>depends_on<br/>variables]

    end


    %% Embedding
    subgraph EmbeddingServer["Embedding Server"]

        FastAPI[FastAPI Service]

        subgraph EmbeddingModels["Embedding Models"]
            OllamaEmbed[Ollama<br/>qwen3-embedding:0.6b]
            SentenceTransformers[e5-large]
        end

        FastAPI --> OllamaEmbed
        FastAPI --> SentenceTransformers

    end

end


%% =======================
%% Storage Layer
%% =======================

subgraph Storage["Storage (PostgreSQL + pgvector)"]

    %% Memory Tables
    subgraph MemoryTables["Memory System"]

        Conversations[conversations<br/>会话记录]

        TaskResults[task_results<br/>任务执行结果]

        DistilledMem[distilled_memories<br/>蒸馏记忆]

    end


    %% Knowledge Base
    subgraph KnowledgeBase["Knowledge Base"]

        KnowledgeChunks[knowledge_chunks_1024]

        VectorIndex[Vector Index<br/>ivfflat]

        KnowledgeChunks --> VectorIndex

    end


    %% Storage Features
    subgraph StorageFeatures["Storage Features"]

        ConnectionPool[连接池<br/>Max 25 / Idle 10]

        TenantIsolation[RLS 租户隔离<br/>SET app.tenant_id]

        WriteBuffer[批量写入缓冲]

        RetrievalGuard[检索限流<br/>100 req/s]

        Timeout[查询超时<br/>30s]

    end

end


%% =======================
%% Connections
%% =======================

User --> LeaderAgent

LeaderAgent --> AHP
AHP --> SubAgents
SubAgents --> AHP

AgentRuntime --> CoreServices

CoreServices --> Storage
EmbeddingServer --> Storage
WorkflowEngine --> Storage


%% =======================
%% Styles
%% =======================

classDef user fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
classDef runtime fill:#e3f2fd,stroke:#1565c0,stroke-width:2px
classDef services fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px
classDef storage fill:#e0f2f1,stroke:#00695c,stroke-width:2px

class User user
class AgentRuntime runtime
class CoreServices services
class Storage storage
```

### Embedding Service Detailed Architecture

The Embedding Service is a standalone vector embedding service for GoAgent, supporting multiple backends:

```mermaid
graph LR
    subgraph Client["Client"]
        Agent[GoAgent Application]
    end
    
    Agent --> |HTTP REST API| EmbeddingServer
    
    subgraph EmbeddingServer["Embedding Service (FastAPI)"]
        API[FastAPI<br/>REST API]
        Normalizer[Text Normalization<br/>Unicode + Lowercase + Trim]
        Cache[Redis Cache<br/>TTL: 24 hours]
        Model[Model Engine]
        
        API --> Normalizer
        Normalizer --> |Cache Miss| Model
        Normalizer --> |Cache Hit| API
        Model --> Cache
        Cache --> API
    end
    
    subgraph Backends["Backend Support"]
        OllamaBackend[Ollama<br/>qwen3-embedding:0.6b<br/>Local Deployment]
        TransformerBackend[SentenceTransformers<br/>e5-large<br/>Cloud Deployment]
    end
    
    Model --> OllamaBackend
    Model --> TransformerBackend
    
    subgraph Features["Features"]
        Batch[Batch Processing]
        Health[Health Check]
        Normalization[Auto Normalization]
        VectorNorm[Vector Normalization]
    end
    
    API --> Features
    
    classDef client fill:#e1f5e1,stroke:#4caf50,stroke-width:2px
    classDef server fill:#e3f2fd,stroke:#2196f3,stroke-width:2px
    classDef backend fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    classDef feature fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px
    
    class Client client
    class EmbeddingServer server
    class Backends backend
    class Features feature
```

**Embedding Service Features**:
- **High Performance**: Supports Ollama local deployment and SentenceTransformers cloud deployment
- **Smart Caching**: Redis cache + text normalization to avoid cache misses
- **Batch Processing**: Supports batch vector generation for improved efficiency
- **Auto Normalization**: Vectors automatically normalized to unit vectors for accurate cosine similarity
- **Health Check**: Built-in health check endpoint

**Configuration File**: `services/embedding/.env`
```env
BACKEND_TYPE=ollama              # Backend type: ollama / transformers
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=qwen3-embedding:0.6b
MODEL_NAME=qwen3-embedding:0.6b
EMBEDDING_DIM=1024
REDIS_URL=redis://localhost:6379
CACHE_TTL=86400
HOST=0.0.0.0
PORT=8000
```

**Code Locations**: 
- `services/embedding/app.py` - Service main program
- `services/embedding/config.py` - Configuration management
- `internal/storage/postgres/embedding/client.go` - Go client

## Tech Stack

### Core Technologies
- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+ with pgvector extension
- **Concurrency**: errgroup, sync
- **Protocol**: Custom AHP Protocol
- **Embedding Service**: FastAPI + Ollama/SentenceTransformers
- **Cache**: Redis

### Main Components
| Component | Purpose | Code Location |
|----------|---------|----------------|
| **Agent System** | Leader/Sub Agent collaboration | `internal/agents/` |
| **Protocol Layer** | Inter-agent communication and heartbeat | `internal/protocol/ahp/` |
| **Memory System** | Session, task, and distilled memory | `internal/memory/` |
| **Storage Layer** | PostgreSQL + pgvector | `internal/storage/postgres/` |
| **Tool System** | Tool registry and invocation | `internal/tools/` |
| **Workflow Engine** | DAG workflow orchestration | `internal/workflow/engine/` |
| **Embedding Service** | Vector embedding generation | `services/embedding/` |

### Dependencies
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/google/uuid` - UUID generation
- `github.com/stretchr/testify` - Testing framework
- `golang.org/x/sync` - Concurrent extensions
- `gopkg.in/yaml.v3` - YAML parsing
- `fastapi` - Embedding service framework
- `redis` - Cache support

## Configuration

### 1. LLM Configuration

**Config File**: `examples/travel/config.yaml`

```yaml
llm:
  provider: openrouter        # LLM provider: openai / ollama / openrouter
  api_key: ""                  # API Key (recommended: use env var OPENROUTER_API_KEY)
  base_url: https://openrouter.ai/api/v1
  model: meta-llama/llama-3.1-8b-instruct
  timeout: 60                  # Request timeout (seconds)
  max_tokens: 2048              # Max response tokens
```

**Code Location**: `internal/llm/client.go:80-100`

### 2. Agent Configuration

```yaml
agents:
  leader:
    id: leader-travel
    max_steps: 10              # Max execution steps
    max_parallel_tasks: 4      # Max parallel tasks
    enable_cache: true          # Enable caching

  sub:
    - id: agent-destination
      type: destination         # Agent type: destination/food/hotel/itinerary
      triggers: [destination]   # Trigger keywords
      max_retries: 3             # Max retry attempts
      timeout: 30                # Timeout (seconds)
```

**Code Location**: `internal/agents/leader/agent.go:30-50`

### 3. Database Configuration

```yaml
storage:
  enabled: true               # Enable PostgreSQL storage
  type: postgres
  host: localhost
  port: 5433                # Docker default port is 5433
  user: postgres
  password: postgres
  database: goagent
  
  pgvector:
    enabled: true             # Enable pgvector
    dimension: 1024           # Vector dimension
```

**Code Location**: `internal/storage/postgres/pool.go:35-50`

### 4. Embedding Service Configuration

```yaml
embedding:
  service_url: http://localhost:8000    # Embedding service address
  model: qwen3-embedding:0.6b          # Model name
  dimension: 1024                       # Vector dimension
  timeout: 30                           # Request timeout (seconds)
```

**Code Location**: `internal/storage/postgres/embedding/client.go:30-50`

### 5. Memory Configuration

```yaml
memory:
  enabled: true               # Enable memory system
  enable_distillation: true   # Enable memory distillation
  distillation_threshold: 3   # Trigger distillation every N rounds
```

**Code Location**: `examples/knowledge-base/main.go:750-760`

### 6. Retrieval Configuration

```yaml
knowledge:
  chunk_size: 1000             # Document chunk size
  chunk_overlap: 100            # Chunk overlap
  top_k: 10                    # Return top K results
  min_score: 0.6               # Minimum similarity threshold
```

**Code Location**: `internal/storage/postgres/repositories/knowledge_repository.go:100-120`

## Quick Start

### 1. Set Environment

```bash
# Set API Key (recommended: use environment variable)
export OPENROUTER_API_KEY="your-api-key"

# Or set in config file (not recommended)
```

### 2. Start Database (Optional, for persistence)

```bash
# Quick start PostgreSQL + pgvector with Docker
docker run -d \
  --name goagent-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goagent \
  -p 5433:5432 \
  pgvector/pgvector:pg15

# Verify connection
docker exec -it goagent-db psql -U postgres -d goagent -c "SELECT version();"
```

### 3. Start Embedding Service (for vector retrieval)

```bash
# Navigate to embedding service directory
cd services/embedding

# Run setup script (install dependencies and model)
./setup.sh

# Start service
./start.sh

# Verify service
curl http://localhost:8000/health
```

### 4. Run Examples

```bash
# Travel planning example
cd examples/travel
go run main.go

# Knowledge base Q&A example (requires database + embedding service)
cd examples/knowledge-base
go run main.go --save README.md  # Import document
go run main.go --chat             # Start Q&A
```

## Project Structure

```
goagent/
├── examples/               # Example applications
│   ├── travel/              # Travel planning
│   ├── knowledge-base/       # Knowledge base Q&A
│   └── simple/              # Simple example
├── internal/                # Core implementation
│   ├── agents/              # Agent system
│   │   ├── base/            # Agent base interfaces
│   │   ├── leader/          # Leader Agent
│   │   └── sub/             # Sub Agent
│   ├── protocol/             # AHP protocol
│   ├── storage/              # PostgreSQL + pgvector
│   ├── memory/               # Memory system
│   └── workflow/             # Workflow engine
├── services/                # Standalone services
│   └── embedding/           # Embedding service
│       ├── app.py           # FastAPI service
│       ├── config.py        # Configuration management
│       └── requirements.txt # Python dependencies
├── api/                     # API layer
│   ├── service/             # Service interfaces
│   └── client/              # Client
└── docs/                    # Documentation
```

## Documentation

- [Quick Start Guide](docs/quick_start_en.md) - Detailed installation and configuration guide
- [FAQ](docs/faq_en.md) - Common issues and solutions
- [Architecture Documentation](docs/arch.md) - Complete architecture design
- [Embedding Service Documentation](services/embedding/README.md) - Embedding service details
- [Integration Guide](docs/integration_guide.md) - How to integrate into existing projects

## Examples

- [Travel Planning](examples/travel/) - Multi-agent collaboration
- [Knowledge Base Q&A](examples/knowledge-base/) - Vector search
- [Simple Example](examples/simple/) - Basic usage
- [Capability Demo](examples/capability-demo/) - Full feature showcase

## Development Guide

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/agents/...

# Run tests with coverage
go test -cover ./...
```

### Building Project

```bash
# Build main program
go build -o bin/goagent ./cmd/server

# Build examples
go build -o bin/travel ./examples/travel
```

### Code Standards

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run
```

---

**Last Updated**: 2026-03-23  
**Version**: v1.0.0  
**Code Base**: Based on actual go-agent code analysis