# GoAgent Bugs 待修复清单

> 生成时间: 2026-05-01
> 状态说明: ✅ 已修复 | 🔧 待修复 | ⚠️ 部分修复

---

## 🔴 Critical

### C1. `extractJSON` 不处理 JSON 字符串内的 `{}` `}`
- **文件**: `internal/llm/output/parser.go:134-176`
- **问题**: 括号匹配算法没有考虑 JSON 字符串内的引号。当 LLM 输出的 JSON 值中包含 `{` 或 `}` 时（如 `"description": "use a {brace}"`），深度计算出错，提取出无效 JSON
- **影响**: LLM 输出包含花括号的场景下解析失败
- **建议**: 使用完整的 JSON tokenizer 替代简单的括号匹配，或引入第三方 JSON 提取库
- **工作量**: 大

### C2. `fixJSONString` 正则破坏有效 JSON（已缓解但未根治）
- **文件**: `internal/llm/output/parser.go:180-226`
- **问题**: `singleQuote`、`singleLineComment`、`unquotedKey` 正则不区分 JSON 结构和字符串值，可能破坏包含 `'`、`//`、`key:` 的字符串值。当前已添加 `json.Valid` 前置检查（有效 JSON 直接返回），但 LLM 输出的 JSON 本身无效时仍可能被错误"修复"
- **影响**: LLM 输出无效 JSON 时，修复可能引入新的错误
- **建议**: 长期方案是替换为基于 AST 的 JSON 修复；短期可接受当前行为（best-effort）
- **工作量**: 大

---

## 🟡 Medium

### M1. `ProcessStream` 逐个 Dispatch 失去并行性
- **文件**: `internal/agents/leader/agent.go:564-591`
- **问题**: `ProcessStream` 中将每个任务单独传给 `Dispatch`（`Dispatch(ctx, []*models.Task{task})`），每次只调度一个任务。而 `Process` 方法将所有任务一次性传给 `Dispatch` 以并行执行
- **影响**: 流式接口的执行效率远低于非流式接口，N 个任务变成 O(N) 串行
- **建议**: 在 `ProcessStream` 中也一次性 `Dispatch` 所有任务，然后逐个发送事件；或者使用 worker pool 并行执行但保持事件顺序
- **工作量**: 中

### M2. `Process` TOCTOU 状态检查竞态
- **文件**: `internal/agents/leader/agent.go:378-386`
- **问题**: 两次调用 `a.Status()` 之间状态可能被另一个 goroutine 改变。高并发下多个 goroutine 可能同时通过检查并执行 `Start()` 或 `Process()`
- **影响**: 并发调用可能导致状态不一致
- **建议**: 使用 CAS 模式或 single-flight 确保原子性
- **工作量**: 中

### M3. `SecretRepository.Import` 部分提交违反原子性
- **文件**: `internal/storage/postgres/repositories/secret_repository.go:554-626`
- **问题**: 单个 secret 处理失败时使用 `continue` 跳过并记录错误，而非回滚事务。如果导入 10 个 secret 其中 5 个失败，事务仍会提交。注释声称 "atomicity: all-or-nothing" 但实际行为并非如此
- **影响**: 部分导入成功、部分失败时数据库处于不一致状态
- **建议**: 在任何单个失败时立即返回 error 触发回滚，实现真正的 all-or-nothing 语义
- **工作量**: 小

### M4. `registerAgents` 闭包捕获循环变量
- **文件**: `api/client/workflow.go:91-107`
- **问题**: `range` 循环中的 `agentConfig` 被闭包捕获。Go 1.22 之前所有闭包共享同一地址，最终所有 agent 使用最后一个配置
- **影响**: Go < 1.22 时所有 agent 使用相同配置
- **建议**: 如果项目要求 Go < 1.22 兼容，需在循环内添加 `agentConfig := agentConfig` 副本；否则确认 go.mod 中 Go 版本 >= 1.22
- **工作量**: 小（需确认 Go 版本要求）

