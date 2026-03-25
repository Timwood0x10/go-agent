// package graph - tests for schedulers.

package graph

import (
	"testing"
)

func TestDefaultScheduler(t *testing.T) {
	scheduler := NewDefaultScheduler()

	// Test empty queue
	if id := scheduler.Select([]string{}); id != "" {
		t.Errorf("expected empty string, got %s", id)
	}

	// Test single item
	if id := scheduler.Select([]string{"node1"}); id != "node1" {
		t.Errorf("expected node1, got %s", id)
	}

	// Test multiple items (FIFO)
	queue := []string{"node1", "node2", "node3"}
	if id := scheduler.Select(queue); id != "node1" {
		t.Errorf("expected node1, got %s", id)
	}
}

func TestPriorityScheduler(t *testing.T) {
	priorities := map[string]int{
		"node1": 1,
		"node2": 10,
		"node3": 5,
	}
	scheduler := NewPriorityScheduler(priorities)

	// Test empty queue
	if id := scheduler.Select([]string{}); id != "" {
		t.Errorf("expected empty string, got %s", id)
	}

	// Test highest priority
	queue := []string{"node1", "node2", "node3"}
	if id := scheduler.Select(queue); id != "node2" {
		t.Errorf("expected node2 (priority 10), got %s", id)
	}

	// Test default priority for unknown node
	queue = []string{"unknown"}
	if id := scheduler.Select(queue); id != "unknown" {
		t.Errorf("expected unknown, got %s", id)
	}

	// Test nil priorities
	scheduler = NewPriorityScheduler(nil)
	queue = []string{"node1", "node2"}
	if id := scheduler.Select(queue); id != "node1" {
		t.Errorf("expected node1 (default priority), got %s", id)
	}
}

func TestShortJobScheduler(t *testing.T) {
	estimates := map[string]int{
		"node1": 100,
		"node2": 50,
		"node3": 200,
	}
	scheduler := NewShortJobScheduler(estimates)

	// Test empty queue
	if id := scheduler.Select([]string{}); id != "" {
		t.Errorf("expected empty string, got %s", id)
	}

	// Test shortest job
	queue := []string{"node1", "node2", "node3"}
	if id := scheduler.Select(queue); id != "node2" {
		t.Errorf("expected node2 (50ms), got %s", id)
	}

	// Test default estimate for unknown node
	queue = []string{"unknown"}
	if id := scheduler.Select(queue); id != "unknown" {
		t.Errorf("expected unknown, got %s", id)
	}

	// Test nil estimates
	scheduler = NewShortJobScheduler(nil)
	queue = []string{"node1", "node2"}
	if id := scheduler.Select(queue); id != "node1" {
		t.Errorf("expected node1 (default estimate), got %s", id)
	}
}
