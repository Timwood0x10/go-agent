# Repositories 模块 API 文档

## 概述

Repositories 模块提供了数据访问层（DAL），负责与 PostgreSQL 数据库的交互。该模块实现了各种 Repository 接口，提供 CRUD 操作和高级查询功能。

## 核心特性

- **数据库抽象**：使用 `DBTX` 接口支持数据库连接和事务
- **向量搜索**：使用 pgvector 扩展实现语义搜索
- **全文搜索**：支持 BM25 排序的关键词搜索
- **租户隔离**：所有操作都支持多租户隔离
- **错误处理**：统一的错误处理和返回
- **事务支持**：支持原子性操作和批量处理

## 可用的 Repository

### 1. ConversationRepository

对话历史数据访问层，提供会话消息的存储和检索。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Create(ctx, conv)` | 创建新的对话消息 |
| `GetByID(ctx, id)` | 根据 ID 获取对话消息 |
| `GetBySession(ctx, sessionID, tenantID, limit)` | 获取指定会话的所有消息 |
| `DeleteBySession(ctx, sessionID, tenantID)` | 删除指定会话的所有消息 |
| `Delete(ctx, id)` | 删除单个对话消息 |
| `GetByUser(ctx, userID, tenantID, limit)` | 获取用户的最近消息 |
| `GetByAgent(ctx, agentID, tenantID, limit)` | 获取代理的最近消息 |
| `CleanupExpired(ctx)` | 清理已过期的对话消息 |
| `UpdateExpiresAt(ctx, sessionID, tenantID, expiresAt)` | 更新会话过期时间 |
| `CountBySession(ctx, sessionID, tenantID)` | 统计会话中的消息数量 |
| `GetRecentSessions(ctx, tenantID, limit)` | 获取最近的会话列表 |

#### 使用示例

```go
repo := repositories.NewConversationRepository(db)

// 创建对话消息
conv := &storage_models.Conversation{
    SessionID: "session-1",
    TenantID:  "tenant-1",
    UserID:    "user-1",
    AgentID:   "agent-1",
    Role:      "user",
    Content:   "Hello, how can I help you?",
    CreatedAt: time.Now(),
}
err := repo.Create(ctx, conv)

// 获取会话消息
messages, err := repo.GetBySession(ctx, "session-1", "tenant-1", 100)
```

### 2. TaskResultRepository

任务结果数据访问层，提供任务执行结果的存储和检索。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Create(ctx, result)` | 创建新的任务结果 |
| `GetByID(ctx, id)` | 根据 ID 获取任务结果 |
| `GetBySession(ctx, sessionID, tenantID, limit)` | 获取指定会话的任务结果 |
| `GetByAgent(ctx, agentID, tenantID, limit)` | 获取指定代理的任务结果 |
| `GetByStatus(ctx, status, tenantID, limit)` | 根据状态获取任务结果 |
| `Update(ctx, result)` | 更新任务结果 |
| `Delete(ctx, id)` | 删除任务结果 |
| `DeleteBySession(ctx, sessionID, tenantID)` | 删除指定会话的所有任务结果 |
| `SearchByVector(ctx, embedding, tenantID, limit)` | 向量相似性搜索 |
| `SearchByKeyword(ctx, query, tenantID, limit)` | 关键词搜索 |
| `GetStatsByAgent(ctx, agentID, tenantID)` | 获取代理的统计信息 |
| `GetStatsByTenant(ctx, tenantID)` | 获取租户的统计信息 |

#### 使用示例

```go
repo := repositories.NewTaskResultRepository(db)

// 创建任务结果
result := &storage_models.TaskResult{
    SessionID: "session-1",
    TenantID:  "tenant-1",
    TaskType:  "chat",
    AgentID:   "agent-1",
    Input:     map[string]interface{}{"query": "test"},
    Status:    "completed",
    CreatedAt: time.Now(),
}
err := repo.Create(ctx, result)
```

