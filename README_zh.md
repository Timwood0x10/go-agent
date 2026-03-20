# GoAgent 框架

一个轻量级、高度可配置的多智能体框架，用于在 Go 中构建 AI 应用。

## GoAgent 是什么？

GoAgent 是一个**通用多智能体框架**，用户只需通过**配置文件（YAML）**即可构建 AI 应用。用户只需要做两件事：

1. 编写 YAML 配置文件
2. 编写简单的启动脚本（几行代码）
3. 框架完成所有复杂逻辑：

- **用户画像解析** - 从自然语言中提取用户偏好
- **动态任务规划** - 基于触发词自动拆解和调度任务
- **工具调度** - 统一管理各种工具
- **结果校验** - 确保输出符合预期格式
- **结果聚合** - 合并多个智能体的返回结果
- **记忆蒸馏** - 自动提取和总结对话中的关键信息
- **存储** - pgvector 向量存储，支持跨会话持久化

## 特性

- **多智能体架构**：Leader 智能体协调多个子智能体并行执行
- **AHP 协议**：自定义智能体心跳协议，用于智能体间通信
- **工作流引擎**：动态 DAG 工作流编排，支持热加载
- **LLM 集成**：统一适配 OpenAI、Ollama、OpenRouter 等 LLM 提供商
- **内存系统**：三级内存管理（会话、用户、任务），支持 RAG
- **优雅关闭**：五阶段关闭流程，支持回调注册
- **限流**：令牌桶、滑动窗口、信号量限流
- **工具系统**：可扩展的工具注册表
- **结果校验**：JSON Schema 校验，自动重试
- **向量存储**：PostgreSQL + pgvector 支持语义搜索和 RAG

## 系统要求

### 最低要求
- Go 1.26.1 或更高版本
- LLM API 访问权限（OpenAI、Ollama 或 OpenRouter）

### 可选要求（用于高级功能）
- PostgreSQL 16+ 带 pgvector 扩展（用于向量存储）
- Redis（用于缓存）
- golangci-lint（用于开发）

### 依赖项

框架使用极少的外部依赖：
- `github.com/fsnotify/fsnotify` - 文件系统监听
- `github.com/google/uuid` - UUID 生成
- `github.com/lib/pq` - PostgreSQL 驱动
- `github.com/stretchr/testify` - 测试框架
- `golang.org/x/*` - 标准 Go 扩展库
- `gopkg.in/yaml.v3` - YAML 解析

无重量级第三方框架依赖。

- **多智能体架构**：Leader 智能体协调多个子智能体并行执行
- **AHP 协议**：自定义智能体心跳协议，用于智能体间通信
- **工作流引擎**：动态 DAG 工作流编排，支持热加载
- **LLM 集成**：统一适配 OpenAI、Ollama、OpenRouter 等 LLM 提供商
- **内存系统**：三级内存管理（会话、用户、任务），支持 RAG
- **优雅关闭**：五阶段关闭流程，支持回调注册
- **限流**：令牌桶、滑动窗口、信号量限流
- **工具系统**：可扩展的工具注册表
- **结果校验**：JSON Schema 校验，自动重试

## 快速开始

### 运行旅行规划示例

```bash
cd /goagent

# 设置 API Key
export OPENROUTER_API_KEY="your-api-key"

# 运行
go run ./examples/travel/main.go
```

### 知识库示例

```bash
cd /goagent

# 启动 PostgreSQL + pgvector
docker run -d \
  --name postgres-pgvector \
  -p 5433:5432 \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goagent \
  pgvector/pgvector:pg16

# 导入文档
cd examples/knowledge-base
go run main.go --save example.md

# 问答
go run main.go --chat
```

### 示例输出

旅行规划示例：
```
=== 请求: 我想去日本东京旅游，5天4晚，预算10000元，喜欢美食和购物 ===
```

## 配置详解

所有配置都在 YAML 文件中。以下是可配置项：

### LLM 设置

```yaml
llm:
  provider: "openrouter"      # "openai", "ollama", "openrouter"
  api_key: ""                 # 使用环境变量：OPENROUTER_API_KEY
  base_url: "https://openrouter.ai/api/v1"
  model: "meta-llama/llama-3.1-8b-instruct"
  timeout: 60                 # 超时时间（秒）
  max_tokens: 4096           # 最大响应 token 数
```

### 智能体设置

```yaml
agents:
  leader:
    id: "leader-travel"
    max_steps: 10
    max_parallel_tasks: 4
    max_validation_retry: 3
    enable_cache: true

  sub:
    - id: "agent-destination"
      type: "destination"
      category: "destination"
      triggers: ["destination"]    # 触发此智能体的关键词
      max_retries: 3
      timeout: 30
      model: "..."               # 可选：为单个智能体指定模型
      provider: "..."            # 可选：为单个智能体指定 provider
```

### Prompt 模板

