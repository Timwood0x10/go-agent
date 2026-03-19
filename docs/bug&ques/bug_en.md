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

---

## Bug #4: ExperienceRepository Multiple Field Handling Errors Causing Test Failures

### Date
2026-03-19

### Severity
High - Causes all ExperienceRepository tests to fail, affecting experience retrieval functionality

### Affected Files
- `internal/storage/postgres/repositories/experience_repository.go`
- `internal/storage/postgres/repositories/experience_repository_test.go`

### Bug Description

#### Symptoms
1. `TestExperienceRepository_Create` test passes, but all other tests involving metadata fail
2. `TestExperienceRepository_UpdateScore` and `TestExperienceRepository_UpdateEmbedding` tests fail with `updated_at` column not found error
3. `TestExperienceRepository_SearchByVector` test fails with vector format error
4. `TestExperienceRepository_ListByType` and `TestExperienceRepository_ListByAgent` tests fail, returning 0 results
5. `TestExperienceRepository_CleanupExpired` test fails due to timezone inconsistency

#### Error Messages
```
# metadata field error
Error: "sql: Scan error on column index 11, name \"metadata\": unsupported Scan, storing driver.Value type []uint8 into type *map[string]interface {}"

# updated_at column error
Error: "pq: column \"updated_at\" of relation \"experiences_1024\" does not exist"

# vector format error
Error: "pq: invalid input syntax for type vector: \"{0,0.0009765625,...}\""

# query returns 0 results
Error: "\"0\" is not greater than or equal to \"1\""
```

### Root Cause Analysis

#### Issue 1: metadata field not converted to text format

##### Incorrect Code
```go
// GetByID method
query := `
    SELECT id, tenant_id, type, input, output, embedding, embedding_model, embedding_version,
           score, success, agent_id, metadata, decay_at, created_at  // ← metadata not converted
    FROM experiences_1024
    WHERE id = $1
`

err := r.db.QueryRowContext(ctx, query, id).Scan(
    &exp.ID, &exp.TenantID, &exp.Type, &exp.Input, &exp.Output,
    &exp.Embedding, &exp.EmbeddingModel, &exp.EmbeddingVersion,
    &exp.Score, &exp.Success, &exp.AgentID, &exp.Metadata,  // ← Direct scan to map[string]interface{}
    &exp.DecayAt, &exp.CreatedAt,
)
```

##### Issue Analysis
- PostgreSQL JSONB type returns data in binary format by default
- Go code expects direct scan to `map[string]interface{}` type
- Type mismatch causes scan failure
- Affects all query methods involving metadata

#### Issue 2: embedding field not converted to text format

##### Incorrect Code
```go
// SearchByVector method
query := `
    SELECT id, tenant_id, type, input, output, embedding, embedding_model, embedding_version,
           score, success, agent_id, metadata, decay_at, created_at,
           1 - (embedding <=> $1) as similarity  // ← embedding not converted
    FROM experiences_1024
    WHERE tenant_id = $2
      AND (decay_at IS NULL OR decay_at > NOW())
    ORDER BY embedding <=> $1
    LIMIT $3
`

err := rows.Scan(
    &exp.ID, &exp.TenantID, &exp.Type, &exp.Input, &exp.Output,
    &exp.Embedding, &exp.EmbeddingModel, &exp.EmbeddingVersion,  // ← Direct scan to []float64
    &exp.Score, &exp.Success, &exp.AgentID, &exp.Metadata,
    &exp.DecayAt, &exp.CreatedAt, &similarity,
)
```

##### Issue Analysis
- pgvector type returns data in binary format by default
- Go code expects direct scan to `[]float64` type
- Type mismatch causes scan failure
- Affects all query methods involving embedding

#### Issue 3: UpdateScore and UpdateEmbedding methods attempt to update non-existent columns

##### Incorrect Code
```go
// UpdateScore method
query := `
    UPDATE experiences_1024
    SET score = $2, updated_at = NOW()  // ← updated_at column doesn't exist
    WHERE id = $1
`

// UpdateEmbedding method
query := `
    UPDATE experiences_1024
    SET embedding = $2, embedding_model = $3, embedding_version = $4, updated_at = NOW()  // ← updated_at column doesn't exist
    WHERE id = $1
`
```

