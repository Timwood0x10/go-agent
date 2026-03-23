# 本地知识库示例

这是一个基于goagent存储模块的本地知识库示例。它演示了如何使用存储模块的高级API快速构建一个功能完整的文档检索和问答系统。

## 功能特性

### 核心功能
- 📄 **文档导入**: 导入文本文档，自动分块、向量化和存储
- 🔍 **智能检索**: 混合检索，结合向量检索和BM25全文检索
- 💬 **交互式问答**: 命令行交互式知识问答
- 📊 **文档管理**: 列出和删除导入的文档
- 🏢 **多租户隔离**: 支持多个独立的租户空间
- ⚡ **高性能**: 基于pgvector的高效向量检索

### 高级功能
- 🎯 **精确模式**: 自动检测和处理精确查询（短查询、特殊符号如 `=+-*/:`）
- 🤖 **完整RAG流程**: 检索 → 生成 → 验证，使用本地LLM（Ollama）
- 🧠 **记忆系统**: 对话历史跟踪和会话管理
- 💾 **记忆蒸馏**: 达到阈值后自动提取和存储对话知识
- 🔬 **事实核查**: 用知识库中的事实信息纠正用户的误解
- 🎨 **智能RAG检测**: 自动确定每个问题是否需要RAG
- 🏠 **本地LLM集成**: 使用Ollama（llama3.2:latest）实现完整的本地设置，保护隐私和速度
- 📝 **知识纠正**: 检测纠正请求（"纠正"、"改正"、"修正"）并搜索相关内容进行更新
- 👤 **自我介绍检测**: 识别用户介绍（"我是XXX"、"我叫XXX"）并存储用户画像
- 💭 **跨会话记忆**: 在新对话中检索用户的偏好和画像

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

2. **Ollama服务**（同时用于嵌入和LLM）
   ```bash
   # 安装Ollama
   curl -fsSL https://ollama.com/install.sh | sh

   # 拉取嵌入模型
   ollama pull qwen3-embedding:0.6b

   # 拉取LLM模型用于答案生成
   ollama pull llama3.2:latest

   # 启动Ollama服务
   ollama serve
   ```

3. **嵌入服务**（可选，可以直接使用Ollama）
   ```bash
   cd services/embedding
   
   ./start.sh
   ```

### 验证安装

```bash
# 检查PostgreSQL
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT * FROM pg_extension WHERE extname='vector';"

# 检查Ollama
curl http://localhost:11434/api/tags
```

## 快速开始

### 前置条件

1. **PostgreSQL + pgvector正在运行**
   ```bash
   docker run -d \
     --name postgres-pgvector \
     -p 5433:5432 \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=goagent \
     pgvector/pgvector:pg16
   ```

2. **Ollama正在运行并已加载所需模型**
   ```bash
   # 启动Ollama
   ollama serve
   
   # 拉取模型（在另一个终端）
   ollama pull qwen3-embedding:0.6b  # 用于嵌入
   ollama pull llama3.2:latest        # 用于答案生成
   ```

3. **嵌入服务正在运行**（可选，可以直接使用Ollama）
   ```bash
   cd services/embedding
   PORT=8000 python3.14 app.py
   ```

### 一键启动

```bash
# 1. 启动嵌入服务
cd services/embedding
./start.sh

# 2. 导入文档
cd ../../examples/knowledge-base
go run main.go --save ../../plan/code_rules.md

# 3. 启动交互式问答
go run main.go --chat
```

### 详细设置

#### 1. 配置数据库

确保PostgreSQL正在运行并正确配置：

```bash
# 检查数据库连接
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT version();"
```

#### 2. 配置应用

编辑`config.yaml`文件以确认数据库、嵌入服务和LLM配置：

```yaml
database:
  host: localhost
  port: 5433
  user: postgres
  password: postgres
  database: goagent

embedding_service_url: http://localhost:8000
embedding_model: qwen3-embedding:0.6b

# LLM配置用于答案生成
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: llama3.2:latest
  timeout: 120
  max_tokens: 2048

# 记忆系统配置
memory:
  enabled: true
  max_history: 10
  max_sessions: 100
  enable_distillation: true
  distillation_threshold: 3

# 知识库配置
knowledge:
  chunk_size: 200
  chunk_overlap: 50
  top_k: 10
  min_score: 0.4
```

#### 3. 导入文档

