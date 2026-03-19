# Bug Log

## Bug #1: Executor runSteps Function

### Date
2026-03-16

### Severity
High - Causes workflow execution timeout and deadlock

### Affected Files
- `internal/workflow/engine/executor.go`
- `internal/workflow/engine/executor_test.go`

### Bug Description

#### Symptoms
1. `TestExecutorCoverage/execute_workflow_with_dependencies` test timeout (30 seconds)
2. `TestExecutorCoverage/execute_workflow_with_agent_error` test failure
3. `TestExecutorCoverage/execute_workflow_with_invalid_agent_type` test failure
4. `TestExecutorHelperFunctionsCoverage/execute_step_with_timeout` test failure

#### Error Messages
```
panic: test timed out after 30s
running tests:
    TestExecutorCoverage (30s)
    TestExecutorCoverage/execute_workflow_with_dependencies (30s)
```

### Root Cause Analysis

#### 1. runSteps Function Concurrency Control Logic Defect

##### Issue 1: stepChan writes but never reads
```go
// Incorrect code
stepChan <- stepID
// Never reads from stepChan
```

This causes:
- When `len(stepChan) >= e.maxParallel`, cannot submit new tasks
- Channel fills up and the entire workflow execution hangs

##### Issue 2: Both Execute and runSteps read from resultChan
```go
// In Execute function
case result := <-resultChan:

// In runSteps function
case result := <-resultChan:
```

This causes:
- Two goroutines competing for the same channel
- Results may be received by the wrong consumer
- Main loop may never receive results

##### Issue 3: Failed steps don't return errors properly
```go
// Incorrect code
if result.Status == StepStatusFailed {
    execution.Status = WorkflowStatusFailed
    execution.Error = result.Error
    // No error returned to caller
}
```

This causes:
- After step failure, workflow continues execution
- Test cases cannot properly detect failures

#### 2. Test Case Issues

##### Issue 1: Missing Timeout field
```go
// Incorrect test code
{
    ID:        "step1",
    Name:      "First Step",
    AgentType: "test-agent",
    Input:     "step1 input",
    // Missing Timeout field
}
```

This causes:
- `Timeout` is 0
- `context.WithTimeout(ctx, 0)` cancels context immediately
- Agent cannot execute normally

##### Issue 2: Timeout test is incorrect
```go
// Incorrect test code
return NewMockAgent("test", "slow-agent", func(ctx context.Context, input any) (any, error) {
    time.Sleep(200 * time.Millisecond)
    // Doesn't check if context is canceled
})
```

This causes:
- Agent doesn't respond to context cancellation
- Timeout cannot be properly detected

### Solution

#### 1. Refactor runSteps Function

Use `sync.WaitGroup` to replace complex channel mechanism:

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

Key improvements:
1. Use `sync.WaitGroup` to manage goroutine lifecycle
2. Introduce `processed` map to track all processed steps
3. Correctly read from `stepChan` to release capacity
4. Simplify event-driven logic, remove `wakeup` channel

#### 2. Fix Execute Function

Return error immediately when receiving failed step:

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

#### 3. Fix Test Cases

##### Add Timeout field to all steps
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
            Timeout:   10 * time.Second, // Add Timeout
        },
        {
            ID:        "step2",
            Name:      "Second Step",
            AgentType: "test-agent",
            DependsOn: []string{"step1"},
            Timeout:   10 * time.Second, // Add Timeout
        },
        {
            ID:        "step3",
            Name:      "Third Step",
            AgentType: "test-agent",
            DependsOn: []string{"step1", "step2"},
            Timeout:   10 * time.Second, // Add Timeout
        },
    },
}
```

##### Fix timeout test
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
            return nil, ctx.Err() // Correctly respond to context cancellation
        }
    }), nil
})
```

### Verification

#### Test Results
All tests pass:
- ✅ `TestExecutorCoverage` - 6/6 subtests pass
- ✅ `TestExecutorHelperFunctionsCoverage` - 5/5 subtests pass
- ✅ `TestRetryLogicCoverage` - 3/3 subtests pass
- ✅ `TestWorkflowExecutionStateCoverage` - 1/1 subtests pass
- ✅ `TestDAGCoverage` - 9/9 subtests pass
- ✅ `TestAgentRegistryCoverage` - 7/7 subtests pass
- ✅ `TestOutputStoreCoverage` - 5/5 subtests pass
- ✅ `TestErrorDefinitionsCoverage` - 1/1 subtests pass
- ✅ `TestWorkflowStatusConstantsCoverage` - 2/2 subtests pass
- ✅ `TestStepStatusConstantsCoverage` - 1/1 subtests pass
- ✅ `TestWorkflowTypesCoverage` - 10/10 subtests pass

#### Code Quality Checks
- ✅ `go vet` - No warnings
- ✅ `gofmt` - Formatting correct
- ✅ `goimports` - Imports correct

