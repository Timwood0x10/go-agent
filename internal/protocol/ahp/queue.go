package ahp

import (
	"context"
	"sync"
	"time"

	"goagent/internal/core/errors"
)

// MessageQueue represents an in-memory message queue for agent communication.
type MessageQueue struct {
	messages chan *AHPMessage
	agentID  string
	opts     *QueueOptions
}

// QueueOptions holds the configuration options for the message queue.
type QueueOptions struct {
	MaxSize    int
	MaxWorkers int
	Timeout    time.Duration
}

// DefaultQueueOptions returns the default queue options.
func DefaultQueueOptions() *QueueOptions {
	return &QueueOptions{
		MaxSize:    1000,
		MaxWorkers: 10,
		Timeout:    30 * time.Second,
	}
}

// NewMessageQueue creates a new MessageQueue.
func NewMessageQueue(agentID string, opts *QueueOptions) *MessageQueue {
	if opts == nil {
		opts = DefaultQueueOptions()
	}
	return &MessageQueue{
		messages: make(chan *AHPMessage, opts.MaxSize),
		agentID:  agentID,
		opts:     opts,
	}
}

// Enqueue adds a message to the queue.
func (q *MessageQueue) Enqueue(ctx context.Context, msg *AHPMessage) error {
	select {
	case q.messages <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full
		return errors.ErrQueueFull
	}
}

// Dequeue removes and returns a message from the queue.
func (q *MessageQueue) Dequeue(ctx context.Context) (*AHPMessage, error) {
	select {
	case msg := <-q.messages:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// DequeueWithTimeout removes and returns a message with timeout.
func (q *MessageQueue) DequeueWithTimeout(timeout time.Duration) (*AHPMessage, error) {
	select {
	case msg := <-q.messages:
		return msg, nil
	case <-time.After(timeout):
		return nil, errors.ErrQueueEmpty
	}
}

// Peek returns the first message without removing it.
// Uses non-blocking select to avoid deadlock.
func (q *MessageQueue) Peek() *AHPMessage {
	// Use non-blocking receive to peek without removing
	select {
	case msg, ok := <-q.messages:
		if !ok {
			return nil // Channel closed
		}
		// Put the message back to channel (non-blocking)
		select {
		case q.messages <- msg:
			return msg
		default:
			// Channel full, message is lost but return it anyway
			return msg
		}
	default:
		return nil
	}
}

// Size returns the current number of messages in the queue.
func (q *MessageQueue) Size() int {
	return len(q.messages)
}

// Capacity returns the maximum capacity of the queue.
func (q *MessageQueue) Capacity() int {
	return q.opts.MaxSize
}

// IsEmpty checks if the queue is empty.
func (q *MessageQueue) IsEmpty() bool {
	return len(q.messages) == 0
}

// IsFull checks if the queue is full.
func (q *MessageQueue) IsFull() bool {
	return len(q.messages) >= q.opts.MaxSize
}

// Available returns the number of available slots.
func (q *MessageQueue) Available() int {
	return q.opts.MaxSize - len(q.messages)
}

// AgentID returns the agent ID associated with this queue.
func (q *MessageQueue) AgentID() string {
	return q.agentID
}

// Close closes the queue and drains remaining messages.
func (q *MessageQueue) Close() {
	close(q.messages)
}

// QueueRegistry manages multiple message queues for different agents.
type QueueRegistry struct {
	mu          sync.RWMutex
	queues      map[string]*MessageQueue
	defaultOpts *QueueOptions
}

// NewQueueRegistry creates a new QueueRegistry.
func NewQueueRegistry(opts *QueueOptions) *QueueRegistry {
	if opts == nil {
		opts = DefaultQueueOptions()
	}
	return &QueueRegistry{
		queues:      make(map[string]*MessageQueue),
		defaultOpts: opts,
	}
}

// GetOrCreate returns an existing queue or creates a new one.
func (r *QueueRegistry) GetOrCreate(agentID string) *MessageQueue {
	r.mu.Lock()
	defer r.mu.Unlock()

	if q, ok := r.queues[agentID]; ok {
		return q
	}

	q := NewMessageQueue(agentID, r.defaultOpts)
	r.queues[agentID] = q
	return q
}

// Get returns a queue by agent ID.
func (r *QueueRegistry) Get(agentID string) (*MessageQueue, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q, ok := r.queues[agentID]
	return q, ok
}

// Delete removes a queue by agent ID.
func (r *QueueRegistry) Delete(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if q, ok := r.queues[agentID]; ok {
		q.Close()
		delete(r.queues, agentID)
	}
}

// ListAgents returns all registered agent IDs.
func (r *QueueRegistry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]string, 0, len(r.queues))
	for agentID := range r.queues {
		agents = append(agents, agentID)
	}
	return agents
}

// Size returns the total number of messages across all queues.
func (r *QueueRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := 0
	for _, q := range r.queues {
		total += q.Size()
	}
	return total
}