### 3. ToolRepository

工具定义数据访问层，提供工具的存储和检索，支持语义搜索。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Create(ctx, tool)` | 创建新的工具 |
| `GetByID(ctx, id)` | 根据 ID 获取工具 |
| `GetByName(ctx, name, tenantID)` | 根据名称获取工具 |
| `Update(ctx, tool)` | 更新工具 |
| `Delete(ctx, id)` | 删除工具 |
| `SearchByVector(ctx, embedding, tenantID, limit)` | 向量相似性搜索 |
| `SearchByKeyword(ctx, query, tenantID, limit)` | 关键词搜索 |
| `ListAll(ctx, tenantID, limit)` | 列出所有工具 |
| `ListByAgentType(ctx, agentType, tenantID, limit)` | 根据代理类型列出工具 |
| `ListByTags(ctx, tags, tenantID, limit)` | 根据标签列出工具 |
| `UpdateUsage(ctx, id, success)` | 更新工具使用统计 |
| `UpdateEmbedding(ctx, id, embedding, model, version)` | 更新工具嵌入 |

#### 使用示例

```go
repo := repositories.NewToolRepository(db)

// 创建工具
tool := &storage_models.Tool{
    TenantID:         "tenant-1",
    Name:             "web_search",
    Description:      "Search the web for information",
    Embedding:        createTestEmbedding(),
    EmbeddingModel:   "e5-large",
    EmbeddingVersion: 1,
    Tags:             []string{"search", "web"},
    CreatedAt:        time.Now(),
}
err := repo.Create(ctx, tool)

// 向量搜索
similarTools, err := repo.SearchByVector(ctx, queryEmbedding, "tenant-1", 10)
```

### 4. KnowledgeRepository

知识库数据访问层，提供知识块的存储和检索，支持 RAG（检索增强生成）。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Create(ctx, chunk)` | 创建新的知识块 |
| `CreateBatch(ctx, chunks)` | 批量创建知识块（事务支持） |
| `GetByID(ctx, id)` | 根据 ID 获取知识块 |
| `Update(ctx, chunk)` | 更新知识块 |
| `Delete(ctx, id)` | 删除知识块 |
| `SearchByVector(ctx, embedding, tenantID, limit)` | 向量相似性搜索 |
| `SearchByKeyword(ctx, query, tenantID, limit)` | 关键词搜索（BM25） |
| `ListByDocument(ctx, documentID, tenantID)` | 列出指定文档的所有知识块 |
| `UpdateEmbedding(ctx, id, embedding, model, version)` | 更新知识块嵌入 |
| `UpdateEmbeddingStatus(ctx, id, status, errorMsg)` | 更新嵌入处理状态 |
| `CleanupExpired(ctx, olderThan)` | 清理过期的知识块 |

#### 使用示例

```go
repo := repositories.NewKnowledgeRepository(db, dbPool)

// 创建知识块
chunk := &storage_models.KnowledgeChunk{
    TenantID:         "tenant-1",
    Content:          "This is a knowledge chunk about AI",
    Embedding:        createTestEmbedding(),
    EmbeddingModel:   "e5-large",
    EmbeddingVersion: 1,
    EmbeddingStatus:  storage_models.EmbeddingStatusCompleted,
    SourceType:       "document",
    DocumentID:       "doc-123",
    ContentHash:      "hash-abc",
    CreatedAt:        time.Now(),
}
err := repo.Create(ctx, chunk)

// 向量搜索
similarChunks, err := repo.SearchByVector(ctx, queryEmbedding, "tenant-1", 5)
```

### 5. ExperienceRepository

