# Bug Log

## Bug #1: Executor runSteps Function

### Date
2026-03-16

### Severity
High - 导致工作流执行超时和死锁

### Affected Files
- `internal/workflow/engine/executor.go`
- `internal/workflow/engine/executor_test.go`

### Bug Description

#### 症状
1. `TestExecutorCoverage/execute_workflow_with_dependencies` 测试超时（30秒）
2. `TestExecutorCoverage/execute_workflow_with_agent_error` 测试失败
3. `TestExecutorCoverage/execute_workflow_with_invalid_agent_type` 测试失败
4. `TestExecutorHelperFunctionsCoverage/execute_step_with_timeout` 测试失败

#### 错误信息
```
panic: test timed out after 30s
running tests:
    TestExecutorCoverage (30s)
    TestExecutorCoverage/execute_workflow_with_dependencies (30s)
```

### Root Cause Analysis

#### 1. runSteps 函数的并发控制逻辑缺陷

##### 问题 1：stepChan 只写入不读取
```go
// 错误的代码
stepChan <- stepID
// 从未从 stepChan 读取
```

这导致：
- 当 `len(stepChan) >= e.maxParallel` 时，无法继续提交新任务
- channel 满后整个工作流执行卡死

##### 问题 2：Execute 和 runSteps 都从 resultChan 读取
```go
// 在 Execute 函数中
case result := <-resultChan:

// 在 runSteps 函数中
case result := <-resultChan:
```

这导致：
- 两个 goroutine 竞争同一个 channel
- 结果可能被错误的消费者接收
- 主循环可能永远收不到结果

##### 问题 3：失败步骤没有正确返回错误
```go
// 错误的代码
if result.Status == StepStatusFailed {
    execution.Status = WorkflowStatusFailed
    execution.Error = result.Error
    // 没有返回错误给调用者
}
```

这导致：
- 步骤失败后，工作流继续执行
- 测试用例无法正确检测到失败

#### 2. 测试用例问题

##### 问题 1：缺少 Timeout 字段
```go
// 错误的测试代码
{
    ID:        "step1",
    Name:      "First Step",
    AgentType: "test-agent",
    Input:     "step1 input",
    // 缺少 Timeout 字段
}
```

这导致：
- `Timeout` 为 0
- `context.WithTimeout(ctx, 0)` 会立即取消 context
- agent 无法正常执行

##### 问题 2：超时测试不正确
```go
// 错误的测试代码
return NewMockAgent("test", "slow-agent", func(ctx context.Context, input any) (any, error) {
    time.Sleep(200 * time.Millisecond)
    // 没有检查 context 是否被取消
})
```

这导致：
- agent 不响应 context 取消
- 超时无法被正确检测

### Solution

#### 1. 重构 runSteps 函数

使用 `sync.WaitGroup` 替代复杂的 channel 机制：

```go
func (e *Executor) runSteps(
    ctx context.Context,
    execution *WorkflowExecution,
    workflow *Workflow,
    executionOrder []string,
    initialInput string,
    stepChan chan string,
    resultChan chan *StepResult,
    errChan chan error,
) {
    stepIndex := 0
    completed := make(map[string]bool)
    processed := make(map[string]bool)
    var mu sync.Mutex
    var wg sync.WaitGroup

    // Submit steps according to execution order
    for stepIndex < len(executionOrder) {
        stepID := executionOrder[stepIndex]
        step := e.findStep(workflow.Steps, stepID)

        // Check if step can be executed based on dependencies
        if !e.canExecute(step, completed, &mu) {
            mu.Lock()
            alreadyProcessed := processed[stepID]
            mu.Unlock()

            if alreadyProcessed {
                stepIndex++
                continue
            }

            wg.Wait()
            continue
        }

        // Wait for capacity
        if len(stepChan) >= e.maxParallel {
            <-stepChan
        }

        stepChan <- stepID
        stepIndex++

        wg.Add(1)
        go func(sid string) {
            defer func() {
                <-stepChan
                wg.Done()

                if r := recover(); r != nil {
                    mu.Lock()
                    processed[sid] = true
                    mu.Unlock()

                    result := &StepResult{
                        StepID: sid,
                        Status: StepStatusFailed,
                        Error:  fmt.Sprintf("panic: %v", r),
                    }
                    resultChan <- result
                }
            }()

            result := e.executeStep(ctx, workflow, sid, initialInput, completed)

            mu.Lock()
            processed[sid] = true
            if result.Status == StepStatusCompleted {
                completed[sid] = true
            }
            mu.Unlock()

            resultChan <- result
        }(stepID)
    }

    wg.Wait()

    mu.Lock()
    allCompleted := len(completed) == len(workflow.Steps)
    mu.Unlock()

    if allCompleted {
        close(resultChan)
        return
    }

    pending := false
    for _, sid := range executionOrder {
        mu.Lock()
        isProcessed := processed[sid]
        mu.Unlock()

        if !isProcessed {
            step := e.findStep(workflow.Steps, sid)
            if !e.canExecute(step, completed, &mu) {
                pending = true
                break
            }
        }
    }

    if pending {
        errChan <- ErrWorkflowIncomplete
        close(resultChan)
    } else {
        close(resultChan)
    }
}
```

