# Style Agent Framework 架构设计

## 系统架构总览

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Input                                    │
│                    "Xiao Ming, male, 22..."                            │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Leader Agent                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                │
│  │ Parse Profile│  │  Task Plan   │  │  Aggregate   │                │
│  │   (LLM)      │  │  (LLM)       │  │   Results    │                │
│  └──────────────┘  └──────────────┘  └──────────────┘                │
│         │                 │                 │                           │
│         └─────────────────┼─────────────────┘                           │
│                           │                                              │
│                    dispatch tasks (并行)                                 │
└───────────────────────────┬─────────────────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  agent_top      │ │ agent_bottom    │ │  agent_head     │
│  agent_shoes    │ │    ...          │ │    ...          │
│  (Worker Actors)│ │                 │ │                 │
└────────┬────────┘ └────────┬────────┘ └────────┬────────┘
         │                  │                  │
         └──────────────────┼──────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      AHP Protocol Layer                                 │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  Message Queue (In-Memory)                                     │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐              │    │
│  │  │ leader  │ │agent_top│ │agent_bt │ │agent_hd │   ...        │    │
│  │  │  queue  │ │  queue  │ │  queue  │ │  queue  │              │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘              │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  Message Types: TASK | RESULT | PROGRESS | ACK | HEARTBEAT            │
└─────────────────────────────────────────────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│     Tools       │ │     LLM         │ │    Storage      │
│                 │ │                  │ │                 │
│ • fashion_search│ │ • gpt-oss:20b   │ │  • PostgreSQL   │
│ • weather_check │ │ • llama3.2:3b   │ │  • pgvector     │
│ • style_recomm  │ │                  │ │                 │
└─────────────────┘ └─────────────────┘ └────────┬────────┘
         │                                    │
         │                                    ▼
         │                        ┌─────────────────────────┐
         │                        │  PostgreSQL + pgvector   │
         │                        │  ┌───────────────────┐  │
         │                        │  │ • Connections    │  │
         │                        │  │   Pool (25/10)   │  │
         │                        │  │ • Transactions  │  │
         │                        │  │ • RLS (Tenant)    │  │
         │                        │  └───────────────────┘  │
         │                        │  ┌───────────────────┐  │
         │                        │  │ • Vector Index   │  │
         │                        │  │   (ivfflat)      │  │
         │                        │  │ • Cosine Search  │  │
         │                        │  └───────────────────┘  │
         │                        └─────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Memory System                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                    │
│  │ SessionMemory│  │UserMemory   │  │TaskMemory   │                    │
│  │  (短期-内存)  │  │ (长期-PG)    │  │ (蒸馏-PG)    │                    │
│  └─────────────┘  └─────────────┘  └─────────────┘                    │
│                    │                                                 │
│  ProductionMemoryManager (PG + pgvector 持久化)              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 消息流转机制

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     消息生命周期 (Message Lifecycle)                     │
└─────────────────────────────────────────────────────────────────────────┘

   [发送]                    [队列]                     [接收/处理]
   
