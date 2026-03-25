# YAML Configuration Quick Start Guide

## 什么是 YAML 配置系统？

YAML 配置系统让你可以通过简单的配置文件定义和执行复杂的图工作流，无需编写任何代码。

## 快速开始

### 1. 创建 YAML 配置文件

创建一个名为 `my_workflow.yaml` 的文件：

```yaml
graph:
  id: "我的工作流"
  start_node: "步骤1"

  nodes:
    - id: "步骤1"
      type: "function"
      description: "验证输入"

    - id: "步骤2"
      type: "function"
      description: "处理数据"

    - id: "步骤3"
      type: "function"
      description: "保存结果"

  edges:
    - from: "步骤1"
      to: "步骤2"
    - from: "步骤2"
      to: "步骤3"
```

### 2. 运行工作流

```bash
cd examples/graph_demo/yaml_config
go run yaml_example.go my_workflow.yaml
```

就这么简单！你的工作流就会自动执行。

## 配置文件结构

```yaml
graph:
  id: "工作流ID"           # 必需：唯一标识符
  start_node: "起始节点"   # 必需：入口节点
  nodes: [...]            # 必需：节点列表
  edges: [...]            # 必需：边列表
```

## 节点类型

### 函数节点 (function)

最简单的节点类型，执行基本操作：

```yaml
- id: "node1"
  type: "function"
  description: "节点描述"
```

### 代理节点 (agent)

使用已注册的代理：

```yaml
- id: "node1"
  type: "agent"
  description: "代理节点"
  config:
    agent_id: "my-agent"  # 代理ID
```

### 工具节点 (tool)

使用已注册的工具：

```yaml
- id: "node1"
  type: "tool"
  description: "工具节点"
  config:
    tool_id: "my-tool"  # 工具ID
```

## 示例

### 简单线性工作流

```bash
go run yaml_example.go simple_workflow.yaml
```

输出：
```
=== Loading graph from YAML: simple_workflow.yaml ===
Graph ID: simple-workflow
Start Node: validate
Nodes: 3
Edges: 2

=== Executing graph ===
[Node validate] Executing...
  Description: Validate input data
[Node process] Executing...
  Description: Process the data
[Node save] Executing...
  Description: Save results

=== Execution Results ===
Graph ID: simple-workflow
Duration: 260.541µs
```

### 条件分支工作流

```bash
go run yaml_example.go conditional_workflow.yaml
```

## 更多示例

查看 `examples/graph_demo/yaml_config/` 目录中的完整示例：

- `simple_workflow.yaml` - 简单线性流程
- `conditional_workflow.yaml` - 条件分支流程
- `yaml_example.go` - 完整的示例程序

## 配置验证

系统会自动验证配置：

- ✅ 图 ID 不能为空
- ✅ 起始节点必须存在
- ✅ 所有节点 ID 必须唯一
- ✅ 所有边必须引用有效的节点
- ✅ 节点类型必须有效

## 状态管理

节点可以通过共享状态读写数据：

```yaml
node.<node-id>.timestamp: "executed"
node.<node-id>.status: "success"
```

## 最佳实践

1. 使用有意义的节点 ID
2. 为复杂节点添加描述
3. 在 YAML 文件中使用注释组织内容
4. 先用简单案例测试配置
5. 部署前验证配置

## 获取帮助

- 详细文档：`examples/graph_demo/yaml_config/README.md`
- 完整示例：`examples/graph_demo/`
- API 文档：`api/service/graph/`

## 下一步

尝试修改示例配置文件，创建你自己的工作流！