##### Issue Analysis
- `experiences_1024` table only has `created_at` column, no `updated_at` column
- Code attempts to update non-existent column causing SQL error
- These two methods cannot execute at all

#### Issue 4: Create method handles zero-value DecayAt incorrectly

##### Incorrect Code
```go
// Create method
query := `
    INSERT INTO experiences_1024
    (tenant_id, type, input, output, embedding, embedding_model, embedding_version,
     score, success, agent_id, metadata, decay_at, created_at)
    VALUES ($1, $2, $3, $4, $5::vector, $6, $7, $8, $9, $10, $11, $12, $13)  // ← Always passes decay_at
    RETURNING id
`

err = r.db.QueryRowContext(ctx, query,
    exp.TenantID, exp.Type, exp.Input, exp.Output, embeddingStr,
    exp.EmbeddingModel, exp.EmbeddingVersion,
    exp.Score, exp.Success, exp.AgentID, metadataJSON,
    exp.DecayAt, exp.CreatedAt,  // ← Even when DecayAt is zero value
).Scan(&id)
```

##### Issue Analysis
- When `DecayAt` is zero value, it gets stored as `0001-01-01 00:00:00`
- Query condition `decay_at > NOW()` filters out these records
- Causes test-created data to be unqueryable
- `ListByType`, `ListByAgent` and other methods return empty results

#### Issue 5: SearchByVector method vector format error

##### Incorrect Code
```go
// SearchByVector method
rows, err := r.db.QueryContext(ctx, query, embedding, tenantID, limit)  // ← Directly pass []float64
```

##### Issue Analysis
- pgvector expects vector format as string `[0.1,0.2,0.3]`
- Go's slice format `{0.1,0.2,0.3}` cannot be parsed by pgvector
- Causes SQL syntax error

#### Issue 6: CleanupExpired test timezone inconsistency

##### Problem Code
```go
// Test code
expiredExp := &storage_models.Experience{
    DecayAt: time.Now().Add(-1 * time.Hour),  // ← Uses local time
}
```

##### Issue Analysis
- Test code uses local time (CST +0800)
- Database uses UTC time
- Timezone conversion causes incorrect time comparison
- Expired experience is considered not expired

### Solution

#### 1. Fix all query methods, add ::text conversion

##### GetByID method
```go
query := `
    SELECT id, tenant_id, type, input, output, embedding::text, embedding_model, embedding_version,
           score, success, agent_id, metadata::text, decay_at, created_at
    FROM experiences_1024
    WHERE id = $1
`

exp := &storage_models.Experience{}
var embeddingStr, metadataStr string
err := r.db.QueryRowContext(ctx, query, id).Scan(
    &exp.ID, &exp.TenantID, &exp.Type, &exp.Input, &exp.Output,
    &embeddingStr, &exp.EmbeddingModel, &exp.EmbeddingVersion,
    &exp.Score, &exp.Success, &exp.AgentID, &metadataStr,
    &exp.DecayAt, &exp.CreatedAt,
)

// Parse embedding string to float64 array
exp.Embedding, err = parseVectorString(embeddingStr)
if err != nil {
    return nil, fmt.Errorf("parse embedding: %w", err)
}

// Parse metadata JSON string to map
if metadataStr != "" {
    if err := json.Unmarshal([]byte(metadataStr), &exp.Metadata); err != nil {
        return nil, fmt.Errorf("parse metadata: %w", err)
    }
}
```

##### SearchByVector method
```go
// Convert embedding to pgvector format
embeddingStr := float64ToVectorString(embedding)

query := `
    SELECT id, tenant_id, type, input, output, embedding::text, embedding_model, embedding_version,
           score, success, agent_id, metadata::text, decay_at, created_at,
           1 - (embedding <=> $1::vector) as similarity
    FROM experiences_1024
    WHERE tenant_id = $2
      AND (decay_at IS NULL OR decay_at > NOW())
    ORDER BY embedding <=> $1::vector
    LIMIT $3
