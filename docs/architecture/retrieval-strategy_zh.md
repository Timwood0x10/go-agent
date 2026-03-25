# 检索策略指南

## 概述

GoAgent Storage 模块提供两种检索策略，以适应不同的使用场景：

1. **简单检索** - 纯向量相似度搜索
2. **高级检索** - 多源混合搜索，包含高级特性

## 简单检索（推荐用于大多数场景）

### 配置

```go
req := &services.SearchRequest{
    Query:    question,
    TenantID: tenantID,
    TopK:     5,
    MinScore: 0.6,
    Plan: &services.RetrievalPlan{
        SearchKnowledge:     true,
        KnowledgeWeight:     1.0,
        EnableKeywordSearch: false, // 禁用关键词搜索，简化检索逻辑
        EnableTimeDecay:     false, // 禁用时间衰减，简化检索逻辑
        TopK:                5,
    },
}
```

### 使用场景

- ✅ **单一知识库**（仅知识库）
- ✅ **简单语义搜索**（RAG、问答）
- ✅ **文档相似度**（查找相似文档）
- ✅ **代码库搜索**（查找相似代码）

### 特性

- **性能**: 快速（单次向量搜索）
- **准确度**: 高（语义相似度准确）
- **复杂度**: 简单（配置最少）
- **资源消耗**: 低（无需额外计算）

### 分数计算

```
最终分数 = 原始余弦相似度
```

分数是 pgvector 的原始余弦相似度（范围：-1 到 1，通常相关结果为 0.6-0.9）。

### 示例结果

查询："RAG"