关键改进：
1. 使用 `sync.WaitGroup` 管理 goroutine 生命周期
2. 引入 `processed` map 跟踪所有已处理的步骤
3. 正确读取 `stepChan` 来释放容量
4. 简化事件驱动逻辑，移除 `wakeup` channel

#### 2. 修复 Execute 函数

当收到失败步骤时立即返回错误：

```go
case result := <-resultChan:
    stepResults = append(stepResults, result)
    execution.StepStates[result.StepID] = &StepState{
        StepID:     result.StepID,
        Status:     result.Status,
        Output:     result.Output,
        Error:      result.Error,
        FinishedAt: time.Now(),
    }
    if result.Status == StepStatusFailed {
        execution.Status = WorkflowStatusFailed
        execution.Error = result.Error
        execution.FinishedAt = time.Now()
        return &WorkflowResult{
            ExecutionID: execution.ID,
            WorkflowID:  workflow.ID,
            Status:      WorkflowStatusFailed,
            Error:       result.Error,
            Duration:    execution.FinishedAt.Sub(execution.StartedAt),
            Steps:       stepResults,
        }, fmt.Errorf("step %s failed: %s", result.StepID, result.Error)
    }
```

#### 3. 修复测试用例

##### 为所有步骤添加 Timeout 字段
```go
workflow := &Workflow{
    ID:   "wf2",
    Name: "Test Workflow with Dependencies",
    Steps: []*Step{
        {
            ID:        "step1",
            Name:      "First Step",
            AgentType: "test-agent",
            Input:     "step1 input",
            Timeout:   10 * time.Second, // 添加 Timeout
        },
        {
            ID:        "step2",
            Name:      "Second Step",
            AgentType: "test-agent",
            DependsOn: []string{"step1"},
            Timeout:   10 * time.Second, // 添加 Timeout
        },
        {
            ID:        "step3",
            Name:      "Third Step",
            AgentType: "test-agent",
            DependsOn: []string{"step1", "step2"},
            Timeout:   10 * time.Second, // 添加 Timeout
        },
    },
}
```

##### 修复超时测试
```go
registry.Register("slow-agent", func(ctx context.Context, config interface{}) (base.Agent, error) {
    return NewMockAgent("test", "slow-agent", func(ctx context.Context, input any) (any, error) {
        select {
        case <-time.After(200 * time.Millisecond):
            return &models.RecommendResult{
                Items: []*models.RecommendItem{
                    {
                        ItemID:      "item1",
                        Name:        "Test Item",
                        Description: "Test result",
                        Price:       100.0,
                    },
                },
            }, nil
        case <-ctx.Done():
            return nil, ctx.Err() // 正确响应 context 取消
        }
    }), nil
})
```

### Verification

#### 测试结果
所有测试通过：
- ✅ `TestExecutorCoverage` - 6/6 子测试通过
- ✅ `TestExecutorHelperFunctionsCoverage` - 5/5 子测试通过
- ✅ `TestRetryLogicCoverage` - 3/3 子测试通过
- ✅ `TestWorkflowExecutionStateCoverage` - 1/1 子测试通过
- ✅ `TestDAGCoverage` - 9/9 子测试通过
- ✅ `TestAgentRegistryCoverage` - 7/7 子测试通过
- ✅ `TestOutputStoreCoverage` - 5/5 子测试通过
- ✅ `TestErrorDefinitionsCoverage` - 1/1 子测试通过
- ✅ `TestWorkflowStatusConstantsCoverage` - 2/2 子测试通过
- ✅ `TestStepStatusConstantsCoverage` - 1/1 子测试通过
- ✅ `TestWorkflowTypesCoverage` - 10/10 子测试通过