### M5. `WorkflowAgentExecutor` 未使用 timeout/maxRetries
- **文件**: `api/client/workflow.go:172-260`
- **问题**: 结构体有 `timeout` 和 `maxRetries` 字段，在构造时被赋值，但 `Process` 方法中完全没有使用。LLM 调用没有超时控制，失败时也没有重试
- **影响**: 配置的超时和重试参数被忽略，可能导致请求无限挂起
- **建议**: 在 `Process` 中使用 `context.WithTimeout` 和重试循环
- **工作量**: 中

### M6. `Execute` 错误处理反模式
- **文件**: `api/service/graph/service.go:136-146`
- **问题**: `g.Execute` 返回错误时，将错误信息放入 `ExecuteResponse.Error` 字段但返回 `nil` error。调用者可能只检查返回的 error 而忽略 `ExecuteResponse.Error`
- **影响**: 错误信息可能被调用方遗漏
- **建议**: 统一错误处理策略，要么通过返回值要么通过 error
- **工作量**: 小

### M7. `stepG` errgroup 未调用 `Wait()`
- **文件**: `internal/workflow/engine/executor.go:236-275`
- **问题**: `stepG` 创建后 `stepG.Wait()` 从未被调用。step goroutine 内部返回的 error 永远不会被收集
- **影响**: step goroutine 的错误被静默丢弃
- **建议**: 要么调用 `stepG.Wait()` 并处理错误，要么移除 errgroup 改用普通 goroutine
- **工作量**: 小

### M8. 失败步骤的空输出写入 OutputStore 导致下游继续执行
- **文件**: `internal/workflow/engine/executor.go:382-393`
- **问题**: 步骤失败后，空字符串输出仍被写入 `OutputStore`。后续依赖此步骤的其他步骤通过 `outputStore.Get(dep)` 获取到空输出并继续执行，违反 DAG 依赖语义
- **影响**: 失败步骤的下游不应该继续执行
- **建议**: 失败时不写入 OutputStore，或在 `resolveInput` 中检查依赖步骤是否成功
- **工作量**: 中

### M9. `HeartbeatSender` 不可重启
- **文件**: `internal/protocol/ahp/heartbeat.go:162-216`
- **问题**: `startOnce` 确保 `Start` 只能执行一次，`Stop` 后无法再次 `Start`
- **影响**: 限制了 agent 的生命周期管理灵活性
- **建议**: 在 `Stop` 中重置 `startOnce`，或使用 `atomic.Bool` 替代
- **工作量**: 小

### M10. `WorkflowAgentExecutor.Process` 中 llmService nil 风险
- **文件**: `api/client/workflow.go:233`
- **问题**: 如果 `w.client.llmService` 为 nil（客户端配置不完整），`e.llmService.GenerateSimple()` 会导致 nil pointer panic
- **影响**: 配置不完整时 panic
- **建议**: 在 `Process` 开头添加 nil 检查
- **工作量**: 小

---

## 🟢 Minor

### m1. 三个 ILIKE 转义函数重复定义
- **文件**: `knowledge_repository.go`, `experience_repository.go`, `tool_repository.go`
- **问题**: `escapeILIKEPattern`、`escapeExpILIKEPattern`、`escapeToolILIKEPattern` 功能完全相同
- **建议**: 提取到共享的 `vector_utils.go` 或新建 `sql_utils.go`

### m2. `float64ToVectorString` 与 `postgres.FormatVector` 重复
- **文件**: `knowledge_repository.go:42-52` vs `vector_utils.go:13-33`
- **问题**: 两个函数功能相同但实现不同
- **建议**: 统一使用 `postgres.FormatVector`，删除 `float64ToVectorString`

### m3. `parseVectorString` 在两个文件中重复
- **文件**: `knowledge_repository.go` vs `distilled_memory_repository.go`
- **问题**: 注释中说"为了避免循环引用"而重复定义
- **建议**: 提取到共享的 `vector_utils.go`