---

## Bug #2: Data Race Conditions in Tests

### Date
2026-03-16

### Severity
High - Data races cause test failures in `go test -race` mode

### Affected Files
- `internal/core/errors/error_scenarios_test.go`

### Bug Description

#### Symptoms
Multiple data races detected when executing `make test-race`:

1. `TestRealHeartbeatMissed` - Data race
2. `TestRealConcurrentErrorHandling` - Multiple data races

#### Error Messages
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

#### 1. TestRealHeartbeatMissed - heartbeatStopped variable race

##### Problem Code
```go
var heartbeatStopped bool

// Goroutine 1: Write
go func() {
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    time.Sleep(50 * time.Millisecond)
    heartbeatCh <- true
    heartbeatStopped = true  // ← Write operation
}()

// Goroutine 2: Read
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
            if heartbeatStopped {  // ← Read operation
                missedCount++
                if missedCount >= 2 {
                    return fmt.Errorf("heartbeat missed for %d cycles", missedCount)
                }
            }
        }
    }
}
```

##### Race Cause
- Multiple goroutines access `heartbeatStopped` variable simultaneously
- One goroutine writes, another reads
- No synchronization mechanism protecting shared variable

#### 2. TestRealConcurrentErrorHandling - Multiple variable races

##### Problem Code
```go
var requestCount int
var successCount int
var errorCount int

// HTTP handler: Read and write requestCount
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    requestCount++  // ← Write operation, no protection
    if requestCount%3 == 0 {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}))

// Worker goroutines: Read and write successCount/errorCount
for i := 0; i < concurrency; i++ {
    go func(id int) {
        result := handler.RetryWithBackoff(context.Background(), appErr, 0, makeRequest)
        
        if result != nil {
            errorCount++  // ← Write operation, no protection
            errorsCh <- result
        } else {
            successCount++  // ← Write operation, no protection
            errorsCh <- nil
        }
    }(i)
}
```

##### Race Cause
- `requestCount`: Multiple HTTP requests modify simultaneously
- `successCount`: Multiple worker goroutines modify simultaneously
- `errorCount`: Multiple worker goroutines modify simultaneously
- No mutex protecting shared variables

### Solution

#### 1. Fix TestRealHeartbeatMissed

Add mutex to protect `heartbeatStopped` variable:

```go
var heartbeatStopped bool
var heartbeatStoppedMu sync.Mutex

// Lock when writing
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

// Lock when reading
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

#### 2. Fix TestRealConcurrentErrorHandling

Add mutex to protect all shared variables:

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

#### 3. Add necessary imports

```go
import (
    "sync"  // Add sync package
    // ... other imports
)
```

### Verification

#### Test Results
Before and after comparison:

**Before:**
```
FAIL: TestRealHeartbeatMissed - race detected
FAIL: TestRealConcurrentErrorHandling - race detected
```

**After:**
```
✅ make test - All pass
✅ make test-race - All pass, no race condition warnings
✅ gofmt - Code formatting correct
```

#### Specific test results
- ✅ `TestRealHeartbeatMissed` - Passes, no data race
- ✅ `TestRealConcurrentErrorHandling` - Passes, no data race
- ✅ `TestRunAllRealScenarios` - All subtests pass
- ✅ All other tests remain passing

#### Code quality checks
- ✅ `go test -race` - No data race warnings
- ✅ `gofmt` - Code formatting correct
- ✅ All test coverage remains at 96.1%

### Lessons Learned

1. **Race condition detection**: `go test -race` is a necessary tool for detecting data races
2. **Shared variable protection**: All variables accessed by multiple goroutines need synchronization protection
3. **Atomic operations first**: Use `sync.Mutex` instead of relying on implicit synchronization
4. **Test concurrent code**: Concurrent tests must verify they pass under the race detector
5. **Minimize critical sections**: Lock holding time should be as short as possible

### Best Practices

1. **Use defer to release lock**: Ensure lock is always released
   ```go
   mu.Lock()
   defer mu.Unlock()
   ```

2. **Read-write separation**: For variables with frequent reads and rare writes, consider using `sync.RWMutex`

3. **Avoid nested locks**: Nested locks easily cause deadlocks and should be avoided

4. **Channel communication**: For simple data passing, consider using channels instead of shared variables

### References
- Go Data Race Detector: https://go.dev/doc/articles/race_detector
- Go Concurrency: https://go.dev/doc/effective_go#concurrency
- sync Package: https://pkg.go.dev/sync

---

## Bug #3: pgvector Vector Search Returns Empty Results

### Date
2026-03-19

### Severity
High - Causes complete failure of knowledge base retrieval functionality

### Affected Files
- `internal/storage/postgres/repositories/knowledge_repository.go`
- `examples/knowledge-base/main.go`

### Bug Description

#### Symptoms
1. Vector search always returns 0 results
2. Data exists in database (14 records, embedding_status = 'completed')
3. Logs show query executes successfully but all scan results fail

#### Error Logs
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

#### Issue: pgvector binary format mismatch with Go types

##### Incorrect Code
```go
// Query statement
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