#### 代码质量检查
- ✅ `go vet` - 无警告
- ✅ `gofmt` - 格式正确
- ✅ `goimports` - 导入正确

---

## Bug #2: Data Race Conditions in Tests

### Date
2026-03-16

### Severity
High - 数据竞争导致测试在 `go test -race` 模式下失败

### Affected Files
- `internal/core/errors/error_scenarios_test.go`

### Bug Description

#### 症状
执行 `make test-race` 时检测到多个数据竞争：

1. `TestRealHeartbeatMissed` - 数据竞争
2. `TestRealConcurrentErrorHandling` - 多个数据竞争

#### 错误信息
```
WARNING: DATA RACE
Write at 0x00c00019411f by goroutine 42:
  goagent/internal/core/errors.TestRealHeartbeatMissed.func1.1()
      /Users/scc/go/src/styleagent/internal/core/errors/error_scenarios_test.go:534 +0x84

Previous read at 0x00c00019411f by goroutine 41:
  goagent/internal/core/errors.TestRealHeartbeatMissed.func1.2()
      /Users/scc/go/src/styleagent/internal/core/errors/error_scenarios_test.go:551 +0x168

==================
WARNING: DATA RACE
Read at 0x00c00029c3d0 by goroutine 57:
  goagent/internal/core/errors.TestRealConcurrentErrorHandling.func1.2()
      /Users/scc/go/src/styleagent/internal/core/errors/error_scenarios_test.go:756 +0x20c

Previous write at 0x00c00029c3d0 by goroutine 56:
  goagent/internal/core/errors.TestRealConcurrentErrorHandling.func1.2()
      /Users/scc/go/src/styleagent/internal/core/errors/error_scenarios_test.go:756 +0x21c
```

### Root Cause Analysis

#### 1. TestRealHeartbeatMissed - heartbeatStopped 变量竞争

##### 问题代码
```go
var heartbeatStopped bool

// Goroutine 1: 写入
go func() {
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    heartbeatStopped = true  // ← 写入操作
}()

// Goroutine 2: 读取
heartbeatMonitor := func(ctx context.Context) error {
    ticker := time.NewTicker(80 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-heartbeatCh:
            missedCount = 0
        case <-ticker.C:
            if heartbeatStopped {  // ← 读取操作
                missedCount++
                if missedCount >= 2 {
                    return fmt.Errorf("heartbeat missed for %d cycles", missedCount)
                }
            }
        }
    }
}
```

##### 竞争原因
- 多个 goroutine 同时访问 `heartbeatStopped` 变量
- 一个 goroutine 写入，另一个 goroutine 读取
- 没有同步机制保护共享变量

#### 2. TestRealConcurrentErrorHandling - 多个变量竞争

##### 问题代码
```go
var requestCount int
var successCount int
var errorCount int

// HTTP handler: 读取和写入 requestCount
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    requestCount++  // ← 写入操作，无保护
    if requestCount%3 == 0 {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}))

// Worker goroutines: 读取和写入 successCount/errorCount
for i := 0; i < concurrency; i++ {
    go func(id int) {
        result := handler.RetryWithBackoff(context.Background(), appErr, 0, makeRequest)
        
        if result != nil {
            errorCount++  // ← 写入操作，无保护
            errorsCh <- result
        } else {
            successCount++  // ← 写入操作，无保护
            errorsCh <- nil
        }
    }(i)
}
```

##### 竞争原因
- `requestCount`：多个 HTTP 请求同时修改
- `successCount`：多个 worker goroutine 同时修改
- `errorCount`：多个 worker goroutine 同时修改
- 没有使用互斥锁保护共享变量

### Solution

#### 1. 修复 TestRealHeartbeatMissed

添加互斥锁保护 `heartbeatStopped` 变量：

