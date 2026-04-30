# GoAgent Multi-Agent Workflow Example

A generic multi-agent task processing system with DAG-based workflow orchestration.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.26+
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
| **Sub Agents** | Task execution (domain-specific processing) | `internal/agents/sub/` |
| **AHP Protocol** | Inter-agent communication | `internal/protocol/ahp/` |
| **LLM Client** | LLM interaction | `internal/llm/client.go` |
| **Template Renderer** | Template variable substitution | `internal/workflow/engine/template.go` |

### Workflow Step Configuration

| Step | Agent Type | Dependencies | Code Location |
|------|-----------|---------------|---------------|
| **analyze-input** | top | None | `examples/simple_newapi/config/workflow.yaml:15-25` |
| **research-topic-a** | top | analyze-input | `examples/simple_newapi/config/workflow.yaml:30-40` |
| **research-topic-b** | bottom | analyze-input | `examples/simple_newapi/config/workflow.yaml:45-55` |
| **research-topic-c** | top | analyze-input | `examples/simple_newapi/config/workflow.yaml:60-70` |
| **aggregate-results** | bottom | All research steps | `examples/simple_newapi/config/workflow.yaml:75-85` |

### Key Feature Implementations

**Code Location References**:
- DAG construction: `internal/workflow/engine/executor.go:80-120`
- Topological sort: `internal/workflow/engine/executor.go:150-200`
- Parallel execution: `internal/workflow/engine/executor.go:250-300`
- Template variable parsing: `internal/workflow/engine/template.go:50-100`
- Step dependency management: `internal/workflow/engine/types.go:30-80`
- Result aggregation: `examples/simple_newapi/main.go:100-140`

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
    - id: "agent-researcher-a"
      type: "top"
      name: "Researcher A"
```

### 3. Define Your Workflow

Edit `config/workflow.yaml`:

```yaml
id: "multi-agent-workflow"
steps:
  - id: "analyze-input"
    name: "Analyze Input"
    agent_type: "top"
    input: "Analyze user requirements from: {{.input}}"

  - id: "research-topic-a"
    name: "Research Topic A"
    agent_type: "top"
    depends_on: ["analyze-input"]
    input: "Based on analysis from {{.analyze-input}}, research and compile findings"
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
  - id: "analyze-input"
    name: "Analyze Input"
    agent_type: "top"
    input: "Analyze from: {{.input}}"

  - id: "process"
    name: "Process"
    depends_on: ["analyze-input"]
    input: "Process based on: {{.analyze-input}}"
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
=== GoAgent Multi-Agent Workflow Example ===

=== Configured Agents ===
  - agent-researcher-a (top): Researcher A
  - agent-researcher-b (bottom): Researcher B
  - agent-analyzer (top): Data Analyzer

=== User Query ===
Analyze the latest tech trends in AI and cloud computing...

=== Executing Workflow ===

=== Workflow Execution Result ===
Execution ID: exec-xxx
Status: completed
Duration: 45s
Total Steps: 5

=== Task Results ===

Analyze Input:
  Status: completed
  Duration: 5s
  Output: {"domains": ["AI", "cloud computing"], "priority": "high"}

Research Topic A:
  Status: completed
  Duration: 12s
  Output: {"items": [{"name": "LLM Advances", "reason": "Key trend in AI"}]}

Research Topic B:
  Status: completed
  Duration: 11s  # Ran in parallel with Research Topic A

Research Topic C:
  Status: completed
  Duration: 10s  # Ran in parallel with Research Topic A

Aggregate Results:
  Status: completed
  Duration: 7s
  Output: Comprehensive analysis report...

=== Final Output ===
{
  "report": {
    "summary": "...",
    "key_findings": ["..."],
    "priorities": ["..."]
  }
}
```

## Next Steps

- Add more agents to your config
- Create complex workflows with multiple dependencies
- Use retry policies for robustness
- Add metadata for tracking and debugging
