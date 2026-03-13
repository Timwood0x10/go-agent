# Workflow Engine 设计文档

## 1. 概述

Workflow Engine 负责加载和执行用户定义的工作流，实现 Agent 的灵活编排。用户可以通过 YAML/JSON 文件自定义任务流程，无需修改代码。

## 2. 工作流定义

### 2.1 基本结构

```yaml
# workflow.yaml
name: "穿搭推荐流程"
version: "1.0.0"
description: "默认的时尚穿搭推荐工作流"

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

### 2.2 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| name | 是 | 工作流名称 |
| version | 否 | 版本号 |
| description | 否 | 描述 |
| agents | 是 | Agent 列表 |
| execution | 是 | 执行配置 |

## 3. Agent 配置

```yaml
agents:
  - id: agent_top
    type: sub              # leader | sub
    prompt_file: ./agents/agent_top.md
    depends_on: [leader]  # 依赖的 Agent ID
    config:
      timeout: 60s
      max_retries: 3
      priority: 10
```

## 4. 执行阶段

### 4.1 Phase 执行

```yaml
execution:
  phase1:
    - agent_top        # 并行执行
    - agent_bottom
    
  phase2:             # 等待 phase1 完成后执行
    - agent_shoes
    - agent_accessory
    
  phase3:             # 可选更多阶段
    - agent_summary
```

### 4.2 DAG 依赖

支持更复杂的 DAG 依赖：

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

## 5. 核心模块

### 5.1 Loader

```go
type WorkflowLoader interface {
    // Load 从文件加载工作流
    Load(path string) (*Workflow, error)
    
    // LoadFromBytes 从字节加载
    LoadFromBytes(data []byte) (*Workflow, error)
    
    // Validate 验证工作流
    Validate(wf *Workflow) error
}
```

### 5.2 Executor

```go
type WorkflowExecutor interface {
    // Execute 执行工作流
    Execute(ctx context.Context, wf *Workflow, input *ExecuteInput) (*ExecuteOutput, error)
    
    // ExecutePhase 执行单个阶段
    ExecutePhase(ctx context.Context, phase string) ([]*TaskResult, error)
    
    // GetStatus 获取执行状态
    GetStatus(execID string) (*ExecutionStatus, error)
}
```

### 5.3 Registry

```go
type AgentRegistry interface {
    // Register 注册 Agent 定义
    Register(ctx context.Context, def *AgentDefinition) error
    
    // Get 获取 Agent 定义
    Get(ctx context.Context, name string) (*AgentDefinition, error)
    
    // List 列出所有 Agent
    List(ctx context.Context) ([]*AgentDefinition, error)
    
    // Reload 热加载
    Reload(ctx context.Context) error
}
```

## 6. 运行时架构

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

## 7. 工作流目录结构

```
workflows/
├── default.yaml          # 默认工作流
├── summer.yaml           # 夏季推荐
├── winter.yaml           # 冬季推荐
├── formal.yaml           # 商务正装
│
├── agents/               # Agent 定义
│   ├── agent_leader.md
│   ├── agent_top.md
│   ├── agent_bottom.md
│   ├── agent_shoes.md
│   ├── agent_head.md
│   └── agent_accessory.md
│
└── templates/            # 模板
    ├── simple.yaml
    └── complex.yaml
```

## 8. 热加载

```go
type HotReloader struct {
    watcher *fsnotify.Watcher
    onChange func(path string)
}

func (r *HotReloader) Start(ctx context.Context) error {
    // 监听文件变化
    // 自动重新加载
}
```

## 9. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| workflow_dir | ./workflows | 工作流目录 |
| agent_dir | ./workflows/agents | Agent 定义目录 |
| default_timeout | 60s | 默认超时 |
| max_parallel | 10 | 最大并行数 |
| enable_hot_reload | true | 热加载开关 |