`

rows, err := r.db.QueryContext(ctx, query, embeddingStr, tenantID, limit)

// Parse in scan loop
for rows.Next() {
    exp := &storage_models.Experience{}
    var similarity float64
    var embeddingStr, metadataStr string
    
    err := rows.Scan(
        &exp.ID, &exp.TenantID, &exp.Type, &exp.Input, &exp.Output,
        &embeddingStr, &exp.EmbeddingModel, &exp.EmbeddingVersion,
        &exp.Score, &exp.Success, &exp.AgentID, &metadataStr,
        &exp.DecayAt, &exp.CreatedAt, &similarity,
    )
    
    // Parse embedding and metadata
    exp.Embedding, err = parseVectorString(embeddingStr)
    if metadataStr != "" {
        json.Unmarshal([]byte(metadataStr), &exp.Metadata)
    }
}
```

##### ListByType and ListByAgent methods
Similarly add `::text` conversion and parsing logic.

#### 2. Fix UpdateScore and UpdateEmbedding methods

##### UpdateScore method
```go
query := `
    UPDATE experiences_1024
    SET score = $2  // ← Remove updated_at
    WHERE id = $1
`
```

##### UpdateEmbedding method
```go
// Convert embedding to pgvector format
embeddingStr := float64ToVectorString(embedding)

query := `
    UPDATE experiences_1024
    SET embedding = $2::vector, embedding_model = $3, embedding_version = $4  // ← Remove updated_at
    WHERE id = $1
`

result, err := r.db.ExecContext(ctx, query, id, embeddingStr, model, version)
```

#### 3. Fix Create method, handle zero-value DecayAt

```go
func (r *ExperienceRepository) Create(ctx context.Context, exp *storage_models.Experience) error {
    // Convert metadata to JSON for database storage
    metadataJSON, err := json.Marshal(exp.Metadata)
    if err != nil {
        return fmt.Errorf("marshal metadata: %w", err)
    }

    // Convert embedding to pgvector format
    embeddingStr := float64ToVectorString(exp.Embedding)

    // Build query with optional decay_at
    var query string
    var args []interface{}

    if exp.DecayAt.IsZero() {
        // Don't set decay_at, let database use default value
        query = `
            INSERT INTO experiences_1024
            (tenant_id, type, input, output, embedding, embedding_model, embedding_version,
             score, success, agent_id, metadata, created_at)
            VALUES ($1, $2, $3, $4, $5::vector, $6, $7, $8, $9, $10, $11, $12)
            RETURNING id
        `
        args = []interface{}{
            exp.TenantID, exp.Type, exp.Input, exp.Output, embeddingStr,
            exp.EmbeddingModel, exp.EmbeddingVersion,
            exp.Score, exp.Success, exp.AgentID, metadataJSON,
            exp.CreatedAt,
        }
    } else {
        // Set decay_at explicitly
        query = `
            INSERT INTO experiences_1024
            (tenant_id, type, input, output, embedding, embedding_model, embedding_version,
             score, success, agent_id, metadata, decay_at, created_at)
            VALUES ($1, $2, $3, $4, $5::vector, $6, $7, $8, $9, $10, $11, $12, $13)
            RETURNING id
        `
        args = []interface{}{
            exp.TenantID, exp.Type, exp.Input, exp.Output, embeddingStr,
            exp.EmbeddingModel, exp.EmbeddingVersion,
            exp.Score, exp.Success, exp.AgentID, metadataJSON,
            exp.DecayAt, exp.CreatedAt,
        }
    }

    var id string
    err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)

    if err != nil {
        return fmt.Errorf("create experience: %w", err)
    }

    exp.ID = id
    return nil
}
```

#### 4. Fix CleanupExpired test, use UTC time

```go
// Create an expired experience
expiredExp := &storage_models.Experience{
    TenantID:         "tenant-1",
    Type:             storage_models.ExperienceTypeQuery,
    Input:            "test input",
    Embedding:        createTestEmbedding(),
    EmbeddingModel:   "e5-large",
    EmbeddingVersion: 1,
    DecayAt:          time.Now().UTC().Add(-1 * time.Hour), // ← Use UTC time
    CreatedAt:        time.Now().UTC(),
}

// Create a non-expired experience
validExp := &storage_models.Experience{
    TenantID:         "tenant-1",
    Type:             storage_models.ExperienceTypeQuery,
    Input:            "test input",
    Embedding:        createTestEmbedding(),
    EmbeddingModel:   "e5-large",
    EmbeddingVersion: 1,
    DecayAt:          time.Now().UTC().Add(30 * 24 * time.Hour), // ← Use UTC time
    CreatedAt:        time.Now().UTC(),
}
```