┌─────────┐              ┌─────────┐                ┌─────────┐
│ Leader  │  TASK       │  MQ     │                │ Sub     │
│         │ ─────────▶  │         │  ─────────▶    │  Agent  │
│         │              │         │                │         │
│         │ ◀─────────  │         │ ◀─────────     │         │
└─────────┘   RESULT    └─────────┘    ACK         └─────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  消息类型 (Message Types)                                                │
├─────────────────────────────────────────────────────────────────────────┤
│  TASK     │ 派发任务          │ Leader  → Sub Agent                    │
│ RESULT    │ 返回结果          │ Sub Agent → Leader                     │
│ PROGRESS  │ 进度汇报          │ Sub Agent → Leader                     │
│ ACK       │ 确认收到          │ Sub Agent → Leader                     │
│ HEARTBEAT │ 心跳保活          │ All Agents                              │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 任务生产消费流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     任务生产-消费模型                                     │
└─────────────────────────────────────────────────────────────────────────┘

                          Leader Agent (Producer)
                                 │
                                 │ 1. 解析用户输入
                                 ▼
                        ┌─────────────────┐
                        │  Parse Profile  │
                        │   (UserProfile) │
                        └────────┬────────┘
                                 │
                                 │ 2. LLM 决策需要哪些 agent
                                 ▼
                        ┌─────────────────┐
                        │ Determine Tasks │
                        │ [top, bottom]   │
                        └────────┬────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
        ┌──────────┐       ┌──────────┐       ┌──────────┐
        │Task Queue│       │Task Queue│       │Task Queue│
        │ agent_top│       │agent_btm │       │   ...    │
        └────┬─────┘       └────┬─────┘       └────┬─────┘
             │                   │                  │
             │    ┌──────────────┴──────────────┐   │
             │    │     Phase 1: 并行派发        │   │
             │    │  ThreadPoolExecutor          │   │
             │    └─────────────────────────────┘   │
             │                                      │
             ▼                                      ▼
       ┌──────────┐                          ┌──────────┐
       │Sub Agent │                          │Sub Agent │
       │ (Consumer)                          │ (Consumer)│
       │             ┌──────────────────────┘          │
       │             │                                   │
       │             │ 3. 处理任务                        │
       │             ▼                                   │
       │    ┌─────────────────┐                         │
       │    │  Execute Task   │                         │
       │    │  ┌───────────┐  │                         │
       │    │  │  Tools    │  │                         │
       │    │  │  (RAG)   │  │                         │
       │    │  │  LLM     │  │                         │
       │    │  └───────────┘  │                         │
       │    └────────┬────────┘                         │
       │             │                                   │
       │             │ 4. 返回结果                       │
       │             ▼                                   │
       │    ┌─────────────────┐                         │
       │    │  Send RESULT    │ ◀───────────────────────┘
       │    │  to Leader     │
       │    └────────┬────────┘
       │             │
       │    ┌────────┴────────┐
       │    │  Phase 2:       │
       │    │  依赖感知的任务  │
       │    │  (top结果给shoes)│
       │    └────────┬────────┘
       │             │
       │             ▼
       │    ┌─────────────────┐
       │    │  Coordination   │
       │    │  Context        │
       │    └────────┬────────┘
       │             │
       └─────────────┘
       
                       
       Leader (Aggregator)
              │
              │ 5. 聚合所有结果
              ▼
       ┌─────────────────┐
       │ Aggregate       │
       │ [top+bottom+    │
       │  head+shoes]   │
       └────────┬────────┘
                │
                ▼
       ┌─────────────────┐
       │ Final Output    │
       │ + Save to DB   │
       └─────────────────┘
```

---

## Actor 模型对应关系

| Actor 模型概念 | iFlow 实现 |
|----------------|-----------|
| Actor | `LeaderAgent`, `OutfitSubAgent` |
| Mailbox | `MessageQueue` (In-Memory) |
| Message | `AHPMessage` (TASK/RESULT/PROGRESS/ACK) |
| Behavior | Agent 内部的 `_handle_task()`, `_recommend()` |
| Supervisor | `LeaderAgent` 协调多个 Sub Agent |
| Failure Handling | DLQ (Dead Letter Queue) |

---

## 关键设计点

| 特性 | 实现方式 |
|------|----------|
| **并发模型** | Worker Pool 派发任务到多个 Sub Agent |
| **通信协议** | In-Memory Message Queue + AHP 自定义协议 |
| **状态管理** | SessionMemory 短期会话 + TaskMemory 蒸馏 |
| **容错机制** | DLQ 存储失败消息，支持重试 |
| **任务协调** | Phase 1 (并行) → Phase 2 (依赖感知) |
| **扩展性** | 可动态注册新的 Agent 类型 |
| **Agent 定义** | Markdown 文件配置，支持热加载 |
| **工作流编排** | YAML/JSON DSL，用户自定义流程 |
| **LLM 输出** | 四层保障机制确保输出一致性 |

---

## Agent 定义 (Markdown 配置)

Agent 采用 Markdown 文件定义，允许非开发人员通过编辑配置文件调整 Agent 行为。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Agent Definition (Markdown)                         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ # agent_top.md                                               │
├─────────────────────────────────────────────────────────────┤
│ ## Metadata                                                 │
│ name: agent_top                                             │
│ version: 1.0.0                                             │
│                                                             │
│ ## Role                                                     │
│ 你是一位专业的时尚穿搭顾问，专注于上衣搭配建议。              │
│                                                             │
│ ## Profile                                                  │
│ expertise: 上衣搭配                                         │
│ category: tops                                              │
│ style_tags: [casual, formal, street]                        │
│                                                             │
│ ## Tools                                                    │
│ - fashion_search                                            │
│ - weather_check                                             │
│ - style_recomm                                              │
│                                                             │
│ ## Instructions                                             │
│ 1. 根据用户风格偏好推荐合适的款式                           │
│ 2. 考虑当地天气因素                                         │
│ 3. 匹配用户预算范围                                         │
│                                                             │
│ ## Output Format                                            │
│ ```json                                                     │
│ { "items": [...], "reason": "..." }                        │
│ ```                                                         │
└─────────────────────────────────────────────────────────────┘
```

