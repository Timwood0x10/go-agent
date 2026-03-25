// package graph - tests for graph executor.

package graph

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecuteWithNilGraph(t *testing.T) {
	state := NewState()
	_, err := (*Graph)(nil).Execute(context.Background(), state)
	if err == nil {
		t.Error("expected error for nil graph")
	}
}

func TestExecuteWithNoStartNode(t *testing.T) {
	graph := NewGraph("test")
	state := NewState()
	_, err := graph.Execute(context.Background(), state)
	if err == nil {
		t.Error("expected error for graph without start node")
	}
}

func TestExecuteWithInvalidStartNode(t *testing.T) {
	graph := NewGraph("test").Start("nonexistent")
	state := NewState()
	_, err := graph.Execute(context.Background(), state)
	if err == nil {
		t.Error("expected error for invalid start node")
	}
}

func TestExecuteWithNilScheduler(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil scheduler")
		}
	}()

	graph := NewGraph("test")
	graph.SetScheduler(nil)
}

func TestExecuteWithComplexDAG(t *testing.T) {
	// Test a complex DAG with multiple paths and conditions
	executionOrder := []string{}

	graph := NewGraph("complex").
		Node("start", &mockNode{id: "start", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "start")
			state.Set("stage", "1")
			return nil
		}}).
		Node("branch1", &mockNode{id: "branch1", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "branch1")
			return nil
		}}).
		Node("branch2", &mockNode{id: "branch2", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "branch2")
			return nil
		}}).
		Node("merge", &mockNode{id: "merge", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "merge")
			return nil
		}}).
		Node("end", &mockNode{id: "end", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "end")
			return nil
		}}).
		Edge("start", "branch1").
		Edge("start", "branch2").
		Edge("branch1", "merge").
		Edge("branch2", "merge").
		Edge("merge", "end").
		Start("start")

	state := NewState()
	result, err := graph.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify execution completed successfully
	if result.State == nil {
		t.Error("expected non-nil state")
	}

	// Check that all nodes were executed
	expectedNodes := []string{"start", "branch1", "branch2", "merge", "end"}
	if len(executionOrder) != len(expectedNodes) {
		t.Errorf("expected %d nodes, got %d", len(expectedNodes), len(executionOrder))
	}

	// Check that start was first
	if executionOrder[0] != "start" {
		t.Errorf("expected start first, got %s", executionOrder[0])
	}

	// Check that end was last
	if executionOrder[len(executionOrder)-1] != "end" {
		t.Errorf("expected end last, got %s", executionOrder[len(executionOrder)-1])
	}
}

func TestExecuteWithMultipleConditions(t *testing.T) {
	executionOrder := []string{}

	graph := NewGraph("multi-condition").
		Node("check", &mockNode{id: "check", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "check")
			state.Set("value", 5)
			return nil
		}}).
		Node("path1", &mockNode{id: "path1", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "path1")
			return nil
		}}).
		Node("path2", &mockNode{id: "path2", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "path2")
			return nil
		}}).
		Node("path3", &mockNode{id: "path3", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "path3")
			return nil
		}}).
		Edge("check", "path1", IfFunc(func(s *State) bool {
			val, _ := s.Get("value")
			intVal, ok := val.(int)
			return ok && intVal < 5
		})).
		Edge("check", "path2", IfFunc(func(s *State) bool {
			val, _ := s.Get("value")
			intVal, ok := val.(int)
			return ok && intVal == 5
		})).
		Edge("check", "path3", IfFunc(func(s *State) bool {
			val, _ := s.Get("value")
			intVal, ok := val.(int)
			return ok && intVal > 5
		})).
		Start("check")

	state := NewState()
	_, err := graph.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Only check and path2 should execute (value == 5)
	if len(executionOrder) != 2 {
		t.Errorf("expected 2 nodes, got %d: %v", len(executionOrder), executionOrder)
	}
	if executionOrder[0] != "check" {
		t.Errorf("expected check first, got %s", executionOrder[0])
	}
	if executionOrder[1] != "path2" {
		t.Errorf("expected path2 second, got %s", executionOrder[1])
	}
}

