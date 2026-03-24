# GoAgent Fashion Recommendation System with Workflow

A fashion recommendation system with multi-agent orchestration using DAG-based workflow.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.21+
- **LLM Provider**: Ollama (llama3.2) or other OpenAI API-compatible services
- **Configuration Format**: YAML
- **Workflow Orchestration**: DAG (Directed Acyclic Graph)
- **Template Engine**: Go template syntax
- **Concurrency Control**: errgroup

### Core Components Used

| Component | Purpose | Code Location |
|-----------|---------|---------------|
| **Workflow Engine** | DAG workflow orchestration | `internal/workflow/engine/executor.go` |
| **Leader Agent** | Workflow startup and coordination | `internal/agents/leader/` |
| **Sub Agents** | Task execution (fashion recommendations) | `internal/agents/sub/` |
| **AHP Protocol** | Inter-agent communication | `internal/protocol/ahp/` |
| **LLM Client** | LLM interaction | `internal/llm/client.go` |
| **Template Renderer** | Template variable substitution | `internal/workflow/engine/template.go` |

### Workflow Step Configuration

| Step | Agent Type | Dependencies | Code Location |
|------|-----------|---------------|---------------|
| **extract-profile** | top | None | `examples/simple_newapi/config/workflow.yaml:15-25` |
| **recommend-tops** | top | extract-profile | `examples/simple_newapi/config/workflow.yaml:30-40` |
| **recommend-bottoms** | bottom | extract-profile | `examples/simple_newapi/config/workflow.yaml:45-55` |
| **recommend-shoes** | shoes | extract-profile | `examples/simple_newapi/config/workflow.yaml:60-70` |
| **aggregate** | leader | All recommendation steps | `examples/simple_newapi/config/workflow.yaml:75-85` |

### Key Feature Implementations

**Code Location References**:
- DAG construction: `internal/workflow/engine/executor.go:80-120`
- Topological sort: `internal/workflow/engine/executor.go:150-200`
- Parallel execution: `internal/workflow/engine/executor.go:250-300`
- Template variable parsing: `internal/workflow/engine/template.go:50-100`
- Step dependency management: `internal/workflow/engine/types.go:30-80`
- Result aggregation: `examples/simple_newapi/main.go:150-200`

## Quick Start

### 1. Configure your LLM

Edit `config/server.yaml`:

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
```

### 2. Configure your Agents

Edit `config/server.yaml`:

```yaml
agents:
  sub:
    - id: "agent-top"
      type: "top"
      category: "tops"
      name: "Top Wear Recommender"
```

### 3. Define Your Workflow

Edit `config/workflow.yaml`:

```yaml
id: "fashion-recommendation"
steps:
  - id: "extract-profile"
    name: "Extract User Preferences"
    agent_type: "top"
    input: "Extract preferences: {{.input}}"
    
  - id: "recommend-tops"
    name: "Recommend Top Wear"
    agent_type: "top"
    depends_on: ["extract-profile"]
    input: "Recommend tops based on {{.extract-profile}}"
```

### 4. Run

```bash
cd examples/simple_newapi
go run main.go
```

## Workflow Orchestration

The system supports DAG-based workflow orchestration:

### Parallel Execution

```yaml
steps:
  - id: "step1"
    name: "First Step"
    agent_type: "top"
    
  - id: "step2"
    name: "Parallel Step 1"
    depends_on: ["step1"]
    
  - id: "step3"
    name: "Parallel Step 2"
    depends_on: ["step1"]  # Runs in parallel with step2
```

### Serial Execution

```yaml
steps:
  - id: "step1"
    name: "First Step"
    agent_type: "top"
    
  - id: "step2"
    name: "Second Step"
    depends_on: ["step1"]
    
  - id: "step3"
    name: "Third Step"
    depends_on: ["step2"]  # Runs after step2
```

### Complex DAG

```yaml
steps:
  - id: "analyze"
    name: "Analyze"
    agent_type: "leader"
    
  - id: "code"
    name: "Generate Code"
    depends_on: ["analyze"]
    agent_type: "code"
    
  - id: "test"
    name: "Generate Tests"
    depends_on: ["code"]
    agent_type: "test"
    
  - id: "docs"
    name: "Generate Docs"
    depends_on: ["analyze"]
    agent_type: "docs"
    
  - id: "review"
    name: "Review"
    depends_on: ["code", "docs"]  # Waits for both
    agent_type: "review"
```

## Workflow Features

### Step Configuration

Each step supports:

- **id**: Unique identifier
- **name**: Display name
- **agent_type**: Agent type to use
- **input**: Task description with template variables
- **depends_on**: List of step IDs this step depends on
- **timeout**: Execution timeout
- **retry_policy**: Retry configuration
- **metadata**: Additional metadata

### Template Variables

Use `{{.step_id}}` to reference output from previous steps:

```yaml
steps:
  - id: "extract-profile"
    name: "Extract Profile"
    agent_type: "top"
    input: "Extract from: {{.input}}"
    
  - id: "recommend"
    name: "Recommend"
    depends_on: ["extract-profile"]
    input: "Recommend based on: {{.extract-profile}}"
```

### Retry Policy

Configure retry behavior:

```yaml
retry_policy:
  max_attempts: 3
  initial_delay: 1s
  max_delay: 5s
  backoff_multiplier: 2.0
```

## How It Works

1. **Load Configuration** - Load agents and workflow from YAML
2. **Build DAG** - Create directed acyclic graph from step dependencies
3. **Topological Sort** - Determine execution order
4. **Execute in Parallel** - Run independent steps concurrently
5. **Collect Results** - Gather outputs from all steps

## Example Output

```
=== GoAgent Fashion Recommendation System with Workflow ===

=== Configured Agents ===
  - agent-top (top): Top Wear Recommender
  - agent-bottom (bottom): Bottom Wear Recommender
  - agent-shoes (shoes): Shoes Recommender

=== User Query ===
I want casual clothes for daily commute...

=== Executing Workflow ===

=== Workflow Execution Result ===
Execution ID: exec-xxx
Status: completed
Duration: 45s
Total Steps: 5

=== Step Results ===

✓ Step: Extract User Preferences
  Status: completed
  Duration: 5s
  Output: {"style": ["casual"], "budget": {"min": 500, "max": 1000}}

✓ Step: Recommend Top Wear
  Status: completed
  Duration: 12s
  Output: {"items": [{"name": "Cotton T-Shirt", "price": 599}]}

✓ Step: Recommend Bottom Wear
  Status: completed
  Duration: 11s  # Ran in parallel with tops

✓ Step: Recommend Shoes
  Status: completed
  Duration: 10s  # Ran in parallel with tops

✓ Step: Aggregate Recommendations
  Status: completed
  Duration: 7s
  Output: Complete outfit recommendation...

=== Final Output ===
{
  "outfit": {
    "top": "...",
    "bottom": "...",
    "shoes": "..."
  }
}
```

## Next Steps

- Add more agents to your config
- Create complex workflows with multiple dependencies
- Use retry policies for robustness
- Add metadata for tracking and debugging
