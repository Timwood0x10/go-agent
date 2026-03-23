# GoAgent 框架

一个轻量级、高度可配置的多智能体框架，用于在 Go 中构建 AI 应用程序。

## GoAgent 是什么？

GoAgent 是一个**通用多智能体框架**，允许用户仅通过**配置（YAML）**构建 AI 应用程序。用户只需要：

1. 编写一个 YAML 配置文件
2. 编写一个简单的启动脚本（几行代码）
3. 框架处理所有复杂的逻辑：

- **用户画像解析** - 从自然语言中提取用户偏好
- **动态任务规划** - 基于触发器自动拆分和调度任务
- **工具调度** - 统一的工具管理
- **结果验证** - 确保输出符合预期的模式
- **结果聚合** - 合并来自多个智能体的结果
- **记忆蒸馏** - 自动提取和总结对话中的关键信息
- **存储** - pgvector 向量存储，用于跨会话持久化

## 特性

- **多智能体架构**：Leader 智能体编排多个子智能体进行并行任务执行
- **AHP 协议**：自定义智能体心跳协议，用于智能体间通信
- **工作流引擎**：基于 DAG 的动态工作流编排，支持热重载
- **LLM 集成**：统一适配器，支持 OpenAI、Ollama、OpenRouter 等 LLM 提供商
- **内存系统**：三层内存管理（会话、用户、任务），支持 RAG
- **优雅关闭**：五阶段关闭机制，支持回调注册
- **速率限制**：令牌桶、滑动窗口和基于信号量的限流
- **工具系统**：可扩展的工具注册表，用于智能体能力
- **结果验证**：JSON Schema 验证，支持自动重试
- **向量存储**：PostgreSQL + pgvector，用于语义搜索和 RAG
- **能力层（ACE）**：智能体能力引擎，用于智能工具选择和能力路由

## 系统要求

### 最低要求
- Go 1.26.1 或更高版本
- LLM API 访问权限（OpenAI、Ollama 或 OpenRouter）

### 可选要求（用于高级功能）
- PostgreSQL 16+ 及 pgvector 扩展（用于向量存储）
- Redis（用于缓存）
- golangci-lint（用于开发）

### 依赖项

框架使用最少的外部依赖：
- `github.com/fsnotify/fsnotify` - 文件系统监视器
- `github.com/google/uuid` - UUID 生成
- `github.com/lib/pq` - PostgreSQL 驱动
- `github.com/stretchr/testify` - 测试框架
- `golang.org/x/*` - 标准 Go 扩展库
- `gopkg.in/yaml.v3` - YAML 解析

没有繁重的第三方框架依赖。

## 快速开始

### 运行旅行规划示例

```bash
cd /goagent

# 设置 API 密钥
export OPENROUTER_API_KEY="your-api-key"

# 运行
go run ./examples/travel/main.go
```

### 尝试知识库示例

```bash
cd goagent

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

# 提问
go run main.go --chat
```

## 配置参考

所有配置都在 YAML 中。以下是您可以配置的内容：

### LLM 设置

```yaml
llm:
  provider: "openrouter"      # "openai", "ollama", "openrouter"
  api_key: ""                 # 使用环境变量：OPENROUTER_API_KEY
  base_url: "https://openrouter.ai/api/v1"
  model: "meta-llama/llama-3.1-8b-instruct"
  timeout: 60                 # 秒
  max_tokens: 4096           # 最大响应令牌数
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
      model: "..."               # 可选：每个智能体的模型
      provider: "..."            # 可选：每个智能体的提供商
```

### 提示词模板

通过 YAML 模板自定义智能体行为：

```yaml
prompts:
  # 用户画像提取 - 将用户输入解析为结构化数据
  profile_extraction: |
    你是一位旅行助手。请从用户的输入中提取旅行偏好信息。
    用户输入: {{.input}}
    ...

  # 推荐 - 生成推荐
  recommendation: |
    请根据以下信息推荐 {{.Category}}：
    目的地: {{index . "destination"}}
    预算: {{index . "budget"}}
    ...
```

**模板变量：**

| 变量 | 描述 |
|------|------|
| `{{.input}}` | 原始用户输入（profile_extraction） |
| `{{.Category}}` | 智能体类型（recommendation） |
| `{{index . "key"}}` | 访问画像字段 |

### 输出设置

```yaml
output:
  format: "table"  # "table", "json", "simple"
  item_template: "{{.Name}} - {{.Price}}"
  summary_template: "获得 {{.Count}} 个项目"
```

### 验证设置

使用 JSON Schema 配置结果验证：

