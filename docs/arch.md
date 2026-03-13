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
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                                    │
         │                                    ▼
         │                        ┌─────────────────────────┐
         │                        │     Vector DB            │
         │                        │  • Sessions             │
         │                        │  • Recommendations      │
         │                        │  • Memories (RAG)       │
         │                        └─────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Memory System                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                    │
│  │ SessionMemory│  │UserMemory   │  │TaskMemory   │                    │
│  │  (短期)      │  │ (长期)       │  │ (蒸馏)       │                    │
│  └─────────────┘  └─────────────┘  └─────────────┘                    │
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
| **并发模型** | `ThreadPoolExecutor` 派发任务到多个 Sub Agent |
| **通信协议** | In-Memory Message Queue + AHP 自定义协议 |
| **状态管理** | `SessionMemory` 短期会话 + `TaskMemory` 蒸馏 |
| **容错机制** | DLQ 存储失败消息，支持重试 |
| **任务协调** | Phase 1 (并行) → Phase 2 (依赖感知) |
| **扩展性** | 可动态注册新的 Agent 类型 |

---

## 目录结构

```
src/
├── agents/
│   ├── leader_agent.py      # Coordinator Actor
│   ├── sub_agent.py         # Worker Actor
│   └── resources.py         # Tools (fashion_search, weather, style)
├── protocol/
│   └── ahp.py              # AHP Protocol (消息定义与队列)
├── core/
│   ├── models.py           # 数据模型
│   ├── errors.py            # 错误定义
│   └── registry.py          # Agent 注册表
├── storage/
│   └── postgres.py         # PostgreSQL + pgvector
└── utils/
    ├── llm.py              # LLM 封装 (支持 Ollama)
    ├── context.py           # Memory System
    └── config.py            # 配置管理
```

---

## 技术栈

| 层级 | 技术选型 |
|------|----------|
| 语言 | Python 3.13 |
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