func TestExecuteWithCycleDetection(t *testing.T) {
	// This test verifies that the executor can handle graphs with potential cycles
	// by using proper executed tracking
	executionOrder := []string{}

	graph := NewGraph("cycle").
		Node("node1", &mockNode{id: "node1", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "node1")
			return nil
		}}).
		Node("node2", &mockNode{id: "node2", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "node2")
			return nil
		}}).
		Node("node3", &mockNode{id: "node3", executeFn: func(ctx context.Context, state *State) error {
			executionOrder = append(executionOrder, "node3")
			return nil
		}}).
		Edge("node1", "node2").
		Edge("node2", "node3").
		Edge("node3", "node2"). // This creates a potential cycle, but executed tracking should prevent it
		Start("node1")

	state := NewState()
	_, err := graph.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify that node2 and node3 were only executed once each
	node2Count := 0
	node3Count := 0
	for _, node := range executionOrder {
		if node == "node2" {
			node2Count++
		}
		if node == "node3" {
			node3Count++
		}
	}

	if node2Count != 1 {
		t.Errorf("expected node2 to execute once, got %d times", node2Count)
	}
	if node3Count != 1 {
		t.Errorf("expected node3 to execute once, got %d times", node3Count)
	}
}

func TestExecuteWithEmptyReadyQueue(t *testing.T) {
	graph := NewGraph("empty").
		Node("node1", &mockNode{id: "node1", executeFn: func(ctx context.Context, state *State) error {
			return nil
		}}).
		Start("node1")

	state := NewState()
	_, err := graph.Execute(context.Background(), state)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteWithStateMutation(t *testing.T) {
	graph := NewGraph("mutation").
		Node("node1", &mockNode{id: "node1", executeFn: func(ctx context.Context, state *State) error {
			state.Set("key1", "value1")
			state.Set("key2", 42)
			return nil
		}}).
		Node("node2", &mockNode{id: "node2", executeFn: func(ctx context.Context, state *State) error {
			// Read values set by node1
			val1, _ := state.Get("key1")
			val2, _ := state.Get("key2")

			if val1 != "value1" {
				t.Errorf("expected key1 to be value1, got %v", val1)
			}
			if val2 != 42 {
				t.Errorf("expected key2 to be 42, got %v", val2)
			}

			// Add new value
			state.Set("key3", "value3")
			return nil
		}}).
		Edge("node1", "node2").
		Start("node1")

	state := NewState()
	result, err := graph.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify all keys are present
	val1, ok := result.State.Get("key1")
	if !ok || val1 != "value1" {
		t.Error("expected key1 in final state")
	}

	val2, ok := result.State.Get("key2")
	if !ok || val2 != 42 {
		t.Error("expected key2 in final state")
	}

	val3, ok := result.State.Get("key3")
	if !ok || val3 != "value3" {
		t.Error("expected key3 in final state")
	}
}

func TestExecuteWithDuration(t *testing.T) {
	graph := NewGraph("duration").
		Node("node1", &mockNode{id: "node1", executeFn: func(ctx context.Context, state *State) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}}).
		Start("node1")

	start := time.Now()
	state := NewState()
	result, err := graph.Execute(context.Background(), state)

	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	_ = time.Since(start) // Track actual execution time

	if result.Duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", result.Duration)
	}

	if result.Duration > 200*time.Millisecond {
		t.Errorf("expected duration < 200ms, got %v", result.Duration)
	}
}

func TestExecuteWithNilState(t *testing.T) {
	graph := NewGraph("test").
		Node("node1", &mockNode{id: "node1", executeFn: func(ctx context.Context, state *State) error {
			if state == nil {
				return errors.New("state is nil")
			}
			return nil
		}}).
		Start("node1")

	_, err := graph.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil state")
	}
}