```yaml
validation:
  enabled: true           # 启用/禁用验证
  schema_type: "travel"  # "fashion", "travel", "custom"
  retry_on_fail: true    # 验证失败时重试 LLM 调用
  max_retries: 3         # 最大重试次数
  strict_mode: false     # 如果为 true，验证失败时返回错误
```

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
    enabled: false          # 启用 pgvector 进行向量搜索
    dimension: 1536         # 嵌入维度
    table_name: "embeddings"
```

### 内存设置（可选）

```yaml
memory:
  enabled: false            # 启用内存系统
  session:
    enabled: true
    max_history: 50         # 最大对话轮数
  user_profile:
    enabled: false          # 启用持久化用户画像
    storage: "memory"       # "memory" 或 "postgres"
    vector_db: false         # 将画像存储为向量
  task_distillation:
    enabled: false          # 启用任务蒸馏
    storage: "memory"       # "memory" 或 "postgres"
    vector_store: false     # 在 pgvector 中存储蒸馏结果
    prompt: "请简洁总结以下任务的关键信息，包括：用户需求、偏好、预算范围。"
```

### 检索策略（可选）

框架为不同用例提供两种检索服务：

| 用例 | 推荐服务 | 描述 |
|------|---------|------|
| **单知识库检索**（RAG、问答、文档搜索） | ✅ SimpleRetrievalService | 纯向量相似性搜索，无复杂权重。最适合单源语义搜索场景。 |
| **精确匹配查询**（如 "a = x"，配置查找） | ✅ SimpleRetrievalService | 精确模式：精确匹配 → 关键词 → 向量（提前返回）。非常适合需要确定性匹配的精确查询。 |
| **多源融合检索**（知识+经验+工具） | ✅ RetrievalService | 具有多源融合、查询重写和时间衰减的混合搜索。用于复杂企业系统。 |
| **复杂企业系统**（时间衰减、权重控制） | ✅ RetrievalService | 高级功能，包括查询权重、源权重、基于时间的评分和结果重新排序。 |

**SimpleRetrievalService 特性：**
- 纯向量相似性搜索（1 - cosine_distance）
- 精确模式：精确匹配 → 关键词 → 向量（提前返回）
- 无复杂权重计算
- 无时间衰减
- 无查询重写
- 简单有效，适用于单知识库场景

**RetrievalService 特性：**
- 多源搜索（知识+经验+工具）
- 查询重写，权重控制（original=1.0, rule=0.7, llm=0.5）
- 源权重配置
- 基于时间的评分衰减
- 结果合并和重新排序
- 复杂的企业级功能

## 架构

```
用户输入
    │
    ▼
┌─────────────────┐
│ Leader 智能体  │ ── 解析画像（LLM）
│                │ ── 规划任务（基于触发器）
└────────┬────────┘
         │ 并行调度
         ▼
┌────────┴────────┐
│ 子智能体         │
│ （并行）         │
└────────┬────────┘
         │ 结果
         ▼
┌─────────────────┐
│ 验证            │ ── JSON Schema 检查
│ (Schema)        │ ── 失败时自动重试（可选）
└────────┬────────┘
         │ 已验证
         ▼
┌─────────────────┐
│ 聚合            │
└─────────────────┘
```

## 项目结构

```

goagent/

├── cmd/                  # 应用程序入口点

│   ├── server/          # 主服务器应用程序

│   ├── migrate_goagent/ # 数据库迁移工具

│   └── setup_test_db/   # 测试数据库设置

├── configs/              # 配置文件

├── docs/                 # 架构文档

├── examples/

│   ├── travel/          # 旅行规划示例

│   ├── simple/           # 简单示例

│   ├── knowledge-base/   # 知识库示例

│   ├── openrouter/       # OpenRouter 示例

│   └── devagent/         # 开发智能体

├── internal/

│   ├── agents/

│   │   ├── base/        # 基础接口

│   │   ├── leader/      # Leader 智能体

│   │   └── sub/          # 子智能体

│   ├── config/          # 配置管理

│   ├── core/

│   │   ├── errors/       # 错误处理

│   │   ├── models/       # 数据模型

│   │   └── registry/     # 组件注册表

│   ├── llm/

│   │   └── output/       # LLM 适配器

│   ├── memory/           # 内存系统

│   ├── observability/    # 日志和跟踪

│   ├── protocol/          # AHP 协议

│   ├── ratelimit/        # 速率限制

│   ├── security/         # 安全工具

│   ├── shutdown/          # 优雅关闭

│   ├── storage/

│   │   └── postgres/     # PostgreSQL + pgvector

│   ├── tools/            # 工具系统

│   └── workflow/         # 工作流引擎

├── knowledge/            # 知识库数据（Python 脚本）

├── services/             # 服务配置

│   └── embedding/        # 嵌入服务

└── pkg/                  # 工具