### 内置变量

| 变量 | 说明 |
|------|------|
| {{.UserProfile}} | 用户画像 |
| {{.SessionID}} | 会话 ID |
| {{.Context}} | 上下文信息 |
| {{.Input}} | 用户输入 |
| {{.Results}} | 上游结果 |

---

## Workflow Engine (工作流编排)

用户可以通过 YAML/JSON 文件自定义工作流，实现灵活的 Agent 编排。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Workflow Engine                                     │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  workflow.yaml                                                       │
├─────────────────────────────────────────────────────────────────────┤
│  name: "穿搭推荐流程"                                               │
│                                                                     │
│  agents:                                                            │
│    - id: leader                                                     │
│      type: leader                                                  │
│      prompt_file: ./agents/agent_leader.md                         │
│                                                                     │
│    - id: agent_top                                                 │
│      type: sub                                                     │
│      prompt_file: ./agents/agent_top.md                            │
│      depends_on: [leader]                                          │
│                                                                     │
│    - id: agent_shoes                                               │
│      type: sub                                                     │
│      prompt_file: ./agents/agent_shoes.md                          │
│      depends_on: [leader, agent_top]                               │
│                                                                     │
│  execution:                                                         │
│    phase1: [agent_top, agent_bottom]                             │
│    phase2: [agent_shoes]                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### 目录结构

```
workflows/
├── default.yaml          # 默认工作流
├── summer.yaml           # 夏季推荐
├── winter.yaml           # 冬季推荐
│
├── agents/               # Agent Markdown 定义
│   ├── agent_leader.md
│   ├── agent_top.md
│   ├── agent_bottom.md
│   ├── agent_shoes.md
│   └── agent_accessory.md
│
└── templates/           # 模板
```

---

## LLM Output 标准化

多 LLM 输出通过四层保障机制确保一致性。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LLM Output Standardization                            │
└─────────────────────────────────────────────────────────────────────────┘

Layer 1: Prompt Template
┌────────────────────────────────────────────────────────────────┐
│ {{.Instructions}}                                                │
│ Output Format:                                                   │
│ ```json                                                          │
│ { "items": [...], "reason": "..." }                            │
│ ```                                                              │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
Layer 2: JSON Schema / Tool Calling
┌────────────────────────────────────────────────────────────────┐
│ {                                                                │
│   "type": "object",                                             │
│   "properties": {                                                │
│     "items": { "type": "array" },                              │
│     "reason": { "type": "string" }                             │
│   }                                                              │
│ }                                                                │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
Layer 3: Output Parser & Validator
┌────────────────────┐    ┌────────────────────┐
│  Parser (解析)    │───▶│  Validator (校验)  │
│  - 提取 JSON      │    │  - Schema 验证     │
│  - 修复破损 JSON  │    │  - 自动重试       │
└────────────────────┘    └────────────────────┘
                              │
                              ▼
Layer 4: LLM Adapter Layer
┌────────────────────────────────────────────────────────────────┐
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                        │
│  │ Ollama  │  │ OpenAI  │  │ Anthropic│                        │
│  │ Adapter │  │ Adapter │  │ Adapter  │                        │
│  └─────────┘  └─────────┘  └─────────┘                        │
│                         (统一抽象，上层不依赖具体模型)            │
└────────────────────────────────────────────────────────────────┘
```

### 完整调用流程

```
LLM 输出
    │
    ▼