```go
var heartbeatStopped bool
var heartbeatStoppedMu sync.Mutex

// 写入时加锁
go func() {
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    
    heartbeatStoppedMu.Lock()
    heartbeatStopped = true
    heartbeatStoppedMu.Unlock()
}()

// 读取时加锁
heartbeatMonitor := func(ctx context.Context) error {
    ticker := time.NewTicker(80 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-heartbeatCh:
            missedCount = 0
        case <-ticker.C:
            heartbeatStoppedMu.Lock()
            stopped := heartbeatStopped
            heartbeatStoppedMu.Unlock()
            
            if stopped {
                missedCount++
                if missedCount >= 2 {
                    return fmt.Errorf("heartbeat missed for %d cycles", missedCount)
                }
            }
        }
    }
}
```

#### 2. 修复 TestRealConcurrentErrorHandling

添加互斥锁保护所有共享变量：

```go
var requestCount int
var requestCountMu sync.Mutex

server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    requestCountMu.Lock()
    requestCount++
    currentRequestCount := requestCount
    requestCountMu.Unlock()
    
    if currentRequestCount%3 == 0 {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}))

var successCount int
var errorCount int
var resultCountMu sync.Mutex

for i := 0; i < concurrency; i++ {
    go func(id int) {
        result := handler.RetryWithBackoff(context.Background(), appErr, 0, makeRequest)
        
        resultCountMu.Lock()
        if result != nil {
            errorCount++
            errorsCh <- result
        } else {
            successCount++
            errorsCh <- nil
        }
        resultCountMu.Unlock()
    }(i)
}
```

#### 3. 添加必要的导入

```go
import (
    "sync"  // 添加 sync 包
    // ... 其他导入
)
```

### Verification

#### 测试结果
修复前后对比：

**修复前：**
```
FAIL: TestRealHeartbeatMissed - race detected
FAIL: TestRealConcurrentErrorHandling - race detected
```

**修复后：**
```
✅ make test - 全部通过
✅ make test-race - 全部通过，无竞态条件警告
✅ gofmt - 代码格式正确
```

#### 具体测试通过情况
- ✅ `TestRealHeartbeatMissed` - 通过，无数据竞争
- ✅ `TestRealConcurrentErrorHandling` - 通过，无数据竞争
- ✅ `TestRunAllRealScenarios` - 所有子测试通过
- ✅ 所有其他测试保持通过

#### 代码质量检查
- ✅ `go test -race` - 无数据竞争警告
- ✅ `gofmt` - 代码格式正确
- ✅ 所有测试覆盖率保持 96.1%

### Lessons Learned

1. **竞态条件检测**：`go test -race` 是检测数据竞争的必要工具
2. **共享变量保护**：所有被多个 goroutine 访问的变量都需要同步保护
3. **原子操作优先**：使用 `sync.Mutex` 而不是依赖隐式同步
4. **测试并发代码**：并发测试必须验证在竞态检测器下通过
5. **最小化临界区**：锁的持有时间应该尽可能短

### Best Practices

1. **使用 defer 释放锁**：确保锁一定会被释放
   ```go
   mu.Lock()
   defer mu.Unlock()
   ```

2. **读写分离**：对于频繁读取、很少写入的变量，考虑使用 `sync.RWMutex`

3. **避免嵌套锁**：嵌套锁容易导致死锁，应该避免

4. **Channel 通信**：对于简单的数据传递，考虑使用 channel 而不是共享变量

### References
- Go Data Race Detector: https://go.dev/doc/articles/race_detector
- Go Concurrency: https://go.dev/doc/effective_go#concurrency
- sync Package: https://pkg.go.dev/sync

---

## Bug #3: pgvector 向量检索返回空结果

### Date
2026-03-19

### Severity
High - 导致知识库检索功能完全失效

### Affected Files
- `internal/storage/postgres/repositories/knowledge_repository.go`
- `examples/knowledge-base/main.go`

### Bug Description

#### 症状
1. 执行向量检索时总是返回 0 个结果
2. 数据库中确实存在数据（14 条记录，embedding_status = 'completed'）
3. 日志显示查询成功执行，但扫描结果时全部失败