### m4. `DistilledMemoryRepository.dbPool` 字段未使用
- **文件**: `internal/storage/postgres/repositories/distilled_memory_repository.go:38-39`
- **问题**: `dbPool *sql.DB` 在构造函数中被赋值但从未使用
- **建议**: 移除该字段

### m5. Router 缺少 panic recovery 中间件
- **文件**: `api/router/router.go`
- **问题**: `ServeHTTP` 直接透传给 `mux`，没有 panic recovery 保护
- **影响**: handler panic 会导致整个服务崩溃
- **建议**: 添加 `http Recoverer` 中间件

### m6. SSE handler 缺少 CORS headers
- **文件**: `api/handler/stream.go`
- **问题**: SSE 端点没有设置 `Access-Control-Allow-Origin` 等 CORS headers
- **影响**: 前端从不同域访问时被浏览器阻止

### m7. SSE handler 缺少事件 ID 字段
- **文件**: `api/handler/stream.go`
- **问题**: SSE 规范支持 `id:` 字段用于断线重连，当前实现没有发送
- **影响**: 客户端断线重连后无法恢复

### m8. eval 包缺少包注释
- **文件**: `internal/eval/*.go`
- **问题**: 所有文件缺少 `// Package eval ...` 形式的包注释
- **影响**: `godoc` 不生成文档

### m9. `TestResult.Metrics` 字段从未被填充
- **文件**: `internal/eval/types.go:43`
- **问题**: 初始化为空 map 但从未写入数据
- **建议**: 移除或在评估阶段填充

### m10. `Pool.IsHealthy` 用连接数判断而非实际查询
- **文件**: `internal/storage/postgres/pool.go:127-130`
- **问题**: `OpenConnections == MaxOpenConns` 时返回 false，但连接池满不一定不健康
- **建议**: 改为执行 `SELECT 1` 检查

### m11. `generateMessageID` 多实例部署时可能碰撞
- **文件**: `internal/protocol/ahp/message.go:195-199`
- **问题**: 全局原子计数器在多实例间不共享，随机后缀只有 10000 种可能
- **影响**: 多实例部署时 messageID 可能重复
- **建议**: 使用 UUID 替代自定义格式

### m12. DLQ.Add 丢弃最老消息时内存泄漏
- **文件**: `internal/protocol/ahp/dlq.go:51-53`
- **问题**: `d.messages = d.messages[1:]` 不释放底层数组对第一个元素的引用
- **建议**: 先 `d.messages[0] = nil` 再切片

### m13. `API_REFERENCE.md` 代码示例与实际签名不符
- **文件**: `docs/API_REFERENCE.md`
- **问题**: `RunSuite` 示例中 `suite` 是 `*TestSuite`（指针），但 `RunSuite` 接收 `TestSuite`（值类型）；`RegisterFactory` 示例忽略了返回的 error
- **建议**: 更新文档

### m14. 自定义 `min` 函数与 Go 1.21 内置冲突
- **文件**: `internal/storage/postgres/repositories/distilled_memory_repository.go:559-563`
- **问题**: Go 1.21+ 内置 `min` 泛型函数，自定义版本可能导致编译错误
- **建议**: 删除自定义 `min`，使用内置版本

### m15. `containsSQLInjectionPatterns` 误报率高
- **文件**: `internal/storage/postgres/security.go:108-137`
- **问题**: 黑名单包含 `SELECT`、`UPDATE` 等常见英文单词，正常文本会被误报
- **建议**: 仅在标识符验证中使用，不用于一般用户输入

### m16. `DSN()` 中 Host/User 未转义
- **文件**: `internal/storage/postgres/config.go:108-113`
- **问题**: 密码已转义，但 Host 和 User 字段也可能包含特殊字符
- **影响**: 低（这些字段通常由配置文件控制）
- **建议**: 统一转义所有字段

---

## ✅ 已修复（本轮 Review 中发现并修复的问题）