| 排名 | 相似度 | 内容 |
|------|--------|------|
| 1 | 0.79 | 配置参数说明（chunk size 和 overlap） |
| 2 | 0.72 | [RAG 最佳实践](https://docs.anthropic.com) |
| 3 | 0.68 | 混合检索流程文档 |

## 高级检索（用于复杂多源场景）

### 配置

```go
req := &services.SearchRequest{
    Query:    question,
    TenantID: tenantID,
    TopK:     10,
    MinScore: 0.5,
    Plan: &services.RetrievalPlan{
        SearchKnowledge:     true,
		SearchExperience:    true,
		SearchTools:         true,
		SearchTaskResults:   false,
		KnowledgeWeight:     0.4,
		ExperienceWeight:    0.3,
		ToolsWeight:         0.2,
		TaskResultsWeight:   0.1,
		EnableQueryRewrite:  true,
		EnableKeywordSearch: true,
		EnableTimeDecay:     true,
		TopK:                10,
    },
}
```

### 使用场景

- ✅ **多源检索**（知识库 + 经验 + 工具）
- ✅ **混合搜索**（向量 + 关键词/BM25）
- ✅ **查询重写**（语义扩展）
- ✅ **时效性数据**（优先最新信息）
- ✅ **复杂企业系统**（多数据源）

### 特性

- **性能**: 较慢（多次搜索 + 重排序）
- **准确度**: 高（多源、智能重排序）
- **复杂度**: 复杂（大量可配置特性）
- **资源消耗**: 较高（额外嵌入向量、计算）

### 分数计算

```
最终分数 = 原始相似度 × 查询权重 × 源权重 × 子源权重 × 时间衰减 × 信号
```

#### 分数组成部分

1. **原始相似度**: pgvector 的余弦相似度（0.6-0.9）
2. **查询权重**: 
   - 原始查询: 1.0
   - 规则重写: 0.7
   - LLM 重写: 0.5
3. **源权重**:
   - 知识库: 0.4
   - 经验: 0.3
   - 工具: 0.2
   - 任务结果: 0.1
4. **子源权重**:
   - 向量搜索: 1.0
   - 关键词搜索: 0.8
5. **时间衰减**: 
   - 指数衰减: `exp(-0.01 × 小时数)`
   - 最小值: 0.1
6. **源特定信号**:
   - 经验: 成功率 (1.2×), 执行时间 (0.8-1.2×)
   - 工具: 成功率 (0.8-1.1×), 需要认证 (0.9×)

### 示例结果

查询："如何配置 chunk size？"

| 排名 | 原始分数 | 最终分数 | 来源 | 内容 |
|------|---------|---------|------|------|
| 1 | 0.85 | 0.34 | 知识库 | Chunk size 配置指南 |
| 2 | 0.72 | 0.22 | 知识库 | 参数调优最佳实践 |
| 3 | 0.65 | 0.13 | 经验 | 之前的配置问题 |
| 4 | 0.58 | 0.12 | 知识库 | 高级配置选项 |

## 性能对比

| 指标 | 简单检索 | 高级检索 |
|------|---------|---------|
| 查询时间 | ~50ms | ~200-500ms |
| 内存使用 | 低 | 中等 |
| CPU 使用 | 低 | 中-高 |
| 检索质量 | 高（单源） | 高（多源）|

## 选择合适的策略

### 使用简单检索，如果：

1. 只需要搜索一个数据源（如知识库）
2. 需要最大性能
3. 需要直接的语义相似度
4. 有单一用途的应用（如文档问答）
5. 数据不需要基于时间的优先级

### 使用高级检索，如果：

1. 需要搜索多个数据源（知识库 + 经验 + 工具）
2. 需要关键词匹配 + 语义搜索
3. 需要查询扩展/重写
4. 有时效性数据需要优先级排序
5. 构建复杂的企业系统，有多种检索需求

## 配置示例

### 示例 1: RAG 系统（简单）

```go
req := &services.SearchRequest{
    Query:    "什么是 RAG？",
    TenantID: "user-123",
    TopK:     5,
    MinScore: 0.6,
    Plan: &services.RetrievalPlan{
        SearchKnowledge:     true,
        KnowledgeWeight:     1.0,
        EnableKeywordSearch: false,
        EnableTimeDecay:     false,
        TopK:                5,
    },
}
```

### 示例 2: 多源企业系统（高级）

```go
req := &services.SearchRequest{
    Query:    "如何使用 storage 模块？",
    TenantID: "company-xyz",
    TopK:     10,
    MinScore: 0.5,
    Plan: &services.RetrievalPlan{
        SearchKnowledge:     true,
        SearchExperience:    true,
        SearchTools:         true,
        SearchTaskResults:   true,
        KnowledgeWeight:     0.5,
        ExperienceWeight:    0.3,
        ToolsWeight:         0.15,
		TaskResultsWeight:   0.05,
        EnableQueryRewrite:  true,
        EnableKeywordSearch: true,
        EnableTimeDecay:     true,
        TopK:                10,
    },
}
```

### 示例 3: 文档相似度（简单）

```go
req := &services.SearchRequest{
    Query:    "和我上一个问题相似",
    TenantID: "user-456",
    TopK:     3,
    MinScore: 0.7,
    Plan: &services.RetrievalPlan{
        SearchKnowledge:     true,
        KnowledgeWeight:     1.0,
        EnableKeywordSearch: false,
        EnableTimeDecay:     false,
        TopK:                3,
    },
}
```

## 调优建议

### 简单检索

1. **MinScore（最小分数）**:
   - 0.7-0.8: 高精确度，结果较少
   - 0.6-0.7: 平衡精确度和召回率（推荐）
   - 0.5-0.6: 更多结果，精确度较低

2. **TopK（返回结果数）**:
   - 3-5: 精准答案
   - 5-10: 全面答案（推荐）
   - 10+: 广泛探索

3. **Chunk Size（文档分块大小）**:
   - 200-300: 高精确度，可能丢失上下文
   - 500-700: 平衡精确度和上下文（推荐）
   - 1000+: 更多上下文，精确度较低

### 高级检索

1. **源权重**: 根据数据重要性调整
   - 提高 KnowledgeWeight 如果文档是主要来源
   - 提高 ExperienceWeight 如果过去的解决方案有价值
   - 提高 ToolsWeight 如果工具推荐是关键

2. **查询重写**: 对模糊查询启用
   - 提升语义理解
   - 可以提高召回率，但会增加延迟

3. **时间衰减**: 对频繁更新的内容启用
   - Lambda 0.01 是默认值
   - 调整 lambda 实现更快/更慢的衰减

## 迁移指南

### 从高级到简单

如果发现高级检索太复杂或太慢，简化配置：

```go
// 之前（高级）
Plan: &services.RetrievalPlan{
    SearchKnowledge:     true,
    KnowledgeWeight:     0.4,
    EnableKeywordSearch: true,
    EnableTimeDecay:     true,
    TopK:                10,
}

// 之后（简单）
Plan: &services.RetrievalPlan{
    SearchKnowledge:     true,
    KnowledgeWeight:     1.0,
    EnableKeywordSearch: false,
    EnableTimeDecay:     false,
    TopK:                5,
}
```

### 性能影响

从高级检索切换到简单检索：

- ⚡ **快 4-10 倍**（50ms → 200-500ms）
- 💾 **少用 30% 内存**（无需处理关键词结果）
- 🔍 **结果更简洁**（纯语义相似度）

## 最佳实践

1. **从简单开始**: 先用简单检索，根据需要逐步添加功能
2. **性能监控**: 监控查询时间和结果质量
3. **A/B 测试**: 用你的数据测试两种策略
4. **逐步调整**: 一次只调整一个参数
5. **监控分数**: 确保 MinScore 阈值设置合理

## 故障排除

### 问题: 没有返回结果

**简单检索**:
- 检查 MinScore 是否过高（试试 0.5）
- 确认文档已索引
- 检查 embedding 模型是否正常工作

**高级检索**:
- 检查所有源权重不为 0
- 确认至少启用了一个源
- 检查所有权重计算后的 MinScore

### 问题: 结果不相关

**简单检索**:
- 提高 MinScore（0.5 → 0.7）
- 检查 chunk size（试试更小的块）
- 验证 embedding 模型质量

**高级检索**:
- 调整源权重
- 禁用不必要的特性（时间衰减、查询重写）
- 检查关键词搜索是否引入噪声

### 问题: 性能慢

**简单检索**:
- 减少 TopK（10 → 5）
- 检查数据库索引
- 验证 pgvector 已优化

**高级检索**:
- 禁用查询重写
- 禁用关键词搜索
- 减少启用的源数量
- 减少 TopK

## 参考资料

- [pgvector 文档](https://github.com/pgvector/pgvector)
- [ChromaDB 最佳实践](https://docs.trychroma.com/guides)
- [RAG 最佳实践](https://docs.anthropic.com/claude/docs/retrieval-augmented-generation)
- [向量搜索优化](https://www.pinecone.io/learn/vector-search-optimization/)