经验库数据访问层，存储和管理代理执行经验。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Create(ctx, experience)` | 创建新的经验记录 |
| `GetByID(ctx, id)` | 根据 ID 获取经验记录 |
| `Update(ctx, experience)` | 更新经验记录 |
| `Delete(ctx, id)` | 删除经验记录 |
| `ListByAgent(ctx, agentID, tenantID, limit)` | 列出指定代理的经验 |
| `ListByTaskType(ctx, taskType, tenantID, limit)` | 根据任务类型列出经验 |
| `SearchByVector(ctx, embedding, tenantID, limit)` | 向量相似性搜索 |
| `GetSuccessRate(ctx, agentID, tenantID)` | 获取成功率统计 |
| `UpdateEmbedding(ctx, id, embedding, model, version)` | 更新经验嵌入 |

#### 使用示例

```go
repo := repositories.NewExperienceRepository(db)

// 创建经验记录
experience := &storage_models.Experience{
    TenantID:         "tenant-1",
    AgentID:          "agent-1",
    TaskType:         "chat",
    TaskInput:        map[string]interface{}{"query": "test"},
    TaskOutput:       map[string]interface{}{"response": "test response"},
    Success:          true,
    Embedding:        createTestEmbedding(),
    EmbeddingModel:   "e5-large",
    EmbeddingVersion: 1,
    CreatedAt:        time.Now(),
}
err := repo.Create(ctx, experience)
```

### 6. SecretRepository

密钥管理数据访问层，提供加密的敏感数据存储。

#### 主要方法

| 方法 | 描述 |
|------|------|
| `Set(ctx, key, value, tenantID)` | 存储密钥（加密） |
| `Get(ctx, key, tenantID)` | 获取密钥（解密） |
| `Delete(ctx, key, tenantID)` | 删除密钥 |
| `List(ctx, tenantID)` | 列出所有密钥（不含值） |
| `SetWithExpiration(ctx, key, value, tenantID, expiresAt)` | 存储带过期时间的密钥 |
| `UpdateMetadata(ctx, key, tenantID, metadata)` | 更新密钥元数据 |
| `CleanupExpired(ctx)` | 清理过期的密钥 |
| `RotateKey(ctx, newKey)` | 轮换加密密钥 |
| `Export(ctx, tenantID)` | 导出密钥（备份） |
| `Import(ctx, tenantID, data, format)` | 导入密钥（恢复） |
| `GetKeyVersion(ctx, key, tenantID)` | 获取密钥版本 |

#### 使用示例

```go
encryptionKey := make([]byte, 32) // 32 bytes for AES-256-GCM
repo := repositories.NewSecretRepository(db, encryptionKey)

// 存储密钥
err := repo.Set(ctx, "api_key", "sk-1234567890", "tenant-1")

// 获取密钥
value, err := repo.Get(ctx, "api_key", "tenant-1")

// 存储带过期时间的密钥
expiresAt := time.Now().Add(30 * 24 * time.Hour)
err = repo.SetWithExpiration(ctx, "temp_key", "temp-value", "tenant-1", expiresAt)
```

## 错误处理

所有 Repository 方法都返回标准错误类型：

- `errors.ErrInvalidArgument`：无效参数
- `errors.ErrRecordNotFound`：记录未找到
- `errors.ErrNoTransaction`：需要事务但不可用
- `errors.ErrSecretExpired`：密钥已过期

## 测试覆盖

当前测试覆盖率：75.0%

测试覆盖包括：
- 正常路径测试
- 边界条件测试
- 错误路径测试
- 并发操作测试
- 租户隔离测试

## 性能考虑

- 使用预编译语句防止 SQL 注入
- 支持批量操作减少数据库往返
- 使用索引优化查询性能
- 支持连接池和事务
- 向量搜索使用 pgvector 优化

## 安全性

- 所有数据库操作都支持上下文取消
- 密钥使用 AES-256-GCM 加密
- 支持租户隔离
- 输入验证和参数化查询
- 定期清理过期数据

## 未来计划

- [ ] 添加缓存层支持
- [ ] 实现读写分离
- [ ] 支持更多向量搜索算法
- [ ] 添加性能监控和日志
- [ ] 支持数据迁移和版本管理