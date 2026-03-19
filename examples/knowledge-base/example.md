# 示例文档 - GoAgent Storage模块

## 概述

GoAgent Storage模块是一个基于PostgreSQL + pgvector的高性能向量存储和检索系统，为AI应用提供强大的数据持久化能力。

## 核心功能

### 1. 向量存储与检索

Storage模块使用pgvector扩展实现高效的向量相似度搜索。它支持：

- **高维向量存储**: 支持1024维向量存储
- **余弦相似度**: 使用余弦相似度进行向量检索
- **批量操作**: 支持批量插入和检索
- **索引优化**: 自动创建向量索引提高查询性能

示例：
```go
// 创建知识块
chunk := &KnowledgeChunk{
    Content:   "这是知识内容",
    Embedding: []float64{0.1, 0.2, ...}, // 1024维向量
}

// 向量检索
results := kbRepo.SearchByVector(ctx, queryEmbedding, tenantID, 10)
```

### 2. 多租户隔离

通过RLS（Row Level Security）和Tenant Guard实现严格的多租户数据隔离：

- **自动隔离**: 所有操作自动应用租户过滤
- **双重保护**: 数据库RLS + 应用层Tenant Guard
- **安全验证**: 验证所有跨租户访问

示例：
```go
// 设置租户上下文
tenantGuard.SetTenantContext(ctx, "tenant-001")

// 后续所有操作自动隔离
results := kbRepo.SearchByVector(ctx, embedding, "tenant-001", limit)
```

### 3. 混合检索

结合向量检索和BM25全文检索，提高检索准确性：

- **向量检索**: 基于语义相似度
- **BM25检索**: 基于关键词匹配
- **结果融合**: 使用RRF算法合并结果
- **时间衰减**: 新知识优先

示例：
```go
req := &SearchRequest{
    Query: "用户问题",
    Plan: &RetrievalPlan{
        SearchKnowledge:     true,
        EnableKeywordSearch: true,
        EnableTimeDecay:     true,
    },
}

results := retrievalService.Search(ctx, req)
```

### 4. 智能缓存

提供多级缓存提高性能：

- **嵌入缓存**: 缓存生成的嵌入向量
- **结果缓存**: 缓存检索结果
- **自动过期**: 支持TTL自动过期
- **降级策略**: 缓存失败不影响服务

### 5. 安全加密

敏感数据使用AES-256-GCM加密存储：

- **密钥管理**: 专门的Secret Repository
- **自动加密**: 自动加密敏感字段
- **密钥轮换**: 支持定期轮换密钥

## 架构设计

### 分层架构

Storage模块采用清晰的分层架构：

```
应用层 (Application)
    ↓
服务层 (Services)
    ↓
数据访问层 (Repositories)
    ↓
核心层 (Core)
    ↓
PostgreSQL + pgvector
```

### 核心组件

1. **Pool**: 数据库连接池管理
2. **TenantGuard**: 租户隔离守卫
3. **RetrievalGuard**: 检索限流熔断
4. **Repository**: 统一数据访问接口
5. **Service**: 业务逻辑服务

## 使用场景

### 1. RAG系统

Storage模块非常适合构建RAG（Retrieval-Augmented Generation）系统：

```
用户问题 → 向量化 → 检索知识库 → 构建上下文 → LLM生成答案
```

### 2. 语义搜索

为应用添加智能语义搜索功能：

```go
// 语义搜索
results := kbRepo.SearchByVector(ctx, queryEmbedding, tenantID, 10)

// 混合搜索
results := retrievalService.Search(ctx, req)
```

### 3. 推荐系统

基于向量相似度的内容推荐：

```go
// 找到相似内容
similarItems := kbRepo.SearchByVector(ctx, itemEmbedding, tenantID, 5)
```

### 4. 知识库管理

构建企业知识库或文档管理系统：

```go
// 导入文档
kb.ImportDocuments(ctx, tenantID, docPath)

// 知识问答
answer := kb.Chat(ctx, tenantID, question)
```

## 性能优化

### 1. 索引优化

确保关键字段有索引：

```sql
CREATE INDEX idx_tenant_id ON knowledge_chunks_1024(tenant_id);
CREATE INDEX idx_document_id ON knowledge_chunks_1024(document_id);
CREATE INDEX idx_embedding_status ON knowledge_chunks_1024(embedding_status);
```

### 2. 批量操作

使用批量接口提高性能：

```go
// 批量插入
kbRepo.CreateBatch(ctx, chunks)

// 批量检索
embeddings := embeddingClient.EmbedBatch(ctx, texts)
```

### 3. 连接池配置

优化连接池参数：

```go
config := &postgres.Config{
    MaxOpenConns:    25,
    MaxIdleConns:    10,
    ConnMaxLifetime: 5 * time.Minute,
}
```

## 最佳实践

### 1. 文档分块

- **chunk_size**: 500-700字符
- **chunk_overlap**: 50-100字符
- **按段落分块**: 保持语义完整性

### 2. 检索参数

- **top_k**: 5-10个结果
- **min_score**: 0.6-0.7
- **启用时间衰减**: 优先新知识

### 3. 租户管理

- **独立租户**: 每个用户/项目使用独立租户
- **验证访问**: 验证所有跨租户操作
- **定期清理**: 清理不活跃租户数据

## 总结

GoAgent Storage模块提供了：

✅ 高性能的向量存储和检索
✅ 严格的多租户隔离
✅ 智能的混合检索
✅ 完善的缓存机制
✅ 强大的安全加密

它是构建AI应用的理想数据持久化解决方案！