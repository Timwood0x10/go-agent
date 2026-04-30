# GoAgent 多 Agent 工作流示例

一个使用基于 DAG 的工作流进行多 Agent 编排的通用任务处理系统。

## 快速开始

### 1. 配置你的 LLM

编辑 `config/server.yaml`：

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"
```

### 2. 配置你的 Agents

编辑 `config/server.yaml`：

```yaml
agents:
  sub:
    - id: "agent-researcher-a"
      type: "top"
      name: "Researcher A"
```

### 3. 定义你的工作流

编辑 `config/workflow.yaml`：

```yaml
id: "multi-agent-workflow"
steps:
  - id: "analyze-input"
    name: "分析输入"
    agent_type: "top"
    input: "分析用户需求: {{.input}}"

  - id: "research-topic-a"
    name: "研究主题 A"
    agent_type: "top"
    depends_on: ["analyze-input"]
    input: "基于 {{.analyze-input}} 的分析，研究和整理发现"
```

### 4. 运行

```bash
cd examples/simple_newapi
go run main.go
```

## 工作流编排

系统支持基于 DAG 的工作流编排：

### 并行执行

```yaml
steps:
  - id: "step1"
    name: "第一步"
    agent_type: "top"

  - id: "step2"
    name: "并行步骤 1"
    depends_on: ["step1"]

  - id: "step3"
    name: "并行步骤 2"
    depends_on: ["step1"]  # 与 step2 并行运行
```

### 串行执行

```yaml
steps:
  - id: "step1"
    name: "第一步"
    agent_type: "top"

  - id: "step2"
    name: "第二步"
    depends_on: ["step1"]

  - id: "step3"
    name: "第三步"
    depends_on: ["step2"]  # 在 step2 之后运行
```

### 复杂 DAG

```yaml
steps:
  - id: "analyze"
    name: "分析"
    agent_type: "leader"

  - id: "code"
    name: "生成代码"
    depends_on: ["analyze"]
    agent_type: "code"

  - id: "test"
    name: "生成测试"
    depends_on: ["code"]
    agent_type: "test"

  - id: "docs"
    name: "生成文档"
    depends_on: ["analyze"]
    agent_type: "docs"

  - id: "review"
    name: "审查"
    depends_on: ["code", "docs"]  # 等待两者
    agent_type: "review"
```

## 工作流功能

### 步骤配置

每个步骤支持：

- **id**: 唯一标识符
- **name**: 显示名称
- **agent_type**: 要使用的 Agent 类型
- **input**: 带有模板变量的任务描述
- **depends_on**: 此步骤依赖的步骤 ID 列表
- **timeout**: 执行超时
- **retry_policy**: 重试配置
- **metadata**: 额外元数据

### 模板变量

使用 `{{.step_id}}` 引用之前步骤的输出：

```yaml
steps:
  - id: "analyze-input"
    name: "分析输入"
    agent_type: "top"
    input: "从以下内容分析: {{.input}}"

  - id: "process"
    name: "处理"
    depends_on: ["analyze-input"]
    input: "基于以下内容处理: {{.analyze-input}}"
```

### 重试策略

配置重试行为：

```yaml
retry_policy:
  max_attempts: 3
  initial_delay: 1s
  max_delay: 5s
  backoff_multiplier: 2.0
```

## 工作原理

1. **加载配置** - 从 YAML 加载 agents 和工作流
2. **构建 DAG** - 根据步骤依赖关系创建有向无环图
3. **拓扑排序** - 确定执行顺序
4. **并行执行** - 并发运行独立步骤
5. **收集结果** - 收集所有步骤的输出

## 示例输出

```
=== GoAgent Multi-Agent Workflow Example ===

=== 配置的 Agents ===
  - agent-researcher-a (top): Researcher A
  - agent-researcher-b (bottom): Researcher B
  - agent-analyzer (top): Data Analyzer

=== 用户查询 ===
分析 AI 和云计算的最新技术趋势...

=== 执行工作流 ===

=== 工作流执行结果 ===
执行 ID: exec-xxx
状态: 已完成
持续时间: 45秒
总步骤数: 5

=== 任务结果 ===

分析输入:
  状态: 已完成
  持续时间: 5秒
  输出: {"domains": ["AI", "云计算"], "priority": "high"}

研究主题 A:
  状态: 已完成
  持续时间: 12秒
  输出: {"items": [{"name": "LLM 进展", "reason": "AI 领域关键趋势"}]}

研究主题 B:
  状态: 已完成
  持续时间: 11秒  # 与研究主题 A 并行运行

研究主题 C:
  状态: 已完成
  持续时间: 10秒  # 与研究主题 A 并行运行

聚合结果:
  状态: 已完成
  持续时间: 7秒
  输出: 综合分析报告...

=== 最终输出 ===
{
  "report": {
    "summary": "...",
    "key_findings": ["..."],
    "priorities": ["..."]
  }
}
```

## 下一步

- 在你的配置中添加更多 agents
- 创建具有多个依赖关系的复杂工作流
- 使用重试策略提高鲁棒性
- 添加元数据用于跟踪和调试

## 技术栈和组件

### 使用的技术
- **语言**: Go 1.26+
- **LLM 提供商**: Ollama (llama3.2) 或其他支持 OpenAI API 的服务
- **配置格式**: YAML
- **工作流编排**: DAG（有向无环图）
- **模板引擎**: Go 模板语法
- **并发控制**: errgroup

### 使用的核心组件

| 组件 | 用途 | 代码位置 |
|------|------|----------|
| **Workflow Engine** | DAG 工作流编排 | `internal/workflow/engine/executor.go` |
| **Leader Agent** | 工作流启动和协调 | `internal/agents/leader/` |
| **Sub Agents** | 任务执行（领域特定处理） | `internal/agents/sub/` |
| **AHP 协议** | Agent 间通信 | `internal/protocol/ahp/` |
| **LLM Client** | LLM 交互 | `internal/llm/client.go` |
| **Template Renderer** | 模板变量替换 | `internal/workflow/engine/template.go` |

### 工作流步骤配置

| 步骤 | Agent 类型 | 依赖关系 | 代码位置 |
|------|-----------|---------|----------|
| **analyze-input** | top | 无 | `examples/simple_newapi/config/workflow.yaml:15-25` |
| **research-topic-a** | top | analyze-input | `examples/simple_newapi/config/workflow.yaml:30-40` |
| **research-topic-b** | bottom | analyze-input | `examples/simple_newapi/config/workflow.yaml:45-55` |
| **research-topic-c** | top | analyze-input | `examples/simple_newapi/config/workflow.yaml:60-70` |
| **aggregate-results** | bottom | 所有研究步骤 | `examples/simple_newapi/config/workflow.yaml:75-85` |

### 关键特性实现

**代码位置佐证**:
- DAG 构建: `internal/workflow/engine/executor.go:80-120`
- 拓扑排序: `internal/workflow/engine/executor.go:150-200`
- 并行执行: `internal/workflow/engine/executor.go:250-300`
- 模板变量解析: `internal/workflow/engine/template.go:50-100`
- 步骤依赖管理: `internal/workflow/engine/types.go:30-80`
- 结果聚合: `examples/simple_newapi/main.go:100-140`

---

**创建日期**: 2026-03-23
**示例类型**: 工作流编排演示
**代码位置**: `examples/simple_newapi/main.go:1-400`
