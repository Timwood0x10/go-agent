package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goagent/internal/core/models"
)

// Executor executes workflows based on DAG ordering.
type Executor struct {
	registry    *AgentRegistry
	outputStore *OutputStore
	maxParallel int
	stepTimeout time.Duration
}

// NewExecutor creates a new Executor.
func NewExecutor(registry *AgentRegistry, outputStore *OutputStore) *Executor {
	return &Executor{
		registry:    registry,
		outputStore: outputStore,
		maxParallel: DefaultMaxParallel,
		stepTimeout: 5 * time.Minute,
	}
}

// Execute executes a workflow.
func (e *Executor) Execute(ctx context.Context, workflow *Workflow, initialInput string) (*WorkflowResult, error) {
	dag, err := NewDAG(workflow.Steps)
	if err != nil {
		return nil, fmt.Errorf("create DAG: %w", err)
	}

	executionOrder, err := dag.GetExecutionOrder()
	if err != nil {
		return nil, fmt.Errorf("get execution order: %w", err)
	}

	execution := &WorkflowExecution{
		ID:         generateExecutionID(),
		WorkflowID: workflow.ID,
		Status:     WorkflowStatusRunning,
		StepStates: make(map[string]*StepState),
		Variables:  make(map[string]interface{}),
		Context:    &models.TaskContext{},
		StartedAt:  time.Now(),
	}

	for k, v := range workflow.Variables {
		execution.Variables[k] = v
	}

	e.outputStore.Clear()

	stepChan := make(chan string, e.maxParallel)
	resultChan := make(chan *StepResult, len(workflow.Steps))
	errChan := make(chan error, 1)

	go e.runSteps(ctx, execution, workflow, executionOrder, initialInput, stepChan, resultChan, errChan)

	var stepResults []*StepResult
	for i := 0; i < len(workflow.Steps); i++ {
		select {
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
		case err := <-errChan:
			execution.Status = WorkflowStatusFailed
			execution.FinishedAt = time.Now()
			return &WorkflowResult{
				ExecutionID: execution.ID,
				WorkflowID:  workflow.ID,
				Status:      WorkflowStatusFailed,
				Error:       err.Error(),
				Duration:    execution.FinishedAt.Sub(execution.StartedAt),
			}, err
		case <-ctx.Done():
			execution.Status = WorkflowStatusCancelled
			execution.FinishedAt = time.Now()
			return nil, ctx.Err()
		}
	}

	execution.Status = WorkflowStatusCompleted
	execution.FinishedAt = time.Now()

	output := make(map[string]interface{})
	for _, result := range stepResults {
		output[result.StepID] = result.Output
	}

	return &WorkflowResult{
		ExecutionID: execution.ID,
		WorkflowID:  workflow.ID,
		Status:      execution.Status,
		Output:      output,
		Duration:    execution.FinishedAt.Sub(execution.StartedAt),
		Steps:       stepResults,
	}, nil
}

// runSteps runs workflow steps in parallel where possible.
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
			// Cannot execute yet, check if this step was already processed
			mu.Lock()
			alreadyProcessed := processed[stepID]
			mu.Unlock()

			if alreadyProcessed {
				// Already processed but not completed, move to next step
				stepIndex++
				continue
			}

			// Wait for dependency to complete
			// Check if we have any active goroutines
			wg.Wait()
			continue
		}

		// Wait for capacity
		if len(stepChan) >= e.maxParallel {
			<-stepChan
		}

		// Start executing the step
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

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if workflow is complete
	mu.Lock()
	allCompleted := len(completed) == len(workflow.Steps)
	mu.Unlock()

	if allCompleted {
		close(resultChan)
		return
	}

	// Check for incomplete workflow
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

// canExecute checks if a step can be executed.
func (e *Executor) canExecute(step *Step, completed map[string]bool, mu *sync.Mutex) bool {
	mu.Lock()
	defer mu.Unlock()
	for _, dep := range step.DependsOn {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// findStep finds a step by ID.
func (e *Executor) findStep(steps []*Step, stepID string) *Step {
	for _, step := range steps {
		if step.ID == stepID {
			return step
		}
	}
	return nil
}

// executeStep executes a single step.
func (e *Executor) executeStep(
	ctx context.Context,
	workflow *Workflow,
	stepID string,
	initialInput string,
	completed map[string]bool,
) *StepResult {
	step := e.findStep(workflow.Steps, stepID)
	if step == nil {
		return &StepResult{
			StepID: stepID,
			Status: StepStatusFailed,
			Error:  "step not found",
		}
	}

	startTime := time.Now()

	input := e.resolveInput(step, initialInput, completed)

	output, err := e.executeWithRetry(ctx, step, input)

	result := &StepResult{
		StepID:   stepID,
		Name:     step.Name,
		Status:   StepStatusCompleted,
		Output:   output,
		Duration: time.Now().Sub(startTime),
	}

	if err != nil {
		result.Status = StepStatusFailed
		result.Error = err.Error()
	}

	e.outputStore.Set(stepID, &StepOutput{
		StepID:    stepID,
		Output:    output,
		Variables: make(map[string]interface{}),
	})

	return result
}

// resolveInput resolves the input for a step.
func (e *Executor) resolveInput(step *Step, initialInput string, completed map[string]bool) string {
	if len(step.DependsOn) == 0 {
		return step.Input
	}

	if step.Input != "" {
		return step.Input
	}

	var depsOutput string
	for _, dep := range step.DependsOn {
		if output, exists := e.outputStore.Get(dep); exists {
			depsOutput = output.Output
			break
		}
	}

	if depsOutput != "" {
		return depsOutput
	}

	return initialInput
}

// executeWithRetry executes a step with retry logic.
func (e *Executor) executeWithRetry(ctx context.Context, step *Step, input string) (string, error) {
	maxAttempts := 1
	initialDelay := time.Second

	if step.RetryPolicy != nil {
		maxAttempts = step.RetryPolicy.MaxAttempts
		initialDelay = step.RetryPolicy.InitialDelay
	}

	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		output, err := e.executeSingle(ctx, step, input)
		if err == nil {
			return output, nil
		}

		lastErr = err

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}

			if step.RetryPolicy != nil {
				delay = time.Duration(float64(delay) * step.RetryPolicy.BackoffMultiplier)
				if delay > step.RetryPolicy.MaxDelay {
					delay = step.RetryPolicy.MaxDelay
				}
			}
		}
	}

	return "", lastErr
}

// executeSingle executes a step once.
func (e *Executor) executeSingle(ctx context.Context, step *Step, input string) (string, error) {
	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	executor := NewAgentExecutor(e.registry)
	return executor.Execute(stepCtx, step, input, &models.TaskContext{})
}

// generateExecutionID generates a unique execution ID.
func generateExecutionID() string {
	return fmt.Sprintf("exec-%d", time.Now().UnixNano())
}