// Scan code
err := rows.Scan(
    &chunk.ID, &chunk.TenantID, &chunk.Content, &chunk.Embedding,  // ← Direct scan to []float64
    &chunk.EmbeddingModel, &chunk.EmbeddingVersion, &chunk.EmbeddingStatus,
    &chunk.SourceType, &chunk.Source, &chunk.Metadata, &chunk.DocumentID,
    &chunk.ChunkIndex, &chunk.ContentHash, &chunk.AccessCount,
    &chunk.CreatedAt, &chunk.UpdatedAt, &similarity,
)
```

##### Issue Analysis
1. **pgvector driver behavior**:
   - pgvector PostgreSQL driver returns vector data in binary format (`[]uint8`) by default
   - This is standard behavior of PostgreSQL binary protocol

2. **Go code expectation**:
   - Code expects direct scan to `[]float64` type
   - Type mismatch causes scan failure

3. **Impact scope**:
   - All vector search operations fail
   - RAG knowledge base, experience retrieval, tool retrieval all fail
   - Entire retrieval system is unusable

4. **Why it wasn't discovered before**:
   - Code looks logically correct
   - Database query executes successfully
   - Failure only occurs when scanning results
   - Lack of detailed error logs made it hard to locate

### Solution

#### 1. Modify SQL query, convert vector column to text format

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

Key changes:
- `embedding::text` - Convert vector column to text format
- `metadata::text` - Also convert JSONB column to text format (preventive modification)

#### 2. Modify scan logic, scan to string variables first

```go
chunks := make([]*storage_models.KnowledgeChunk, 0)
rowCount := 0
for rows.Next() {
    rowCount++
    chunk := &storage_models.KnowledgeChunk{}
    var similarity float64
    var embeddingStr, metadataStr string  // ← Scan to strings first
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

Key changes:
1. Add string variables `embeddingStr` and `metadataStr`
2. First scan to string variables
3. Use `parseVectorString` function to parse vector string
4. Use `json.Unmarshal` to parse metadata JSON
5. Add detailed logging

#### 3. parseVectorString function (ensure correct implementation)

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

#### Test Results
Before and after comparison:

**Before:**
```
INFO Vector search query succeeded
WARN Failed to scan row row=1 error="sql: Scan error on column index 3, name \"embedding\": unsupported Scan, storing driver.Value type []uint8 into type *[]float64"
INFO Vector search completed rows_scanned=5 chunks_returned=0
INFO Vector search succeeded results_count=0
```

**After:**
```
INFO Vector search query succeeded
INFO Vector search completed rows_scanned=5 chunks_returned=5
INFO Vector search succeeded results_count=5
```

#### Functional verification
- ✅ Vector search successfully returns results
- ✅ Similarity scores correctly calculated
- ✅ Content correctly returned (contains "智能缓存", "分层架构" and other keywords)
- ✅ All knowledge base functions work normally

#### Code quality checks
- ✅ `go build` - Compilation successful
- ✅ `go vet` - No warnings
- ✅ `gofmt` - Formatting correct
- ✅ Detailed logging for easier future debugging

### Lessons Learned

1. **PostgreSQL binary protocol**:
   - PostgreSQL drivers use binary protocol by default to return data
   - Complex types (like pgvector) need explicit text conversion
   - This differs from behavior of other databases like MySQL

2. **Importance of type safety**:
   - Go's type system catches type mismatches at runtime
   - But issues only appear when scanning data
   - Cannot detect these errors at compile time

3. **Value of debugging logs**:
   - Detailed logging is crucial for locating problems
   - Specific error messages from scan failures help quickly locate issues
   - Recommend adding detailed INFO/WARN logs on critical paths

4. **pgvector specifics**:
   - pgvector is a PostgreSQL extension with different behavior than standard types
   - Need special attention to read/write methods for vector types
   - Recommend referencing pgvector official documentation and example code

### Best Practices

1. **Handle PostgreSQL extension types**:
   ```go
   // Good practice: explicitly convert to text
   SELECT embedding::text, metadata::text FROM table
   
   // Avoid: directly scan complex types
   SELECT embedding, metadata FROM table  // May cause type mismatch
   ```

2. **Add type conversion helper functions**:
   ```go
   // Vector string parsing
   func parseVectorString(vecStr string) ([]float64, error)
   
   // Vector formatting
   func FormatVector(vec []float64) string
   ```

3. **Defensive programming**:
   ```go
   // Check scan errors
   if err := rows.Scan(...); err != nil {
       log.Warn("Failed to scan row", "error", err)
       continue  // Skip error row, don't interrupt entire query
   }
   ```

4. **Detailed error logging**:
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