通过 YAML 模板自定义智能体行为：

```yaml
prompts:
  # 用户画像提取 - 将用户输入解析为结构化数据
  profile_extraction: |
    你是一位旅行助手。请从用户的输入中提取旅行偏好信息。
    用户输入: {{.input}}
    ...

  # 推荐生成 - 生成推荐结果
  recommendation: |
    请根据以下信息推荐 {{.Category}}：
    目的地: {{index . "destination"}}
    预算: {{index . "budget"}}
    ...
```

**模板变量：**

| 变量 | 描述 |
|------|------|
| `{{.input}}` | 原始用户输入（profile_extraction）|
| `{{.Category}}` | 智能体类型（recommendation）|
| `{{index . "key"}}` | 访问画像字段 |

### 输出设置

```yaml
output:
  format: "table"  # "table", "json", "simple"
  item_template: "{{.Name}} - {{.Price}}"
  summary_template: "获取了 {{.Count}} 个项目"
```

### 校验设置

配置 JSON Schema 结果校验：

```yaml
validation:
  enabled: true           # 启用/禁用校验
  schema_type: "travel"  # "fashion", "travel", "custom"
  retry_on_fail: true    # 校验失败时重试 LLM 调用
  max_retries: 3         # 最大重试次数
  strict_mode: false     # true: 校验失败返回错误; false: 记录日志并继续
```

**校验字段说明：**

| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| **Travel Schema (`schema_type: "travel"`)** |
| item_id | string | 是 | 唯一标识符 |
| name | string | 是 | 名称 |
| category | string | 是 | destination/food/hotel/itinerary/transport/activity |
| description | string | 否 | 描述 |
| price | number | 否 | 价格 (>= 0) |
| url | string | 否 | 链接 (uri 格式) |
| image_url | string | 否 | 图片链接 (uri 格式) |
| style | array | 否 | 风格标签列表 |
| colors | array | 否 | 颜色列表 |
| match_reason | string | 否 | 推荐理由 |
| brand | string | 否 | 品牌 |
| metadata | object | 否 | 额外元数据 |
| **结果级字段** |
| session_id | string | 否 | 会话 ID |
| user_id | string | 否 | 用户 ID |
| items | array | 是 | 项目数组 (最少1个) |
| reason | string | 否 | 推荐理由 |
| total_price | number | 否 | 总价 (>= 0) |
| match_score | number | 否 | 匹配分数 (0-1) |
| **Fashion Schema (`schema_type: "fashion"`)** |
| item_id | string | 是 | 唯一标识符 |
| category | string | 是 | top/bottom/dress/outerwear/shoes/accessory/bag/hat |
| name | string | 是 | 名称 |
| brand | string | 否 | 品牌 |
| price | number | 是 | 价格 (>= 0) |
| url | string | 否 | 链接 (uri 格式) |
| image_url | string | 否 | 图片链接 (uri 格式) |

**校验行为：**
- `retry_on_fail: true` - 校验失败时自动重试 LLM 调用
- `strict_mode: true` - 校验失败时返回错误；否则仅记录日志并继续使用未校验的结果

### 存储设置（可选）

```yaml
storage:
  enabled: false            # 启用 PostgreSQL 存储
  type: "postgres"
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "postgres"
  database: "goagent"
  ssl_mode: "disable"
  pgvector:
    enabled: false          # 启用 pgvector 向量搜索
    dimension: 1536         # 嵌入维度
    table_name: "embeddings"
```

### 内存设置（可选）

```yaml
memory:
  enabled: false            # 启用内存系统
  session:
    enabled: true
    max_history: 50         # 最大对话轮次
  user_profile:
    enabled: false          # 启用持久化用户画像
    storage: "memory"       # "memory" 或 "postgres"
    vector_db: false         # 存储为向量
  task_distillation:
    enabled: false          # 启用任务蒸馏
    storage: "memory"       # "memory" 或 "postgres"
    vector_store: false     # 存储在 pgvector 中
    prompt: "请简洁总结以下任务的关键信息，包括：用户需求、偏好、预算范围。"
```

### 检索策略（可选）

框架提供两种检索服务以适应不同使用场景：

| 使用场景 | 推荐服务 | 描述 |
|----------|---------|------|
| **单知识库检索**（RAG、Q&A、文档搜索） | ✅ SimpleRetrievalService | 纯向量相似度搜索，无复杂权重计算。最适合单源语义搜索场景。 |
| **精确匹配查询**（如 "a = x"、配置项查询） | ✅ SimpleRetrievalService | 精确模式，采用 Exact Match → Keyword → Vector 流程（提前返回）。适合需要确定性匹配的精确查询。 |
| **多源融合检索**（Knowledge + Experience + Tools） | ✅ RetrievalService | 混合搜索，支持多源融合、查询重写和时间衰减。用于复杂企业系统。 |
| **复杂企业系统**（需要时间衰减、权重控制） | ✅ RetrievalService | 高级功能，包括查询权重、源权重、基于时间的评分和结果重排。 |