Parser 解析 ── 提取 JSON
    │ 失败
    ▼
Fix JSON ── 修复破损
    │ 失败
    ▼
Validator 校验 ── Schema 验证
    │ 失败
    ▼
Retry (3次) ── 自动重试
    │
    ▼
返回结构化结果
```

---

## 目录结构

```
src/
├── agents/
│   ├── leader/           # Leader Agent
│   └── sub/             # Sub Agent
│
├── workflow/            # 工作流引擎
│   ├── loader.go       # 加载工作流 (YAML/JSON)
│   ├── executor.go     # 执行引擎 (DAG)
│   └── registry.go     # Agent 注册表
│
├── agents/definition/  # Agent Markdown 定义
│   ├── agent_leader.md
│   ├── agent_top.md
│   └── ...
│
├── protocol/ahp/       # AHP Protocol
│
├── core/
│   ├── models/         # 数据模型
│   ├── errors/         # 错误码
│   └── registry/       # Agent 注册
│
├── llm/                # LLM 层
│   ├── adapter/        # 多模型适配器
│   ├── parser/         # Output Parser
│   ├── validator/      # Schema Validator
│   └── ollama/         # Ollama 实现
│
├── storage/postgres/   # PostgreSQL + pgvector
│
├── tools/resources/    # Tools
│
├── memory/context/     # Memory System
│
├── shutdown/           # 优雅退出
│
└── ratelimit/         # 限流/背压
```

---

## 技术栈

| 层级 | 技术选型 |
|------|----------|
| 语言 | golang |
| LLM | Ollama (gpt-oss:20b / llama3.2:3b) |
| 协议 | AHP (自定义) |
| 存储 | PostgreSQL + pgvector |
| 并发 | ThreadPoolExecutor |
| 消息队列 | In-Memory (asyncio.Queue) |

---

## 消息格式 (AHP Protocol)

```python
class AHPMessage:
    message_id: str
    method: AHPMethod        # TASK, RESULT, PROGRESS, ACK, HEARTBEAT
    agent_id: str            # 发送方
    target_agent: str        # 接收方
    task_id: str             # 任务ID
    session_id: str          # 会话ID
    payload: Dict             # 消息内容
    timestamp: datetime
```

---

## 错误码体系

### 错误码规范

```
格式: XX-YYY-ZZZ
  - XX:   模块代码 (01-Agent, 02-Protocol, 03-Storage, 04-LLM, 05-Tools)
  - YYY:  错误类型 (001-099 系统级, 100-199 业务级)
  - ZZZ:  具体错误序号
```

### 错误码表

| 错误码 | 名称 | 说明 | 可重试 | 最大重试次数 |
|--------|------|------|--------|--------------|
| **01-Agent** |
| 01-001 | AgentNotFound | Agent 未注册 | 否 | 0 |
| 01-002 | AgentTimeout | Agent 执行超时 | 是 | 3 |
| 01-003 | AgentPanic | Agent 内部 panic | 是 | 2 |
| 01-004 | TaskQueueFull | 任务队列满 | 是 | 5 |
| 01-005 | DependencyCycle | 任务依赖循环 | 否 | 0 |
| **02-Protocol** |
| 02-001 | InvalidMessage | 消息格式错误 | 否 | 0 |
| 02-002 | MessageTimeout | 消息发送超时 | 是 | 3 |
| 02-003 | HeartbeatMissed | 心跳丢失 | 是 | 5 |
| **03-Storage** |
| 03-001 | DBConnectionFailed | 数据库连接失败 | 是 | 3 |
| 03-002 | QueryFailed | 查询失败 | 是 | 2 |
| 03-003 | VectorSearchFailed | 向量搜索失败 | 是 | 2 |
| **04-LLM** |
| 04-001 | LLMRequestFailed | LLM 请求失败 | 是 | 3 |
| 04-002 | LLMTimeout | LLM 响应超时 | 是 | 2 |
| 04-003 | LLMQuotaExceeded | 配额超限 | 否 | 0 |

### 统一错误处理

```go
type ErrorCode struct {
    Code       string                 `json:"code"`
    Message    string                 `json:"message"`
    Module     string                 `json:"module"`
    Retry      bool                   `json:"retry"`
    RetryMax   int                    `json:"retry_max"`
    Backoff    time.Duration          `json:"backoff"`
}

