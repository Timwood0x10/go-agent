package shutdown

import (
	"context"
	"sync"
	"time"
)

// CallbackRegistry manages shutdown callbacks.
type CallbackRegistry struct {
	callbacks map[Phase][]RegisteredCallback
	mu        sync.RWMutex
}

// RegisteredCallback represents a registered callback.
type RegisteredCallback struct {
	ID       string
	Priority int
	Fn       Callback
	Timeout  time.Duration
	OnError  func(error)
}

// NewCallbackRegistry creates a new CallbackRegistry.
func NewCallbackRegistry() *CallbackRegistry {
	return &CallbackRegistry{
		callbacks: make(map[Phase][]RegisteredCallback),
	}
}

// Register registers a callback for a phase.
func (r *CallbackRegistry) Register(phase Phase, id string, priority int, fn Callback, timeout time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	callback := RegisteredCallback{
		ID:       id,
		Priority: priority,
		Fn:       fn,
		Timeout:  timeout,
	}

	r.callbacks[phase] = append(r.callbacks[phase], callback)

	// Sort by priority (higher priority first)
	for i := 0; i < len(r.callbacks[phase])-1; i++ {
		for j := i + 1; j < len(r.callbacks[phase]); j++ {
			if r.callbacks[phase][j].Priority > r.callbacks[phase][i].Priority {
				r.callbacks[phase][i], r.callbacks[phase][j] = r.callbacks[phase][j], r.callbacks[phase][i]
			}
		}
	}

	return nil
}

// Unregister removes a callback by ID.
func (r *CallbackRegistry) Unregister(phase Phase, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	callbacks, exists := r.callbacks[phase]
	if !exists {
		return ErrCallbackNotFound
	}

	for i, cb := range callbacks {
		if cb.ID == id {
			r.callbacks[phase] = append(callbacks[:i], callbacks[i+1:]...)
			return nil
		}
	}

	return ErrCallbackNotFound
}

// GetCallbacks returns callbacks for a phase, sorted by priority.
func (r *CallbackRegistry) GetCallbacks(phase Phase) []Callback {
	r.mu.RLock()
	defer r.mu.RUnlock()

	callbacks, exists := r.callbacks[phase]
	if !exists {
		return nil
	}

	result := make([]Callback, 0, len(callbacks))
	for _, cb := range callbacks {
		result = append(result, cb.Fn)
	}

	return result
}

// Clear removes all callbacks for a phase.
func (r *CallbackRegistry) Clear(phase Phase) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.callbacks, phase)
}

// Count returns the number of callbacks for a phase.
func (r *CallbackRegistry) Count(phase Phase) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.callbacks[phase])
}

// SetOnError sets error handler for a callback.
func (r *CallbackRegistry) SetOnError(phase Phase, id string, onError func(error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	callbacks, exists := r.callbacks[phase]
	if !exists {
		return ErrCallbackNotFound
	}

	for i, cb := range callbacks {
		if cb.ID == id {
			callbacks[i].OnError = onError
			return nil
		}
	}

	return ErrCallbackNotFound
}

// CallbackRegistry errors.
var (
	ErrCallbackNotFound = &CallbackError{"callback not found"}
)

// CallbackError represents a callback error.
type CallbackError struct {
	msg string
}

func (e *CallbackError) Error() string {
	return e.msg
}

// CallbackChain allows chaining multiple callbacks.
type CallbackChain struct {
	callbacks []Callback
}

// NewCallbackChain creates a new CallbackChain.
func NewCallbackChain() *CallbackChain {
	return &CallbackChain{
		callbacks: make([]Callback, 0),
	}
}

// Add adds a callback to the chain.
func (c *CallbackChain) Add(fn Callback) *CallbackChain {
	c.callbacks = append(c.callbacks, fn)
	return c
}

// Execute executes all callbacks in order.
func (c *CallbackChain) Execute(ctx context.Context) error {
	for _, fn := range c.callbacks {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteParallel executes all callbacks in parallel.
func (c *CallbackChain) ExecuteParallel(ctx context.Context) error {
	if len(c.callbacks) == 0 {
		return nil
	}

	errChan := make(chan error, len(c.callbacks))
	done := make(chan struct{})

	for _, fn := range c.callbacks {
		go func(callback Callback) {
			if err := callback(ctx); err != nil {
				errChan <- err
			}
		}(fn)
	}

	go func() {
		for range c.callbacks {
		}
		close(done)
	}()

	select {
	case <-done:
		close(errChan)
		for err := range errChan {
			if err != nil {
				return err
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