```bash
# 导入文档
go run main.go --save README.md

# 导入其他文档
go run main.go --save ../../docs/storage/api.md
go run main.go --save ../../README.md
```

#### 4. 启动问答

```bash
# 启动交互式问答
go run main.go --chat
```

然后输入你的问题：

#### 示例1：普通对话（无RAG）
```
You: 你好
Assistant: Hello! How can I help you today?

You: 我叫小明
Assistant: Hello Xiao Ming! Nice to meet you.
```

#### 示例2：知识检索（带RAG）
```
You: 什么是RAG？

Found 3 relevant results:

[1] Score: 0.892
Content: RAG (Retrieval-Augmented Generation) is an AI system architecture combining retrieval and generation...
Source: README.md

Assistant: RAG (Retrieval-Augmented Generation) is an AI system architecture that combines 
information retrieval with text generation. It works by first retrieving relevant documents from a 
knowledge base, then using those documents as context to generate more accurate and factual 
responses. [Document 1]
```

#### 示例3：精确模式（精确匹配）
```
You: chunk_size = 200

Precision mode triggered
Retrieving exact match...

Assistant: For your question about chunk_size, according to the knowledge base, the 
recommended chunk_size for optimal performance and semantic integrity is between 50-100 characters. 
This helps maintain context while ensuring precise matching. [Document 1]
```

#### 示例4：事实核查
```
You: Go 语言应该经常使用 panic 来处理错误，对吧？

Assistant: According to the provided knowledge base context, using panic to handle errors in 
Go is prohibited. panic() is only allowed for fatal startup failures, not for business logic. 
Therefore, Go requires using error interfaces to handle business logic errors instead of using 
panic. [Document 1]
```

#### 示例5：记忆蒸馏
```
You: 你好
Assistant: Hello! How can I help you?

You: 我叫小明
Assistant: Hello Xiao Ming! Nice to meet you.

You: 还记得我的名字吗？
🎯 [记忆蒸馏] 对话轮数达到阈值 (3/3)，触发记忆蒸馏...
🔄 [记忆蒸馏] 开始蒸馏会话...
✅ [记忆蒸馏] 蒸馏完成！

Assistant: Yes, I remember! Your name is Xiao Ming. [Based on distilled memory]
```

#### 示例6：知识纠正
```
You: go-agent 是什么？

Assistant: GoAgent是一个基于PostgreSQL + pgvector的高性能向量存储和检索系统...

You: 纠正一下，go-agent是通用ai agent开发框架，用go实现
🔧 [知识纠正] 检测到纠正请求

Assistant: 我检测到你想要纠正知识。搜索到了相关结果。纠正功能已记录，请继续对话。
```

#### 示例7：自我介绍检测
```
You: 我是小明，喜欢编程，精通rust和golang，不喜欢python
👤 [自我介绍检测] user_id=小明

Assistant: 你好 小明！我已经记录了你的信息。
```

#### 示例8：跨会话记忆检索
```
# 第一次对话：
You: 我是小明，精通rust和golang
# ... 蒸馏触发 ...

# 新对话：
You: 我是小明，我的技术栈是什么？
👤 [自我介绍检测] user_id=小明
💭 [记忆检索] 从蒸馏记忆中加载用户画像

Assistant: 根据你的历史记录，你的技术栈包括Rust和Golang。
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

Options:
  --save <path>     导入文档到知识库
  --chat            启动交互式问答模式
  --list            列出所有导入的文档
  --delete <id>     删除指定文档
  --tenant <id>     指定租户ID（默认：default）
  --config <path>   配置文件路径（默认：config.yaml）
```

### 配置选项

#### 数据库配置
```yaml
database:
  host: localhost
  port: 5433
  user: postgres
  password: postgres
  database: goagent
```

#### 嵌入配置
```yaml
embedding_service_url: http://localhost:8000
embedding_model: qwen3-embedding:0.6b
```

#### LLM配置
```yaml
llm:
  provider: ollama              # LLM提供商（ollama, openrouter）
  base_url: http://localhost:11434
  model: llama3.2:latest       # 用于答案生成的LLM模型
  timeout: 120                  # LLM生成超时（秒）
  max_tokens: 2048              # LLM响应中的最大token数
```

#### 记忆系统配置
```yaml
memory:
  enabled: true                  # 启用记忆系统
  max_history: 10               # 保留的最大对话轮次
  max_sessions: 100              # 存储的最大会话数
  enable_distillation: true     # 启用自动蒸馏
  distillation_threshold: 3     # 触发蒸馏前的消息数
```

