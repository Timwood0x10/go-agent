# Workflow Engine Design Document

## 1. Overview

The Workflow Engine is responsible for loading and executing user-defined workflows, enabling flexible Agent orchestration. Users can customize task flows through YAML/JSON files without modifying code.

## 2. Workflow Definition

### 2.1 Basic Structure

```yaml
# workflow.yaml
name: "Fashion Recommendation Flow"
version: "1.0.0"
description: "Default fashion recommendation workflow"

agents:
  - id: leader
    type: leader
    prompt_file: ./agents/agent_leader.md
    
  - id: agent_top
    type: sub
    prompt_file: ./agents/agent_top.md
    depends_on: [leader]
    
  - id: agent_bottom
    type: sub
    prompt_file: ./agents/agent_bottom.md
    depends_on: [leader]
    
  - id: agent_shoes
    type: sub
    prompt_file: ./agents/agent_shoes.md
    depends_on: [leader, agent_top]

execution:
  phase1:
    - agent_top
    - agent_bottom
  phase2:
    - agent_shoes
```

### 2.2 Field Description

| Field | Required | Description |
|-------|----------|-------------|
| name | Yes | Workflow name |
| version | No | Version number |
| description | No | Description |
| agents | Yes | Agent list |
| execution | Yes | Execution configuration |

## 3. Agent Configuration

```yaml
agents:
  - id: agent_top
    type: sub              # leader | sub
    prompt_file: ./agents/agent_top.md
    depends_on: [leader]  # Dependent Agent IDs
    config:
      timeout: 60s
      max_retries: 3
      priority: 10
```

## 4. Execution Phases

### 4.1 Phase Execution

```yaml
execution:
  phase1:
    - agent_top        # Parallel execution
    - agent_bottom
    
  phase2:             # Execute after phase1 completes
    - agent_shoes
    - agent_accessory
    
  phase3:             # Optional more phases
    - agent_summary
```

### 4.2 DAG Dependencies

Supports more complex DAG dependencies:

```yaml
execution:
  dag:
    edges:
      - from: leader
        to: [agent_top, agent_bottom]
      - from: agent_top
        to: agent_shoes
      - from: [agent_bottom, agent_shoes]
        to: agent_summary
```

## 5. Core Modules

### 5.1 Loader

```go
type WorkflowLoader interface {
    // Load loads workflow from file
    Load(path string) (*Workflow, error)
    
    // LoadFromBytes loads from bytes
    LoadFromBytes(data []byte) (*Workflow, error)
    
    // Validate validates workflow
    Validate(wf *Workflow) error
}
```

### 5.2 Executor

```go
type WorkflowExecutor interface {
    // Execute executes workflow
    Execute(ctx context.Context, wf *Workflow, input *ExecuteInput) (*ExecuteOutput, error)
    
    // ExecutePhase executes single phase
    ExecutePhase(ctx context.Context, phase string) ([]*TaskResult, error)
    
    // GetStatus gets execution status
    GetStatus(execID string) (*ExecutionStatus, error)
}
```

### 5.3 Registry

```go
type AgentRegistry interface {
    // Register registers Agent definition
    Register(ctx context.Context, def *AgentDefinition) error
    
    // Get gets Agent definition
    Get(ctx context.Context, name string) (*AgentDefinition, error)
    
    // List lists all Agents
    List(ctx context.Context) ([]*AgentDefinition, error)
    
    // Reload hot reload
    Reload(ctx context.Context) error
}
```

## 6. Runtime Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Workflow Engine                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │    Loader    │───▶│   Executor   │───▶│   Registry   │   │
│  │  (YAML/JSON) │    │   (DAG Run)  │    │  (Agent Pool)│   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│         │                   │                   │              │
│         ▼                   ▼                   ▼              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │ workflow.yaml│    │   Message    │    │ agent_*.md   │   │
│  │              │    │    Queue     │    │              │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 7. Workflow Directory Structure

```
workflows/
├── default.yaml          # Default workflow
├── summer.yaml           # Summer recommendations
├── winter.yaml           # Winter recommendations
├── formal.yaml           # Business formal
│
├── agents/               # Agent definitions
│   ├── agent_leader.md
│   ├── agent_top.md
│   ├── agent_bottom.md
│   ├── agent_shoes.md
│   ├── agent_head.md
│   └── agent_accessory.md
│
└── templates/            # Templates
    ├── simple.yaml
    └── complex.yaml
```

## 8. Hot Reload

```go
type HotReloader struct {
    watcher *fsnotify.Watcher
    onChange func(path string)
}

func (r *HotReloader) Start(ctx context.Context) error {
    // Watch for file changes
    // Auto reload
}
```

## 9. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| workflow_dir | ./workflows | Workflow directory |
| agent_dir | ./workflows/agents | Agent definition directory |
| default_timeout | 60s | Default timeout |
| max_parallel | 10 | Max parallel tasks |
| enable_hot_reload | true | Hot reload enable |