```

## 能力层（ACE）

**智能体能力引擎（ACE）**为智能体提供智能工具选择和能力路由。它解决了多智能体系统中工具选择稳定性和准确性的问题。

### 问题陈述

没有 ACE：
- LLM 看到所有可用工具（例如 12-22 个工具）
- 工具选择变得不稳定和不准确
- 更高的令牌使用和更慢的响应
- LLM 可能选择不合适的工具

有 ACE：
- LLM 只看到相关工具（2-4 个工具）
- 更好的工具选择准确性
- 减少令牌使用和更快的响应
- 跨查询的一致工具匹配

### 能力

能力是工具提供的高级抽象。系统支持 8 种核心能力：

| 能力 | 描述 | 英文关键词 | 中文关键词 |
|------------|-------------|------------------|------------------|
| `math` | 数学计算 | calculate, sum, multiply, divide, compute | 计算, 求和, 乘, 除, 加, 减, 数字, 公式, 数学 |
| `knowledge` | 知识检索 | what, who, explain, search, find | 什么, 谁, 解释, 信息, 搜索, 查找, 查询, 知识 |
| `memory` | 内存访问/存储 | remember, store, recall, history | 记住, 存储, 回忆, 历史, 记忆, 保存 |
| `text` | 文本处理 | parse, format, validate, transform | 解析, 格式, 验证, 转换, 文本, 提取, 分析, 处理 |
| `network` | 网络/API 请求 | api, request, fetch, http, url | 请求, 获取, 下载, 网络, 网页, 网址 |
| `time` | 日期/时间操作 | time, date, schedule, timestamp | 时间, 日期, 时刻, 时间戳, 日历, 持续, 何时, 几点, 现在, 当前 |
| `file` | 文件系统操作 | file, read, write, delete, list | 文件, 目录, 读取, 写入, 删除, 列出, 保存, 加载, 路径, 文件夹 |
| `external` | 外部系统交互 | execute, run, command, script | 外部, 系统, 执行, 运行, 命令, 脚本 |

### ACE 工作流程

```
用户查询
    │
    ▼
┌─────────────────────┐
│ LLM 意图分析       │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ 能力检测           │  ← 关键词匹配（英文 + 中文）
│ - math             │
│ - knowledge        │
│ - memory           │
│ - text             │
│ - network          │
│ - time             │
│ - file             │
│ - external         │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   工具过滤          │  ← 将能力映射到工具
│ - math → calculator │
│ - time → datetime   │
│ - file → file_tools │
│ - network → http,   │
│           scraper   │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   工具排序          │  ← 优先考虑相关工具
│ - 相关性评分       │
│ - 类别匹配         │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  LLM 使用 2-4 工具  │  ← 专注的工具集
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  工具执行           │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  结果格式化        │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  用户响应          │
└─────────────────────┘
```

### 工具分类

#### 数学工具
- `calculator`: 数学表达式计算，支持复杂公式
- `datetime`: 日期和时间操作

**示例:**
```bash
> Calculate 100*(100+1)/2
[TOOL:calculator {"expression": "100*(100+1)/2"}]
Result: 5050

> 计算 1 到 100 的和
[TOOL:calculator {"expression": "100*(100+1)/2"}]
Result: 5050
```

#### 网络工具
- `http_request`: HTTP 请求（GET/POST/PUT/DELETE）
- `web_scraper`: 网页内容提取和解析

**示例:**
```bash
> Fetch data from https://httpbin.org/get
[TOOL:http_request {"url": "https://httpbin.org/get"}]
Result: {"args": {}, "headers": {...}}

> Extract content from https://example.com
[TOOL:web_scraper {"url": "https://example.com"}]
Result: {"title": "Example Domain", "content": "..."}
```

#### 文件工具
- `file_tools`: 文件系统操作（读取、写入、列表）

**示例:**
```bash
> List files in current directory
[TOOL:file_tools {"operation": "list", "directory_path": "."}]
Result: 
目录: .
  📁 bin
  📁 config
  📄 main.go (12345 bytes)