以下问题已在代码中修复，此清单仅作记录：

| ID | 问题 | 文件 |
|----|------|------|
| F1 | `Wrapf(nil, ...)` 吞掉 HTTP 错误 | `ollama.go`, `openai.go` |
| F2 | `ProcessStream` 缺少 memory 管理 | `leader/agent.go` |
| F3 | `http.Client.Timeout` 与流式传输冲突 | `client.go` + 3 adapters |
| F4 | SSE handler 请求体无大小限制 | `stream.go` |
| F5 | `time.Duration` YAML 解析失败 | `eval/types.go` |
| F6 | `PluginRegistry` 无锁并发访问 | `factory.go` |
| F7 | `distillWg.Add` 竞态 panic | `leader/agent.go` |
| F8 | `GenerateMarkdown` 索引越界 | `eval/report.go` |
| F9 | 蒸馏 context 生命周期错误 | `leader/agent.go` |
| F10 | Transport nil 隐式依赖 | 3 output adapters |
| F11 | `RegisterExecutor` 数据竞争 | `dispatcher.go` |
| F12 | `Dispatch` 错误传播失效 | `dispatcher.go` |
| F13 | `regexp.MustCompile` 可能 panic | `validator.go` |
| F14 | `RegisterValidator` 数据竞争 | `validator.go` |
| F15 | `ValidateRecommendResult` nil item | `validator.go` |
| F16 | completed map data race | `executor.go` |
| F17 | Graph 无环检测 | `graph/executor.go` |
| F18 | `AgentNode` nil input | `graph/node.go` |
| F19 | `fixJSONString` 破坏有效 JSON | `parser.go` |
| F20 | `parseCondition` 操作符匹配顺序 | `graph_builder.go` |
| F21 | `Enqueue/IsFull` TOCTOU 消息丢失 | `protocol.go` |
| F22 | `ProcessStream` 重复事件 | `workflow.go` |
| F23 | `CheckTimeouts` 重复报告离线 agent | `heartbeat.go` |
| F24 | `IsEmpty/Available` 不考虑 backupBuffer | `queue.go` |
| F25 | `taskMemory.Stop()` 未调用 | `manager_impl.go` |
| F26 | `searchVector` 未使用 embedding 缓存 | `retrieval_service.go` |
| F27 | `ReloadInterval` 无默认值 | `config.go` |
| F28 | `NewAgentTestRunner` nil executor | `agent_runner.go` |
| F29 | SET tenant_id SQL 注入 | `distilled_memory_repository.go` |
| F30 | ILIKE 通配符注入 | 3 repositories |
| F31 | 连接池租户上下文泄漏 | `distilled_memory_repository.go` |
| F32 | embedding 类型不匹配 | `knowledge_repository.go` |
| F33 | `UpdateEmbedding` 缺少类型转换 | `knowledge_repository.go` |
| F34 | `Import` 事务 committed 标志 | `secret_repository.go` |
| F35 | 加密密钥长度验证 | `secret_repository.go` |
| F36 | DSN 密码转义 | `config.go` |
| F37 | `QueryRow` 连接泄漏 | `pool.go` |
| F38 | 错误消息缺少 StatusCode | `ollama/openai/openrouter.go` |
| F39 | `filepath.Join` 替代字符串拼接 | `loader.go` |

---

## 🔴 Critical（第二轮扫描新增）

### C3. `LogTracer` Record 方法对 nil 指针解引用 panic
- **文件**: `internal/observability/log.go:31-107`
- **问题**: `RecordLLMCall`/`RecordToolCall`/`RecordAgentStep`/`RecordError` 均未对传入参数做 nil 检查。tracer 通常在 defer 或错误处理路径中被调用，nil 指针会导致原始错误信息丢失
- **建议**: 每个方法开头添加 `if call == nil { return }` 等检查

