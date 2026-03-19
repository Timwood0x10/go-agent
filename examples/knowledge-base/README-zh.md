# Local Knowledge Base Example

这是一个基于goagent storage模块的本地知识库示例程序。它展示了如何使用storage模块的高级API快速构建一个功能完整的文档检索和问答系统。

## 功能特性

- 📄 **文档导入**: 支持导入文本文档，自动分块、向量化、存储
- 🔍 **智能检索**: 向量检索 + BM25全文检索混合搜索
- 💬 **交互问答**: 命令行交互式知识问答
- 📊 **文档管理**: 列出、删除已导入的文档
- 🏢 **多租户隔离**: 支持多个独立的租户空间
- ⚡ **高性能**: 基于pgvector的高效向量检索

## 系统要求

### 必需组件

1. **PostgreSQL 16 + pgvector扩展**
   ```bash
   # 使用Docker启动PostgreSQL
   docker run -d \
     --name postgres-pgvector \
     -p 5433:5432 \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=goagent \
     pgvector/pgvector:pg16
   ```

2. **Ollama嵌入服务**
   ```bash
   # 安装Ollama
   curl -fsSL https://ollama.com/install.sh | sh

   # 拉取嵌入模型
   ollama pull nomic-embed-text

   # 启动Ollama服务
   ollama serve
   ```

### 验证安装

```bash
# 检查PostgreSQL
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT * FROM pg_extension WHERE extname='vector';"

# 检查Ollama
curl http://localhost:11434/api/tags
```

## 快速开始


### 特例，一键启动
```shell

./servies/embedding/start.sh // 启动embedding 服务

go run main.go --save ./example.md

go run main.go --chat
```

### 不嫌麻烦折腾流

#### 1. 配置数据库

确保PostgreSQL数据库已启动并配置正确：

```bash
# 检查数据库连接
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT version();"
```

#### 2. 配置应用

编辑 `config.yaml` 文件，确认数据库和嵌入服务配置：

```yaml
database:
  host: localhost
  port: 5433
  user: postgres
  password: postgres
  database: goagent

embedding_service_url: http://localhost:11434
embedding_model: nomic-embed-text
```

#### 3. 导入文档

```bash
# 导入一个文档
go run main.go --save README.md

# 导入其他文档
go run main.go --save ../../docs/storage/api.md
go run main.go --save ../../README.md
```

#### 4. 开始问答

```bash
# 启动交互式问答
go run main.go --chat
```

然后输入你的问题：

```
You: 什么是RAG？

Found 3 relevant results:

[1] Score: 0.892
Content: RAG（Retrieval-Augmented Generation）是一种结合检索和生成的AI系统架构...
Source: README.md

[2] Score: 0.856
Content: Storage模块支持向量检索和BM25全文检索的混合搜索...
Source: api.md

[3] Score: 0.743
Content: 向量检索使用pgvector实现高性能的相似度搜索...
Source: api.md

You: storage模块有哪些功能？

Found 2 relevant results:

[1] Score: 0.934
Content: Storage模块提供向量存储、检索、多租户隔离、混合检索等功能...
Source: api.md

[2] Score: 0.887
Content: 核心能力包括：向量存储与检索、多租户隔离、混合检索、智能缓存等...
Source: README.md

You: exit
Goodbye!
```

#### 5. 管理文档

```bash
# 列出所有文档
go run main.go --list

# 输出示例：
# Documents:
#   - ID: 1234567890abcdef, Source: README.md, Chunks: 12
#   - ID: abcdef1234567890, Source: api.md, Chunks: 45

# 删除指定文档
go run main.go --delete 1234567890abcdef
```

## 使用指南

### 命令行选项

```bash
go run main.go [options]

选项:
  --save <path>     导入文档到知识库
  --chat            启动交互式问答模式
  --list            列出所有已导入的文档
  --delete <id>     删除指定文档
  --tenant <id>     指定租户ID (默认: default)
  --config <path>   配置文件路径 (默认: config.yaml)
```