type AppError struct {
    Code    ErrorCode
    Err     error
    Stack   string
    Context map[string]interface{}
}
```

---

## 优雅 Shutdown 流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Shutdown Signal Flow                               │
└─────────────────────────────────────────────────────────────────────────┘

                       SIGTERM / SIGINT
                              │
                              ▼
                 ┌────────────────────────┐
                 │  Phase 1: Stop Accept  │
                 │  停止接收新请求/任务     │
                 └───────────┬────────────┘
                             │
                             │ 等待 30s (grace period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 2: Cancel       │
                 │  取消所有 Agent 上下文   │
                 └───────────┬────────────┘
                             │
                             │ 等待 60s (shutdown period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 3: Drain Queues │
                 │  处理完积压消息          │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 4: Save State   │
                 │  落盘内存状态            │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 5: Close        │
                 │  关闭 DB/连接池         │
                 └───────────┬────────────┘
                             │
                             ▼
                          EXIT 0
```

### Shutdown 阶段状态

| 阶段 | 状态码 | 说明 |
|------|--------|------|
| Running | 0 | 正常运行 |
| Stopping | 1 | 停止接收新请求 |
| Draining | 2 | 处理积压任务 |
| Exiting | 3 | 清理退出 |

### 实现要点

- **Grace Period**: 30秒，允许新请求完成当前处理
- **Shutdown Period**: 60秒，等待队列清空
- **Force Timeout**: 超过总时间后强制退出
- **Signal 捕获**: 同时支持 SIGTERM（推荐）和 SIGINT

---

## 限流与背压机制

### 限流策略

| 场景 | 限流方式 | 阈值 |
|------|----------|------|
| Agent 并发数 | 信号量 (Semaphore) | 每 Agent 最大 10 并发 |
| 任务队列 | 队列长度限制 | 单队列最大 1000 条 |
| LLM 请求 | 令牌桶 (Token Bucket) | 每秒 10 请求 |
| 全局 QPS | 滑动窗口 | 系统最大 100 QPS |

---

## 数据库连接池设计

采用"谁用谁连接，用完释放"的原则，避免长时间占用数据库连接资源。

### 传统模式 vs 连接池模式

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    数据库连接管理模式对比                                  │
└─────────────────────────────────────────────────────────────────────────┘

传统模式 (长连接):
┌──────────┐         ┌──────────┐         ┌──────────┐
│  Agent 1 │────────▶│   DB     │◀────────│  Agent 2 │
│          │  占用    │  (长连接)│   占用   │          │
└──────────┘         └──────────┘         └──────────┘
                           │
                           │ 连接一直保持，资源浪费
                           ▼
                    ┌──────────┐
                    │  Agent 3 │────────▶ 等待...
                    └──────────┘

连接池模式 (用完释放):
┌──────────┐         ┌──────────┐         ┌──────────┐
│  Agent 1 │────────▶│         │◀────────│  Agent 2 │
│  获取连接 │         │ Connection│         │ 获取连接  │
│  使用归还 │         │   Pool   │         │ 使用归还  │
└──────────┘         │         │         └──────────┘
                     └──────────┘
                           │
                           │ 按需获取，资源高效利用
                           ▼
                    ┌──────────┐
                    │  Agent 3 │────────▶ 立即获取
                    └──────────┘
```

### 连接池设计

```go
// 连接池管理器
type ConnectionPool struct {
    maxOpen    int           // 最大打开连接数
    maxIdle    int           // 最大空闲连接数
    maxLifetime time.Duration // 连接最大生命周期
    
    mu         sync.Mutex
    openCount  int           // 当前打开的连接数
    idleCount  int           // 当前空闲的连接数
    connections chan *DBConn  // 连接池队列
}

// 获取连接
func (p *ConnectionPool) Get(ctx context.Context) (*DBConn, error) {
    select {
    case conn := <-p.connections:
        // 从池中获取空闲连接
        if conn.IsValid() {
            return conn, nil
        }
        // 连接已过期，重新创建
        return p.createConn()
        
    case <-ctx.Done():
        return nil, ctx.Err()
        
    default:
        // 池中没有空闲连接
        if p.openCount >= p.maxOpen {
            // 达到最大连接数，等待
            return p.waitForConnection(ctx)
        }
        // 创建新连接
        return p.createConn()
    }
}