**SimpleRetrievalService 特性：**
- 纯向量相似度搜索（1 - cosine_distance）
- 精确模式：Exact Match → Keyword → Vector（提前返回）
- 无复杂权重计算
- 无时间衰减
- 无查询重写
- 简单高效，适合单知识库场景

**RetrievalService 特性：**
- 多源搜索（Knowledge + Experience + Tools）
- 查询重写及权重控制（original=1.0, rule=0.7, llm=0.5）
- 源权重配置
- 基于时间的评分衰减
- 结果融合和重排
- 企业级高级功能

## 架构



```

用户输入

    │

    ▼

┌─────────────────┐

│ Leader 智能体   │ ── 解析用户画像 (LLM)

│                │ ── 规划任务 (基于触发词)

└────────┬────────┘

         │ 并行派发

         ▼

┌────────┴────────┐

│ 子智能体          │

│ (并行)           │

└────────┬────────┘

         │ 返回结果

         ▼

┌─────────────────┐

│ 结果校验 (Schema) │ ── JSON Schema 验证

│                │ ── 失败自动重试 (可选)

└────────┬────────┘

         │ 校验通过

         ▼

┌─────────────────┐

│ 结果聚合          │

└─────────────────┘

```

## 项目结构

```
goagent/
├── cmd/                  # 应用入口
├── configs/              # 配置文件
├── docs/                 # 架构文档
├── examples/
│   ├── travel/          # 旅行规划示例
│   └── simple/           # 简单示例
├── internal/
│   ├── agents/
│   │   ├── base/        # 基础接口
│   │   ├── leader/      # Leader 智能体
│   │   └── sub/          # 子智能体
│   ├── core/
│   │   ├── errors/       # 错误处理
│   │   └── models/       # 数据模型
│   ├── llm/
│   │   └── output/       # LLM 适配器
│   ├── memory/           # 内存系统
│   ├── protocol/          # AHP 协议
│   ├── ratelimit/        # 限流
│   ├── shutdown/          # 优雅关闭
│   ├── storage/          # PostgreSQL 存储
│   ├── tools/            # 工具系统
│   └── workflow/         # 工作流引擎
└── pkg/                  # 工具函数
```

## 示例

### 1. 旅行规划智能体 (`examples/travel/`)
多智能体旅行助手，演示：
- 从自然语言中解析用户画像
- 基于触发词的动态任务规划
- 子智能体并行执行
- 结果聚合

**运行：**
```bash
export OPENROUTER_API_KEY="your-api-key"
go run ./examples/travel/main.go
```

### 2. 知识库 (`examples/knowledge-base/`)
本地文档检索和问答系统，演示：
- 文档导入和分块
- 向量相似度检索 (pgvector)
- 多租户隔离
- 交互式聊天界面

**运行：**
```bash
cd examples/knowledge-base
go run main.go --save example.md
go run main.go --chat
```

### 3. 简单智能体 (`examples/simple/`)
基础多智能体示例，包含穿搭推荐。

**运行：**
```bash
go run ./examples/simple/main.go
```

详细配置请参阅各示例的 README 文件。

## 开发

### 前置要求
- Go 1.26.1+
- golangci-lint: `brew install golangci-lint`
- staticcheck: `go install honnef.co/go/tools/cmd/staticcheck@latest`
- goimports: `go install golang.org/x/tools/cmd/goimports@latest`

### 命令

```bash
# 安装依赖
make install

# 格式化代码
make fmt

# 运行所有检查（lint + test）
make check

# 运行测试并生成覆盖率
make test

# 运行测试并开启竞态检测
make test-race

# 运行代码检查
make lint

# 运行 CI 检查（install, fmt, lint, test-race）
make ci

# 构建二进制文件
make build

# 构建所有二进制文件
make build-all

# 清理构建产物
make clean

# 显示帮助信息
make help
```


运行 `make check-all` 验证所有覆盖率要求。

## 文档

- [架构](docs/arch.md) - 系统架构概览
- [智能体](docs/agents/) - 智能体设计和定义
- [核心](docs/core/) - 核心组件（错误、模型、注册表）
- [引擎](docs/engine/) - 工作流引擎设计
- [LLM](docs/llm/) - LLM 集成和查询重写
- [内存](docs/memory/) - 内存系统设计
- [协议](docs/protocol/) - AHP 协议规范
- [限流](docs/ratelimit/) - 限流策略
- [关闭](docs/shutdown/) - 优雅关闭机制
- [存储](docs/storage/) - PostgreSQL 存储与 pgvector
- [工具](docs/tools/) - 工具系统设计
- [检索策略](docs/retrieval-strategy_zh.md) - 知识检索策略

## 许可证

MIT 许可证