### 多租户使用

```bash
# 为不同用户/项目创建独立的知识库空间
go run main.go --save user1_doc.pdf --tenant user1
go run main.go --save user2_doc.pdf --tenant user2

# 每个租户只能看到自己的文档
go run main.go --list --tenant user1
go run main.go --chat --tenant user2
```

### 配置调优

编辑 `config.yaml` 优化检索效果：

```yaml
knowledge:
  chunk_size: 500          # 较小的chunk提高精确度
  chunk_overlap: 50        # 保持上下文连贯性
  top_k: 5                 # 返回更多候选结果
  min_score: 0.6           # 提高相似度阈值
```

**参数说明:**

- `chunk_size`: 文档分块大小
  - 小值 (200-300): 更精确，但可能丢失上下文
  - 中值 (500-700): 平衡精确度和上下文（推荐）
  - 大值 (1000+): 更多上下文，但精确度降低

- `chunk_overlap`: 分块重叠大小
  - 通常设置为chunk_size的10-20%
  - 有助于保持语义连贯性

- `top_k`: 检索返回结果数
  - 3-5: 精确答案
  - 5-10: 综合答案（推荐）
  - 10+: 广泛探索

- `min_score`: 最小相似度阈值
  - 0.7-0.8: 高相关性，结果少
  - 0.6-0.7: 平衡相关性和结果数量（推荐）
  - 0.5-0.6: 更多结果，可能包含不相关内容

## 工作原理

### 导入流程

```
文档读取 → 智能分块 → 生成嵌入向量 → 存储到PostgreSQL + pgvector
```

1. **文档读取**: 读取文档内容
2. **智能分块**: 按配置的大小和重叠进行分块
3. **生成嵌入**: 使用Ollama服务为每个chunk生成1024维向量
4. **向量存储**: 存储到PostgreSQL的pgvector表中

### 检索流程

```
用户问题 → 向量化 → 混合检索 → 结果排序 → 返回相关内容
```

1. **问题向量化**: 将用户问题转换为向量
2. **混合检索**: 同时执行向量检索和BM25检索
3. **结果排序**: 使用RRF算法合并和排序结果
4. **返回结果**: 返回TopK个最相关的知识块

## 高级用法

### 批量导入

```bash
# 批量导入多个文档
for file in docs/*.md; do
  go run main.go --save "$file" --tenant default
done
```

### 自定义分块

修改 `main.go` 中的 `chunkDocument` 方法实现自定义分块逻辑：

```go
func (kb *KnowledgeBase) chunkDocument(content string, chunkSize, chunkOverlap int) []*Chunk {
    // 实现自定义分块逻辑
    // - 按段落分块
    // - 按语义分块
    // - 按章节分块
}
```

### 集成LLM

扩展 `StartChat` 方法，集成LLM服务生成答案：

```go
func (kb *KnowledgeBase) StartChat(ctx context.Context, tenantID string) {
    // ... 检索逻辑 ...

    // 调用LLM生成答案
    answer := callLLM(question, results)
    fmt.Printf("\nAI: %s\n", answer)
}
```

## 架构说明

### 核心组件

```
KnowledgeBase (高级API)
    ├── Pool (数据库连接池)
    ├── KnowledgeRepository (知识库数据访问)
    ├── RetrievalService (智能检索)
    ├── EmbeddingClient (嵌入服务)
    ├── TenantGuard (租户隔离)
    └── RetrievalGuard (限流熔断)
```

### 数据流

**导入文档:**
```
文档 → 分块 → 嵌入向量 → PostgreSQL + pgvector
```

**知识问答:**
```
问题 → 检索请求 → 混合检索 → 结果排序 → 返回相关内容
```

## 性能优化