**记忆系统功能：**
- 跟踪对话历史以提供上下文
- 达到阈值后自动蒸馏
- 将蒸馏记忆存储在知识库中
- 实现跨会话的对话连续性
- 检测用户自我介绍并存储画像
- 从蒸馏记忆中检索用户偏好

**意图检测功能：**
- 知识纠正：检测纠正关键词（"纠正"、"改正"、"修正"、"不对"、"不是"）
- 自我介绍：检测介绍模式（"我是XXX"、"我叫XXX"）
- 从蒸馏记忆中加载用户画像

#### 知识库配置
```yaml
knowledge:
  chunk_size: 200              # 文档分块大小（字符）
  chunk_overlap: 50            # 分块重叠大小（字符）
  top_k: 10                     # 检索结果数量
  min_score: 0.4                # 最小相似度阈值
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

### 事实核查

系统可以自动检测并纠正用户的误解：

```bash
# 启动聊天模式
go run main.go --chat

# 示例：
You: Go 语言应该经常使用 panic 来处理错误，对吧？

# 系统将：
# 1. 检测到错误的假设
# 2. 从知识库中检索事实信息
# 3. 使用事实生成纠正的答案

Assistant: According to the provided knowledge base context, using panic to handle 
errors in Go is prohibited. panic() is only allowed for fatal startup failures, not for 
business logic. Therefore, Go requires using error interfaces to handle business logic errors 
instead of using panic.
```

### 批量导入

```bash
# 批量导入多个文档
for file in docs/*.md; do
  go run main.go --save "$file" --tenant default
done
```

### 检查蒸馏记忆

```bash
# 运行基于Go的蒸馏检查器
go run cmd/check_distillation/main.go

# 或构建后运行
go build -o check_distillation cmd/check_distillation/main.go
./check_distillation
```

## 工作原理

### 导入流程

```
文档读取 → 智能分块 → 生成嵌入向量 → 存储到 PostgreSQL + pgvector
```

1. **文档读取**：读取文档内容
2. **智能分块**：根据配置的大小和重叠进行分块
3. **生成嵌入**：使用嵌入服务为每个分块生成1024维向量
4. **向量存储**：存储到PostgreSQL pgvector表

### 检索流程（完整RAG流程）

```
用户问题 → RAG检测 → 检索（精确/召回模式） → LLM生成 → 事实核查 → 答案
```

1. **RAG检测**：使用LLM确定问题是否需要知识库搜索
   - 需要RAG：技术问题、文档查询、事实性问题
   - 不需要RAG：普通对话、问候、个人信息

2. **精确模式**（用于短查询或特殊符号）：
   - 精确匹配 → 关键词搜索 → 向量搜索（提前返回）
   - 无多查询，无分数稀释

3. **召回模式**（用于复杂查询）：
   - 多查询生成（原始+重写）
   - 混合检索（向量+关键词）
   - 结果排序和重新排序

4. **LLM生成**：使用本地LLM基于检索的上下文生成自然语言答案
   - 包含对话历史以提供上下文
   - 事实核查以纠正用户的误解

5. **记忆管理**：
   - 跟踪对话历史
   - 达到阈值后自动蒸馏（默认：3轮）
   - 将蒸馏记忆存储在知识库中以备将来检索

### 记忆蒸馏流程

```
对话历史 → 阈值检查 → 提取关键信息 → 生成嵌入 → 存储到知识库
```

1. **对话跟踪**：在会话记忆中存储每条消息
2. **阈值检查**：监控消息数量（可配置，默认：3）
3. **蒸馏触发**：达到阈值时，提取对话摘要
4. **向量生成**：为蒸馏记忆生成嵌入
5. **知识存储**：存储在知识库中以备将来检索

### 意图检测流程

```
用户输入 → 意图分析 → 路由到处理器 → 执行操作 → 返回响应
```

1. **意图分析**：检测用户意图类型
   - 知识纠正：检测纠正关键词（"纠正"、"改正"、"修正"、"不对"、"不是"）
   - 自我介绍：检测介绍模式（"我是XXX"、"我叫XXX"）
   - 普通问题：默认处理

2. **路由到处理器**：
   - 纠正：搜索知识库 → 记录纠正请求
   - 自我介绍：提取用户ID → 从蒸馏记忆中加载画像
   - 普通：执行标准RAG流程

3. **执行操作**：根据意图执行适当的操作
4. **返回响应**：向用户提供适当的响应

## 高级用法

### 精确模式示例

精确模式自动触发于短查询或包含特殊符号的查询：

```bash
# 启动聊天模式
go run main.go --chat

# 精确模式示例：
You: chunk_size = 200
# → 使用精确匹配 → 关键词 → 向量流程

You: a = x
# → 使用精确匹配 → 关键词 → 向量流程

You: timeout > 0
# → 使用精确匹配 → 关键词 → 向量流程

You: Go 代码规范是什么？
# → 使用召回模式和RAG
```

### 记忆蒸馏

系统在达到阈值后自动蒸馏对话历史：

```bash
# 启动聊天模式
go run main.go --chat

# 示例对话：
You: 你好
Assistant: Hello! How can I help you?

You: 我叫小明
Assistant: Hello Xiao Ming! Nice to meet you.

You: 还记得我的名字吗？
# → 触发记忆蒸馏（第3条消息）
# → 将对话摘要存储在知识库中
# → 可在将来的对话中检索

# 检查蒸馏记忆：
go run cmd/check_distillation/main.go
```

### 配置调优

#### 知识库参数

```yaml
knowledge:
  chunk_size: 500          # 较小的分块提高精确度
  chunk_overlap: 50        # 保持上下文连续性
  top_k: 5                 # 返回更多候选结果
  min_score: 0.6           # 提高相似度阈值
```

#### 记忆系统参数

```yaml
memory:
  enabled: true              # 启用记忆系统
  max_history: 10           # 保留的最大对话轮次
  enable_distillation: true  # 启用自动蒸馏
  distillation_threshold: 3  # 触发蒸馏前的消息数
```

**参数说明：**

- `chunk_size`：文档分块大小
  - 小值（200-300）：更精确，但可能丢失上下文
  - 中值（500-700）：平衡精确度和上下文（推荐）
  - 大值（1000+）：更多上下文，但精确度较低

- `chunk_overlap`：分块重叠大小
  - 通常设置为chunk_size的10-20%
  - 有助于保持语义连续性

- `top_k`：检索结果数量
  - 3-5：精确答案
  - 5-10：全面答案（推荐）
  - 10+：广泛探索

- `min_score`：最小相似度阈值
  - 0.7-0.8：高相关性，结果较少
  - 0.6-0.7：平衡相关性和结果数量（推荐）
  - 0.5-0.6：更多结果，可能包含不相关内容

- `distillation_threshold`：触发蒸馏前的消息数
  - 较低值（2-3）：更频繁蒸馏，每批上下文较少
  - 较高值（5-10）：较少蒸馏，每批上下文较多
  - 推荐：3用于活跃对话

## 架构

### 核心组件

```
KnowledgeBase (高级API)
    ├── Pool (数据库连接池)
    ├── KnowledgeRepository (知识库数据访问)
    ├── RetrievalService (智能检索 - SimpleRetrievalService)
    │   ├── 精确模式（精确匹配 → 关键词 → 向量）
    │   └── 召回模式（多查询 + 混合检索）
    ├── EmbeddingClient (嵌入服务)
    ├── LLMClient (本地LLM用于答案生成)
    ├── MemoryManager (对话历史和蒸馏)
    ├── TenantGuard (租户隔离)
    └── RetrievalGuard (限流熔断器)
```

### 数据流

**文档导入：**
```
文档 → 分块 → 嵌入向量 → PostgreSQL + pgvector
```

**知识问答（完整RAG）：**
```
问题 → RAG检测 → 精确/召回模式 → 检索 → LLM生成 → 事实核查 → 答案
```

**记忆管理：**
```
消息 → 会话记忆 → 阈值检查 → 蒸馏 → 知识库存储
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

导入大量文档时，分批处理：

```bash
# 每批导入10个文档
find docs/ -name "*.md" | head -n 10 | xargs -I {} go run main.go --save "{}"
```

## 故障排除

### 问题1：数据库连接失败

```
Error: create database pool: connection refused
```

**解决方案：**
- 检查PostgreSQL是否运行：`docker ps | grep postgres`
- 检查端口是否正确：`netstat -an | grep 5433`
- 检查配置文件中的数据库配置

### 问题2：嵌入服务不可用

```
Error: Failed to embed chunk: connection refused
```

**解决方案：**
- 检查Ollama是否运行：`ps aux | grep ollama`
- 检查模型是否已下载：`ollama list`
- 重启Ollama服务：`ollama serve`

### 问题3：导入超时

```
Import timeout (5 minutes exceeded)
```

**解决方案：**
- 检查嵌入服务响应速度
- 减小文档大小或增加分块数量
- 检查网络连接
- 检查哪个分块超时（日志显示详细信息）

### 问题4：搜索超时

```
Search timeout. Please try again.
```

**解决方案：**
- 检查数据库连接状态
- 检查嵌入服务是否正常
- 减少搜索结果数量（降低top_k值）
- 检查是否有高并发请求

### 问题5：程序冻结

**症状**：程序无响应，无法退出

**预防措施：**
- 所有操作都有超时保护（导入5分钟，搜索30秒）
- 每个分块都有独立超时（60秒）
- 使用Ctrl+C中断程序
- 非阻塞输入（bufio.Scanner）

**解决方案：**
- 按Ctrl+C中断程序
- 检查Ollama服务是否卡住：`curl http://localhost:11434/api/tags`
- 检查数据库是否卡住：`docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT 1;"`
- 重启相关服务

### 问题6：检索结果差

**解决方案：**
- 调整`chunk_size`和`chunk_overlap`
- 降低`min_score`阈值
- 增加`top_k`值
- 尝试不同的嵌入模型

### 问题8：未安装pgvector

```
Error: type "vector" does not exist
```

**解决方案：**
```bash
# 在PostgreSQL中安装pgvector扩展
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "CREATE EXTENSION vector;"
```

### 问题9：记忆蒸馏未触发

**症状**：尽管达到阈值，对话继续进行而不触发蒸馏

**解决方案：**
- 检查记忆配置：`memory.enable_distillation: true`
- 检查阈值：`memory.distillation_threshold: 3`
- 检查日志中的蒸馏触发：查找`🎯 [记忆蒸馏]`
- 验证对话消息数量与阈值匹配

### 问题10：LLM生成失败

**症状**：错误消息"LLM generation failed, falling back to raw results"

**解决方案：**
- 检查Ollama LLM模型是否可用：`ollama list`
- 验证配置中的LLM模型名：`llm.model: llama3.2:latest`
- 检查Ollama服务是否运行：`curl http://localhost:11434/api/tags`
- 增加LLM超时：`llm.timeout: 120`

### 问题11：事实核查不工作

**症状**：系统同意用户的错误陈述

**解决方案：**
- 确保LLM提示包含事实核查指令
- 检查检索的文档包含正确信息
- 验证该问题触发了RAG（检查日志）
- 尝试重新表述问题以触发RAG

### 问题12：精确模式未触发

**症状**：复杂查询使用向量搜索而不是精确匹配

**解决方案：**
- 精确模式触发于：`len(query) <= 10` 或包含 `=+-*/:`
- 检查日志：`Using precision mode`
- 对于较长的查询，系统正确使用召回模式
- 尝试更短的查询或包含特殊符号

## 扩展功能

### 1. 支持更多文档格式

集成PDF、Word和其他文档解析库：

```go
import "github.com/unidoc/unipdf/v3/extractor"

func loadPDF(path string) (string, error) {
    // PDF解析逻辑
}
```

### 2. 添加文档元数据

为文档添加更多元数据：

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

- **语言**：Go 1.21+
- **数据库**：PostgreSQL 16 + pgvector
- **嵌入服务**：Ollama（qwen3-embedding:0.6b）或自定义Python服务
- **LLM**：Ollama（llama3.2:latest）用于答案生成
- **配置**：YAML
- **检索**：
  - 向量相似度（pgvector）
  - BM25全文检索
  - 精确模式（精确匹配 → 关键词 → 向量）
  - 智能RAG检测
- **记忆系统**：基于会话的对话历史和自动蒸馏
- **事实核查**：自动检测和纠正用户误解

## 参考

- [存储API文档](../../docs/storage/api.md)
- [检索策略指南](../../docs/retrieval-strategy.md)
- [记忆系统文档](../../docs/memory/)
- [pgvector文档](https://github.com/pgvector/pgvector)
- [Ollama文档](https://github.com/ollama/ollama)
- [RAG最佳实践](https://docs.anthropic.com/claude/docs/retrieval-augmented-generation)
- [LLM查询重写](../../docs/llm/llm_query_rewrite.md)

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！