// 归还连接
func (p *ConnectionPool) Put(conn *DBConn) error {
    if !conn.IsValid() {
        // 连接已失效，关闭
        conn.Close()
        p.mu.Lock()
        p.openCount--
        p.mu.Unlock()
        return nil
    }
    
    // 放回池中
    select {
    case p.connections <- conn:
        return nil
    default:
        // 池已满，关闭连接
        conn.Close()
        p.mu.Lock()
        p.openCount--
        p.mu.Unlock()
        return nil
    }
}
```

### Agent 使用模式

```go
// Agent 中使用连接池
func (a *SubAgent) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
    // 从池中获取连接
    conn, err := pool.Get(ctx)
    if err != nil {
        return nil, err
    }
    defer pool.Put(conn)  // 用完归还
    
    // 使用连接执行操作
    result, err := a.doQuery(ctx, conn, task)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### 连接池配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| max_open | 25 | 最大打开连接数 |
| max_idle | 10 | 最大空闲连接数 |
| conn_max_lifetime | 5m | 连接最大生命周期 |
| conn_max_idle_time | 1m | 空闲连接最大存活时间 |
| max_wait_time | 30s | 获取连接最大等待时间 |

### 监控指标

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| db_open_connections | 当前打开的连接数 | > 20 |
| db_idle_connections | 当前空闲的连接数 | < 2 |
| db_wait_count | 等待连接次数 | > 100 |
| db_wait_duration | 等待连接总时长 | > 1s |

### 背压机制

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Backpressure Flow                               │
└─────────────────────────────────────────────────────────────────────────┘

    请求入口                    队列层                      Agent 层
    ────────                   ──────                      ───────
    
    ┌───────┐                 ┌─────────────┐             ┌─────────┐
    │ Request│                 │             │             │         │
    └───┬───┘                 │  Queue      │             │  Agent  │
        │                     │  [500/1000] │             │  Pool   │
        │ 超过阈值             │             │             │         │
        ├────────────────────▶│  触发背压   │             │         │
        │                     │             │             │         │
        │ ◀───────────────────┤  拒绝/排队  │◀────────────┤         │
        │  429 Too Many       │             │  处理中     │         │
        │  + Retry-After     │             │             │         │
                                                
    背压策略:
    1. 队列 80%  →  告警通知
    2. 队列 90%  →  拒绝新任务 (429)
    3. 队列 100% →  触发 DLQ
```

### 实现方案

```go
// 令牌桶限流
type TokenBucket struct {
    rate       float64       // 每秒令牌数
    capacity   int           // 桶容量
    tokens     float64
    lastUpdate time.Time
    mu         sync.Mutex
}

// 背压控制
type Backpressure struct {
    queueLimit    int           // 队列上限
    currentLoad   atomic.Int32   // 当前负载
    rejectionRate float64       // 拒绝率阈值
    
    // 响应头
    RetryAfter   time.Duration  // 建议重试时间
    RetryCount   int            // 已重试次数
}

// 限流策略选择
var LimiterStrategy = map[string]Limiter{
    "llm":    NewTokenBucket(10, 50),   // LLM 请求
    "agent":  NewSemaphore(10),          // Agent 并发
    "global": NewSlidingWindow(100),    // 全局 QPS
}
```

### 监控指标

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| queue_depth | 队列深度 | > 800 |
| rejection_rate | 拒绝率 | > 5% |
| latency_p99 | 延迟 P99 | > 5s |
| active_agents | 活跃 Agent 数 | < 50% 利用率 |

---

## 生产环境补充建议

### 可观测性

- **日志**: 结构化 JSON 日志，分级输出 (DEBUG/INFO/WARN/ERROR)
- **指标**: Prometheus + Grafana 面板
- **链路**: OpenTelemetry 分布式追踪

### 扩展性预留

- 消息队列支持 Redis Stream 替换
- 存储层支持多数据源切换
- Agent 支持动态注册与发现

---