### 1. 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_knowledge_tenant_id ON knowledge_chunks_1024(tenant_id);
CREATE INDEX idx_knowledge_document_id ON knowledge_chunks_1024(document_id);
CREATE INDEX idx_knowledge_embedding_status ON knowledge_chunks_1024(embedding_status);
```

### 2. 缓存配置

```yaml
embedding_service_url: http://localhost:11434
embedding_model: nomic-embed-text
```

### 3. 批量处理

导入大量文档时，建议分批处理：

```bash
# 每批导入10个文档
find docs/ -name "*.md" | head -n 10 | xargs -I {} go run main.go --save "{}"
```

## 故障排查

### 问题1: 数据库连接失败

```
Error: create database pool: connection refused
```

**解决方案:**
- 检查PostgreSQL是否运行: `docker ps | grep postgres`
- 检查端口是否正确: `netstat -an | grep 5433`
- 检查配置文件中的数据库配置

### 问题2: 嵌入服务不可用

```
Error: Failed to embed chunk: connection refused
```

**解决方案:**
- 检查Ollama是否运行: `ps aux | grep ollama`
- 检查模型是否已下载: `ollama list`
- 重启Ollama服务: `ollama serve`

### 问题3: 导入超时

```
Import timeout (5 minutes exceeded)
```

**解决方案:**
- 检查嵌入服务响应速度
- 减小文档大小或增加分块数量
- 检查网络连接
- 查看具体哪个chunk超时（日志会显示）

### 问题4: 检索超时

```
Search timeout. Please try again.
```

**解决方案:**
- 检查数据库连接状态
- 检查嵌入服务是否正常
- 减少检索结果数量（降低top_k值）
- 检查是否有大量并发请求

### 问题5: 程序卡死

**症状**: 程序无响应，无法退出

**预防措施:**
- 所有操作都有超时保护（导入5分钟，检索30秒）
- 每个chunk独立超时（60秒）
- 使用Ctrl+C可以中断程序
- 输入使用非阻塞IO（bufio.Scanner）

**解决方案:**
- 按Ctrl+C中断程序
- 检查Ollama服务是否卡死: `curl http://localhost:11434/api/tags`
- 检查数据库是否卡死: `docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT 1;"`
- 重启相关服务

### 问题6: 检索结果不理想

**解决方案:**
- 调整 `chunk_size` 和 `chunk_overlap`
- 降低 `min_score` 阈值
- 增加 `top_k` 值
- 尝试不同的嵌入模型

### 问题7: pgvector未安装

```
Error: type "vector" does not exist
```

**解决方案:**
```bash
# 在PostgreSQL中安装pgvector扩展
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "CREATE EXTENSION vector;"
```

## 扩展功能

### 1. 支持更多文档格式

集成PDF、Word等文档解析库：

```go
import "github.com/unidoc/unipdf/v3/extractor"

func loadPDF(path string) (string, error) {
    // PDF解析逻辑
}
```

### 2. 添加文档元数据

为文档添加更多元信息：

```go
type DocumentMetadata struct {
    Title       string    `json:"title"`
    Author      string    `json:"author"`
    CreatedAt   time.Time `json:"created_at"`
    Tags        []string  `json:"tags"`
    Category    string    `json:"category"`
}
```

### 3. 实现文档版本管理

支持文档的版本控制和更新：

```go
func (kb *KnowledgeBase) UpdateDocument(ctx context.Context, tenantID, docID string) error {
    // 删除旧版本
    kb.DeleteDocument(ctx, tenantID, docID)
    // 导入新版本
    kb.ImportDocuments(ctx, tenantID, docPath)
}
```

## 技术栈

- **语言**: Go 1.21+
- **数据库**: PostgreSQL 16 + pgvector
- **嵌入服务**: Ollama (nomic-embed-text)
- **配置**: YAML
- **检索**: 向量相似度 + BM25 + RRF

## 参考资源

- [Storage API文档](../../docs/storage/api.md)
- [pgvector文档](https://github.com/pgvector/pgvector)
- [Ollama文档](https://github.com/ollama/ollama)
- [RAG最佳实践](https://docs.anthropic.com/claude/docs/retrieval-augmented-generation)

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！