### Verification

#### Test Results
Before and after comparison:

**Before:**
```
--- FAIL: TestExperienceRepository_UpdateScore (0.01s)
--- FAIL: TestExperienceRepository_UpdateEmbedding (0.01s)
--- FAIL: TestExperienceRepository_ListByType (0.01s)
--- FAIL: TestExperienceRepository_ListByAgent (0.01s)
--- FAIL: TestExperienceRepository_CleanupExpired (0.01s)
```

**After:**
```
✅ TestExperienceRepository_Create - PASS
✅ TestExperienceRepository_GetByID - PASS
✅ TestExperienceRepository_GetByID_NotFound - PASS
✅ TestExperienceRepository_Update - PASS
✅ TestExperienceRepository_Delete - PASS
✅ TestExperienceRepository_SearchByVector - PASS
✅ TestExperienceRepository_ListByType - PASS
✅ TestExperienceRepository_UpdateScore - PASS
✅ TestExperienceRepository_ListByAgent - PASS
✅ TestExperienceRepository_UpdateEmbedding - PASS
✅ TestExperienceRepository_CleanupExpired - PASS
✅ TestExperienceRepository_GetStatistics - PASS
✅ TestExperienceRepository_ConcurrentOperations - PASS
✅ TestExperienceRepository_AllTypes - PASS
✅ TestExperienceRepository_ContextCancelled - PASS
```

#### Functional verification
- ✅ Experience creation and query work normally
- ✅ Vector similarity search returns correct results
- ✅ List by type and agent ID queries work normally
- ✅ Expired experience cleanup works correctly
- ✅ Statistics query works correctly
- ✅ Concurrent operations handled correctly

#### Code quality checks
- ✅ `go build` - Compilation successful
- ✅ `go vet` - No warnings
- ✅ `gofmt` - Formatting correct
- ✅ All tests pass

### Lessons Learned

1. **Consistent handling of PostgreSQL extension types**:
   - All queries involving pgvector and JSONB need unified handling
   - Should check type conversion consistency during code review
   - Recommend creating unified helper functions to handle these types

2. **Impact of database schema changes**:
   - Need to check all related SQL queries when adding or removing columns
   - Should use database migration tools to manage schema changes
   - Recommend documenting table structure in documentation

3. **Best practices for time handling**:
   - Database applications should consistently use UTC time
   - Test code should also use UTC time to ensure consistency
   - Only perform timezone conversion at the user interface layer

4. **Defensive programming for zero-value handling**:
   - Should explicitly handle zero-value cases for optional fields
   - Can use database default values instead of explicitly passing zero values
   - Recommend adding validation logic at the model layer

### Best Practices

1. **Unified type conversion helper functions**:
   ```go
   // Vector conversion
   func float64ToVectorString(vec []float64) string
   func parseVectorString(vecStr string) ([]float64, error)
   
   // JSON conversion
   func marshalMetadata(metadata map[string]interface{}) ([]byte, error)
   func unmarshalMetadata(data []byte) (map[string]interface{}, error)
   ```

2. **Defensive programming for database queries**:
   ```go
   // Check scan errors
   if err := rows.Scan(...); err != nil {
       log.Warn("Failed to scan row", "error", err)
       continue  // Skip error row
   }
   
   // Handle empty values
   if metadataStr == "" {
       exp.Metadata = make(map[string]interface{})
   }
   ```

3. **Consistency in time handling**:
   ```go
   // Always use UTC time
   createdAt := time.Now().UTC()
   decayAt := time.Now().UTC().Add(30 * 24 * time.Hour)
   ```

4. **Conditional handling of optional fields**:
   ```go
   // Conditionally build SQL query
   if exp.DecayAt.IsZero() {
       // Use database default value
   } else {
       // Explicitly set value
   }
   ```

### References
- pgvector Type Casting: https://github.com/pgvector/pgvector#usage
- PostgreSQL JSONB: https://www.postgresql.org/docs/current/datatype-json.html
- Go Time Handling: https://go.dev/doc/effective_go#time
- PostgreSQL Default Values: https://www.postgresql.org/docs/current/ddl-default.html