#### 错误日志
```
INFO Vector search query succeeded
WARN Failed to scan row row=1 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
WARN Failed to scan row row=2 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
WARN Failed to scan row row=3 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
WARN Failed to scan row row=4 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
WARN Failed to scan row row=5 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
INFO Vector search completed rows_scanned=5 chunks_returned=0
INFO Vector search succeeded results_count=0
```

### Root Cause Analysis

#### 问题：pgvector 二进制格式与 Go 类型不匹配

##### 错误代码
```go
// 查询语句
query := `
    SELECT id, tenant_id, content, embedding, embedding_model, embedding_version,
           embedding_status, source_type, source, metadata, document_id,
           chunk_index, content_hash, access_count, created_at, updated_at,
           1 - (embedding <=> $1::vector) as similarity
    FROM knowledge_chunks_1024
    WHERE tenant_id = $2
      AND embedding_status = 'completed'
    ORDER BY embedding <=> $1::vector
    LIMIT $3
`

// 扫描代码
err := rows.Scan(
    &chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,  // ← 直接扫描到 []float64
    &chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
    &chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
    &chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
    &chunk.CreatedAt, &chunk.UpdatedAt, &similarity,
)
```

##### 问题分析
1. **pgvector 驱动行为**：
   - pgvector PostgreSQL 驱动默认以二进制格式（`[]uint8`）返回向量数据
   - 这是 PostgreSQL binary protocol 的标准行为

2. **Go 代码期望**：
   - 代码期望直接扫描到 `[]float64` 类型
   - 类型不匹配导致扫描失败

3. **影响范围**：
   - 所有向量检索操作都失败
   - RAG 知识库、经验检索、工具检索全部失效
   - 整个检索系统不可用

4. **为什么之前没发现**：
   - 代码看起来逻辑正确
   - 数据库查询成功执行
   - 只有在扫描结果时才失败
   - 没有详细的错误日志，导致难以定位

### Solution

#### 1. 修改 SQL 查询，将向量列转换为文本格式

```go
query := `
    SELECT id, tenant_id, content, embedding::text, embedding_model, embedding_version,
           embedding_status, source_type, source, metadata::text, document_id,
           chunk_index, content_hash, access_count, created_at, updated_at,
           1 - (embedding <=> $1::vector) as similarity
    FROM knowledge_chunks_1024
    WHERE tenant_id = $2
      AND embedding_status = 'completed'
    ORDER BY embedding <=> $1::vector
    LIMIT $3
`
```

关键改动：
- `embedding::text` - 将向量列转换为文本格式
- `metadata::text` - 将 JSONB 列也转换为文本格式（预防性修改）

#### 2. 修改扫描逻辑，先扫描到字符串变量

```go
chunks := make([]*storage_models.KnowledgeChunk, 0)
rowCount := 0
for rows.Next() {
    rowCount++
    chunk := &storage_models.KnowledgeChunk{}
    var similarity float64
    var embeddingStr, metadataStr string  // ← 先扫描到字符串
    var documentID sql.NullString

    err := rows.Scan(
        &chunk.ID, &chunk.TenantID, &chunk.Content, &embeddingStr,
        &chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
        &chunk.SourceType, &chunk.Source, &metadataStr, &documentID,
        &chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
        &chunk.CreatedAt, &chunk.UpdatedAt, &similarity,
    )
    if err != nil {
        slog.Warn("Failed to scan row", "row", rowCount, "error", err)
        continue
    }

    // Parse embedding string to []float64
    chunk.Embedding, err = parseVectorString(embeddingStr)
    if err != nil {
        slog.Warn("Failed to parse embedding", "row", rowCount, "error", err)
        continue
    }

    // Parse metadata JSON string to map
    if metadataStr != "" {
        if err := json.Unmarshal([]byte(metadataStr), &chunk.Metadata); err != nil {
            slog.Warn("Failed to parse metadata", "row", rowCount, "error", err)
            chunk.Metadata = make(map[string]interface{})
        }
    }

    // Handle nullable document_id
    if documentID.Valid {
        chunk.DocumentID = documentID.String
    }

    // Store similarity in metadata for downstream processing
    if chunk.Metadata == nil {
        chunk.Metadata = make(map[string]interface{})
    }
    chunk.Metadata["similarity"] = similarity
    chunks = append(chunks, chunk)
}

slog.Info("Vector search completed", "rows_scanned", rowCount, "chunks_returned", len(chunks))
```