### C4. `CacheWithTTL.Get` 锁升级导致竞态条件
- **文件**: `internal/memory/context/cache.go:221-239`
- **问题**: 先 `RLock` 读取，发现过期后释放 `RLock` 再获取 `Lock` 删除。在两次锁之间，另一个 goroutine 可能已删除该 key 或替换为新值，导致删除有效的新条目
- **建议**: 不在 `Get` 中删除过期项，交给 cleanup goroutine 惰性清理；或始终使用 `Lock()`

### C5. `RAG.Add`/`RAG.Delete` 内存与持久存储不一致
- **文件**: `internal/memory/context/rag.go:87-107, 196-216`
- **问题**: 条目先添加到内存 map，再尝试持久化。如果持久化失败返回 error，内存中已存在该条目，导致内存与数据库不一致
- **建议**: 先持久化再更新内存，或持久化失败时回滚内存

### C6. `TaskMemory.Stop` 不等待清理 goroutine 退出
- **文件**: `internal/memory/context/task.go:81-85`
- **问题**: `Stop` 使用 `sync.Once` 关闭 `cleanupStopCh` 但不调用 `wg.Wait()`。对比 `SessionMemory.StopCleanup` 会等待 goroutine 退出。关闭后立即销毁 `TaskMemory` 会导致 use-after-free
- **建议**: 添加 `wg.Wait()` 调用

### C7. `TaskMemory.Distill` 释放锁后访问内部指针（数据竞争）
- **文件**: `internal/memory/context/task.go:312-338`
- **问题**: `RLock` 获取 `task` 指针后立即 `RUnlock`，然后在无锁状态下访问 `task.Input`、`task.Output` 等字段。另一个 goroutine 可能同时通过 `Set` 覆盖或 `Delete` 删除该 task
- **建议**: 在持有锁时复制所有需要的字段到局部变量

### C8. `ProductionMemoryManager` Start/Stop 状态机损坏
- **文件**: `internal/memory/production_manager.go:183-240`
- **问题**: `started` 和 `stopped` 是独立标志。`Start` → `Stop` → `Start` 序列中，第二次 `Start` 因 `stopped=true` 不执行，但 `started` 仍为 `true`。状态机无法正确重启
- **建议**: 使用单一状态字段或状态枚举

---

## 🟡 Medium（第二轮扫描新增）

### M11. `SemaphoreLimiter.Available()` 返回值语义完全相反
- **文件**: `internal/ratelimit/semaphore.go:99-101`
- **问题**: `len(l.sem)` 返回已获取的槽位数，但方法名 `Available` 语义应为可用槽位数。应返回 `cap(l.sem) - len(l.sem)`
- **影响**: 调用方依赖此值做决策时会得到相反的结果

### M12. `SemaphoreLimiter.Allow` 获取信号量后无法释放
- **文件**: `internal/ratelimit/semaphore.go:58-68`
- **问题**: `Allow` 获取槽位但 `Limiter` 接口没有 `Release` 方法，导致信号量泄漏
- **建议**: `Allow` 应仅检查是否有可用槽位，不实际获取

### M13. `Factory.Register` / `DefaultFactory` 无并发保护
- **文件**: `internal/ratelimit/limiter.go:81-83, 106`
- **问题**: `creators` map 在 `Register` 写入、`Create` 读取时无锁保护。运行时注册会导致 panic

### M14. `Backpressure.processLoop` 单线程串行处理，`maxActive` 形同虚设
- **文件**: `internal/ratelimit/backpressure.go:153-176`
- **问题**: 单 goroutine 串行处理请求，`active` 永远不超过 1。`maxActive` 参数未起到限制并发的作用
- **建议**: 使用 worker pool 模式

### M15. `AdaptiveLimiter.Increase/Decrease` 替换整个 limiter 丢失状态
- **文件**: `internal/ratelimit/backpressure.go:308-343`
- **问题**: 每次创建新的 `TokenBucketLimiter`，旧 limiter 中等待中的请求永远等下去
- **建议**: 使用 `SetRate` 方法修改速率