> 列出当前目录下的文件
[TOOL:file_tools {"operation": "list", "directory_path": "."}]
Result: 目录: . 📁 bin 📁 config 📄 main.go (12345 bytes)
```

#### 文本工具
- `text_processor`: 文本处理（计数、转换、分割）
- `json_tools`: JSON 解析和转换
- `data_validation`: 数据验证
- `data_transform`: 数据转换
- `regex_tool`: 正则表达式匹配
- `log_analyzer`: 日志分析

#### 知识工具
- `knowledge_search`: 知识库搜索
- `knowledge_add`: 添加知识
- `knowledge_update`: 更新知识
- `knowledge_delete`: 删除知识
- `correct_knowledge`: 纠正知识

#### 内存工具
- `memory_search`: 搜索对话历史
- `user_profile`: 用户画像管理
- `distilled_memory_search`: 蒸馏内存搜索

#### 系统工具
- `id_generator`: ID 生成（UUID、短ID）

#### 执行工具
- `code_runner`: 代码执行（Python、JavaScript）

#### 规划工具
- `task_planner`: 任务规划

### 关键特性

1. **自动能力检测**：查询中的关键词与能力匹配（英文 + 中文）
2. **动态工具过滤**：只向 LLM 显示相关工具（2-4 个工具，而不是 12-22 个）
3. **减少令牌使用**：提示令牌减少 60-80%
4. **更好的准确性**：专注的工具选择提高可靠性
5. **可扩展性**：易于添加新能力和工具
6. **中文支持**：所有能力的完整中文关键词支持
7. **相对路径处理**：自动将相对路径转换为绝对路径
8. **文件名建议**：文件未找到时的智能建议
9. **提示溢出防护**：提示超过限制时回退到核心工具

### 使用示例

```go
// 使用 ACE 创建智能体
toolCfg := &agent.AgentToolConfig{
    Enabled: nil, // 启用所有工具
}

agent, err := NewCapabilityDemoAgent(
    "demo-agent-1",
    "Demo Agent",
    "Demonstrates ACE workflow",
    toolCfg,
    llmClient,
    systemPrompt,
)

// 使用 ACE 处理用户查询
resp, err := agent.Process(ctx, "Calculate 1 to 100 sum")
// ACE 自动：
// 1. 检测 [math] 能力
// 2. 匹配 [calculator] 工具
// 3. 执行计算
// 4. 返回格式化结果
```

### 试用演示

```bash
cd examples/capability-demo
go run main.go

# 尝试这些查询：
> Calculate 1 to 100 sum
> What time is it?
> List files in current directory
> 搜索信息
> 计算 1+2
> 列出当前目录下的文件
```

### 实现细节

- **核心实现**：`internal/tools/resources/core/capability.go`
- **工具实现**：`internal/tools/resources/builtin/`
- **演示应用程序**：`examples/capability-demo/`
- **设计文档**：`/plan/CapabilityLayer.md`

## 示例

### 1. 旅行规划智能体 (`examples/travel/`)
多智能体旅行助手，展示：
- 从自然语言解析用户画像
- 基于触发器的动态任务规划
- 并行子智能体执行
- 结果聚合

**运行：**
```bash
export OPENROUTER_API_KEY="your-api-key"
go run ./examples/travel/main.go
```

### 2. 知识库 (`examples/knowledge-base/`)
本地文档检索和问答系统，展示：
- 带分块的文档导入
- 向量相似性搜索（pgvector）
- 多租户隔离
- 交互式聊天界面

**运行：**
```bash
cd examples/knowledge-base
go run main.go --save example.md
go run main.go --chat
```

### 3. 简单智能体 (`examples/simple/`)
基础多智能体示例，包含时尚推荐。

**运行：**
```bash
go run ./examples/simple/main.go
```

查看各个示例的 README 获取详细配置。

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

# 运行竞态检测测试
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

# 显示帮助
make help
```

运行 `make check-all` 以验证所有覆盖率要求。

## 贡献

欢迎贡献！请遵循以下准则：

1. **代码风格**
   - 提交前运行 `make fmt`
   - 通过 `make lint` 检查
   - 为新功能添加测试

2. **测试**
   - 所有测试必须通过：`make test`
   - 维护覆盖率要求
   - 为新功能添加集成测试

3. **文档**
   - 为新示例更新 README
   - 为复杂逻辑添加内联注释
   - 为结构更改更新架构文档

4. **Pull 请求**
   - 在 PR 描述中描述更改
   - 引用相关问题
   - 确保 CI 检查通过

## 许可证

MIT 许可证

## 文档

- [架构](docs/arch.md) - 系统架构概述
- [智能体](docs/agents/) - 智能体设计和定义
- [核心](docs/core/) - 核心组件（错误、模型、注册表）
- [引擎](docs/engine/) - 工作流引擎设计
- [LLM](docs/llm/) - LLM 集成和查询重写
- [内存](docs/memory/) - 内存系统设计
- [协议](docs/protocol/) - AHP 协议规范
- [速率限制](docs/ratelimit/) - 速率限制策略
- [关闭](docs/shutdown/) - 优雅关闭机制
- [存储](docs/storage/) - PostgreSQL 存储和 pgvector
- [工具](docs/tools/) - 工具系统设计
- [检索策略](docs/retrieval-strategy.md) - 知识检索策略

## 许可证

MIT 许可证