# Simple New API Example

本示例演示如何使用新的分层API（GoAgent v2）来构建应用。

## 快速开始

### 前置条件

1. Go 1.26.1+
2. Ollama（或其他LLM服务）

### 安装Ollama

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# 启动Ollama服务
ollama serve
```

### 下载模型

```bash
ollama pull llama3.2
```

### 运行示例

```bash
cd examples/simple_newapi
go run main.go
```

## 示例说明

本示例演示了新分层API的以下功能：

### 1. Agent Management（Agent管理）

- 创建不同类型的Agent（上衣、下装、鞋子等）
- 查询Agent信息
- 列出所有Agent

```go
agentSvc, err := client.Agent()
agent, err := agentSvc.CreateAgent(ctx, &core.AgentConfig{
    ID:   "agent-top-1",
    Name: "Top Wear Recommender",
    Type: "agent_top",
})
```

### 2. Memory Management（内存管理）

- 创建用户会话
- 添加对话消息
- 查询历史消息

```go
memorySvc, err := client.Memory()
sessionID, err := memorySvc.CreateSession(ctx, &core.SessionConfig{
    UserID:   "user-001",
    TenantID: "tenant-001",
})
memorySvc.AddMessage(ctx, sessionID, core.MessageRoleUser, "我想找一些适合日常通勤的衣服")
```

### 3. LLM Operations（LLM操作）

- 文本生成（推荐建议）
- 向量嵌入生成（语义搜索）

```go
llmSvc, err := client.LLM()
response, err := llmSvc.GenerateSimple(ctx, "请推荐3件上衣")
embedding, err := llmSvc.GenerateEmbedding(ctx, &core.EmbeddingRequest{
    Input: "休闲风格的通勤穿搭",
})
```

### 4. Knowledge Retrieval（知识检索）

- 添加知识条目
- 搜索相关知识
- 列出知识库内容

```go
retrievalSvc, err := client.Retrieval()
retrievalSvc.AddKnowledge(ctx, &core.KnowledgeItem{
    Content: "休闲风格适合日常通勤",
    Tags:    []string{"casual", "commute"},
})
results, err := retrievalSvc.Search(ctx, "tenant-001", "休闲通勤穿搭建议")
```

## API分层架构

新的API采用三层架构：

```
Client Layer (goagent/api/client)
    ↓
Service Layer (goagent/api/service/*)
    ↓
Core Layer (goagent/api/core)
    ↓
Internal Layer (goagent/internal/*)
```

### 各层职责

1. **Client Layer**：统一的客户端入口，管理所有服务
2. **Service Layer**：业务逻辑实现，编排各模块
3. **Core Layer**：核心接口定义，公共类型
4. **Internal Layer**：具体实现细节

## 配置说明

配置文件位于 `config/server.yaml`：

```yaml
llm:
  provider: "ollama"
  base_url: "http://localhost:11434"
  model: "llama3.2"

database:
  enabled: false  # 是否启用数据库持久化

memory:
  enabled: true
  session:
    max_history: 50

retrieval:
  enabled: true
  top_k: 10
  min_score: 0.4
```

## 与旧API的区别

| 特性 | 旧API | 新API |
|------|-------|-------|
| 导入路径 | `goagent/api` | `goagent/api/client` |
| 初始化 | 直接注入依赖 | 使用配置对象 |
| 服务访问 | `client.Agent()` | `client.Agent()` 返回error |
| 类型定义 | 各自包中 | 统一在 `goagent/api/core` |
| 错误处理 | 各自包的错误 | 统一错误类型 |

## 迁移指南

从旧API迁移到新API：

1. 更新导入路径
2. 使用新的配置方式
3. 更新类型引用（使用core包中的类型）
4. 更新错误处理

详细迁移指南请参考 [MIGRATION.md](../../api/MIGRATION.md)

## 常见问题

### Q: 必须立即迁移到新API吗？

A: 不是必须的。旧API仍然可用，但新API提供更好的架构和功能。

### Q: 新API支持所有旧API的功能吗？

A: 是的，并且提供了更多新功能。

### Q: 如何配置数据库持久化？

A: 在配置文件中设置 `database.enabled: true` 并配置数据库连接信息。

## 更多资源

- [API架构文档](../../api/ARCHITECTURE.md)
- [API使用指南](../../api/README.md)
- [迁移指南](../../api/MIGRATION.md)
- [编码规范](../../plan/code_rules.md)

## 贡献

欢迎提交Issue和Pull Request！