### M16. `RAG.evictOldest` 使用 Go map 随机迭代，不驱逐最旧条目
- **文件**: `internal/memory/context/rag.go:242-255`
- **问题**: 函数名为 `evictOldest` 但 Go map 迭代顺序随机，实际驱逐任意条目
- **建议**: 维护按时间排序的有序数据结构

### M17. `RAG.Search` 在持锁状态下执行网络 I/O
- **文件**: `internal/memory/context/rag.go:119-138`
- **问题**: `RLock` 下调用 `r.vectorSearch.Search`（Postgres 查询），数据库慢时阻塞所有其他操作
- **建议**: 在锁外执行数据库查询

### M18. `SimpleRetrievalService.SetConfig` 无并发保护
- **文件**: `internal/storage/postgres/services/simple_retrieval_service.go:303-305`
- **问题**: `s.config = config` 无锁保护，与 `Search` 并发读写导致数据竞争

### M19. `TaskMemory.GetSteps`/`GetResults` 返回内部切片引用
- **文件**: `internal/memory/context/task.go:205-244`
- **问题**: 返回内部切片允许调用者修改，与 `AddStep` 并发时产生数据竞争
- **建议**: 返回副本

### M20. `UserMemory.GetPreferences`/`GetHistory`/`GetStyleEvolution` 返回内部切片
- **文件**: `internal/memory/context/user.go:135-213`
- **问题**: 同 M19

### M21. `ProductionMemoryManager.CreateSession` session ID 可预测且可能碰撞
- **文件**: `internal/memory/production_manager.go:248`
- **问题**: `fmt.Sprintf("session_%d", time.Now().UnixNano())` 在高并发下可能碰撞
- **建议**: 使用 UUID

### M22. `ProductionMemoryManager.AddMessage` 匿名用户静默降级
- **文件**: `internal/memory/production_manager.go:321-323`
- **问题**: 会话不在缓存中时，消息被分配给 "anonymous"，静默丢失用户关联
- **建议**: 记录警告或返回错误

---

## 🟢 Minor（第二轮扫描新增）

### m17. `RAG.Search` 使用冒泡排序 O(n²)
- **文件**: `internal/memory/context/rag.go:159-165`
- **建议**: 使用 `sort.Slice`

### m18. `RAG.contains` 自定义实现，空子串返回 false（`strings.Contains` 返回 true）
- **文件**: `internal/memory/context/rag.go:280-291`

### m19. `UserMemory.Set` 覆盖现有用户不保留数据
- **文件**: `internal/memory/context/user.go:79-99`

### m20. `isPrecisionMode` 对包含 `:` 的普通文本过于激进
- **文件**: `internal/storage/postgres/services/simple_retrieval_service.go:154`

### m21. `WeightedSemaphoreLimiter` 的 `cond.Wait` 不响应 context 取消
- **文件**: `internal/ratelimit/semaphore.go:135-163`

### m22. `Cache.cleanupLoop` 当 `ttl` 极小时 `time.NewTicker(0)` 会 panic
- **文件**: `internal/memory/context/cache.go:129`

### m23. `LogTracer.WithTrace` 每次生成新 ID，不继承已有 trace ID
- **文件**: `internal/observability/noop.go:51-53, log.go:118-121`

### m24. `base.DefaultConfig` 未设置 ID 字段
- **文件**: `internal/agents/base/agent.go:86-93`

### m25. `AgentEvent` 缺少 `EventError` 类型
- **文件**: `internal/agents/base/agent.go:30-39`

### m26. `ProductionMemoryManager` 未验证 `embeddingClient` 是否为 nil
- **文件**: `internal/memory/production_manager.go`

### m27. `NewTokenBucketLimiter`/`NewSlidingWindowLimiter`/`NewSemaphoreLimiter` 未校验 nil config
- **文件**: `internal/ratelimit/token_bucket.go:21, sliding_window.go:21, semaphore.go:19`