关键改动：
1. 添加字符串变量 `embeddingStr` 和 `metadataStr`
2. 先扫描到字符串变量
3. 使用 `parseVectorString` 函数解析向量字符串
4. 使用 `json.Unmarshal` 解析元数据 JSON
5. 添加详细的日志记录

#### 3. parseVectorString 函数（已存在，确保正确实现）

```go
func parseVectorString(vecStr string) ([]float64, error) {
    // pgvector stores vectors in text format like "[0.1,0.2,0.3,...]"
    if len(vecStr) == 0 {
        return []float64{}, nil
    }

    // Remove brackets and split by comma
    vecStr = strings.Trim(vecStr, "[]")
    if vecStr == "" {
        return []float64{}, nil
    }

    parts := strings.Split(vecStr, ",")
    result := make([]float64, len(parts))
    for i, part := range parts {
        val, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &result[i])
        if err != nil || val != 1 {
            return nil, fmt.Errorf("failed to parse vector component: %w", err)
        }
    }

    return result, nil
}
```

### Verification

#### 测试结果
修复前后对比：

**修复前：**
```
INFO Vector search query succeeded
WARN Failed to scan row row=1 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
INFO Vector search completed rows_scanned=5 chunks_returned=0
INFO Vector search succeeded results_count=0
```

**修复后：**
```
INFO Vector search query succeeded
INFO Vector search completed rows_scanned=5 chunks_returned=5
INFO Vector search succeeded results_count=5
```

#### 功能验证
- ✅ 向量检索成功返回结果
- ✅ 相似度分数正确计算
- ✅ 内容正确返回（包含"智能缓存"、"分层架构"等关键词）
- ✅ 所有知识库功能正常工作

#### 代码质量检查
- ✅ `go build` - 编译成功
- ✅ `go vet` - 无警告
- ✅ `gofmt` - 格式正确
- ✅ 详细日志记录，便于后续调试

### Lessons Learned

1. **PostgreSQL 二进制协议**：
   - PostgreSQL 驱动默认使用二进制协议返回数据
   - 复杂类型（如 pgvector）需要显式转换为文本格式
   - 这与 MySQL 等其他数据库的行为不同

2. **类型安全的重要性**：
   - Go 的类型系统会在运行时捕获类型不匹配
   - 但只有在扫描数据时才会暴露问题
   - 编译时无法检测到这类错误

3. **调试日志的价值**：
   - 详细的日志记录对于定位问题至关重要
   - 扫描失败的具体错误信息帮助快速定位问题
   - 建议在关键路径添加详细的 INFO/WARN 日志

4. **pgvector 特殊性**：
   - pgvector 是 PostgreSQL 的扩展，行为与标准类型不同
   - 需要特别注意向量类型的读写方式
   - 建议参考 pgvector 官方文档和示例代码

### Best Practices

1. **处理 PostgreSQL 扩展类型**：
   ```go
   // 好的做法：显式转换为文本
   SELECT embedding::text, metadata::text FROM table
   
   // 避免：直接扫描复杂类型
   SELECT embedding, metadata FROM table  // 可能导致类型不匹配
   ```

2. **添加类型转换的辅助函数**：
   ```go
   // 向量字符串解析
   func parseVectorString(vecStr string) ([]float64, error)
   
   // 向量格式化
   func FormatVector(vec []float64) string
   ```

3. **防御性编程**：
   ```go
   // 检查扫描错误
   if err := rows.Scan(...); err != nil {
       log.Warn("Failed to scan row", "error", err)
       continue  // 跳过错误行，不中断整个查询
   }
   ```

4. **详细的错误日志**：
   ```go
   slog.Warn("Failed to scan row", 
       "row", rowCount, 
       "error", err)
   ```

### References
- pgvector Documentation: https://github.com/pgvector/pgvector
- PostgreSQL Binary Protocol: https://www.postgresql.org/docs/current/protocol.html
- Go SQL Scanner Interface: https://pkg.go.dev/database/sql#Scanner
- PostgreSQL Type Casting: https://www.postgresql.org/docs/current/sql-createcast.html