// package graph - provides dynamic agent orchestration with pluggable scheduling.

package graph

// Scheduler defines the interface for node scheduling.
type Scheduler interface {
	// Select returns the next node ID to execute from the ready queue.
	Select(ready []string) string
}

// DefaultScheduler provides FIFO scheduling, consistent with Workflow Engine.
type DefaultScheduler struct{}

// NewDefaultScheduler creates a new default scheduler.
func NewDefaultScheduler() *DefaultScheduler {
	return &DefaultScheduler{}
}

// Select returns the first ready node (FIFO).
func (s *DefaultScheduler) Select(ready []string) string {
	if len(ready) == 0 {
		return ""
	}
	return ready[0]
}

// PriorityScheduler provides priority-based scheduling.
type PriorityScheduler struct {
	priorities map[string]int
}

// NewPriorityScheduler creates a new priority scheduler.
func NewPriorityScheduler(priorities map[string]int) *PriorityScheduler {
	if priorities == nil {
		priorities = make(map[string]int)
	}
	return &PriorityScheduler{priorities: priorities}
}

// Select returns the ready node with the highest priority.
func (s *PriorityScheduler) Select(ready []string) string {
	if len(ready) == 0 {
		return ""
	}

	bestNode := ready[0]
	bestPriority := s.getPriority(bestNode)

	for _, nodeID := range ready[1:] {
		priority := s.getPriority(nodeID)
		if priority > bestPriority {
			bestNode = nodeID
			bestPriority = priority
		}
	}

	return bestNode
}

// getPriority returns the priority for a node ID, defaulting to 0.
func (s *PriorityScheduler) getPriority(nodeID string) int {
	if s == nil || s.priorities == nil {
		return 0
	}
	priority, ok := s.priorities[nodeID]
	if !ok {
		return 0
	}
	return priority
}

// ShortJobScheduler provides shortest-job-first scheduling.
type ShortJobScheduler struct {
	estimates map[string]int // estimated latency in milliseconds
}

// NewShortJobScheduler creates a new short-job scheduler.
func NewShortJobScheduler(estimates map[string]int) *ShortJobScheduler {
	if estimates == nil {
		estimates = make(map[string]int)
	}
	return &ShortJobScheduler{estimates: estimates}
}

// Select returns the ready node with the shortest estimated execution time.
func (s *ShortJobScheduler) Select(ready []string) string {
	if len(ready) == 0 {
		return ""
	}

	bestNode := ready[0]
	bestEstimate := s.getEstimate(bestNode)

	for _, nodeID := range ready[1:] {
		estimate := s.getEstimate(nodeID)
		if estimate < bestEstimate {
			bestNode = nodeID
			bestEstimate = estimate
		}
	}

	return bestNode
}

// getEstimate returns the estimated latency for a node ID, defaulting to max int.
func (s *ShortJobScheduler) getEstimate(nodeID string) int {
	if s == nil || s.estimates == nil {
		return 1<<31 - 1 // max int
	}
	estimate, ok := s.estimates[nodeID]
	if !ok {
		return 1<<31 - 1 // max int
	}
	return estimate
}
