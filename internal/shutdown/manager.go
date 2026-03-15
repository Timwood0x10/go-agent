package shutdown

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Phase represents a shutdown phase.
type Phase int

const (
	PhasePreShutdown Phase = iota
	PhaseGraceful
	PhaseForce
	PhaseDone
)

// Phase names for logging.
var phaseNames = map[Phase]string{
	PhasePreShutdown: "pre-shutdown",
	PhaseGraceful:    "graceful",
	PhaseForce:       "force",
	PhaseDone:        "done",
}

// String returns the phase name.
func (p Phase) String() string {
	name, ok := phaseNames[p]
	if !ok {
		return "unknown"
	}
	return name
}

// IsValid checks if the phase is valid.
func (p Phase) IsValid() bool {
	_, ok := phaseNames[p]
	return ok
}

// Manager coordinates the shutdown process across multiple components.
type Manager struct {
	phases       map[Phase]*PhaseHandler
	currentPhase Phase
	mu           sync.RWMutex
	timeout      time.Duration
	wg           sync.WaitGroup
}

// PhaseHandler handles a specific shutdown phase.
type PhaseHandler struct {
	phase     Phase
	callbacks []Callback
	timeout   time.Duration
	onTimeout func()
	onPanic   func(interface{})
}

// Callback is a function called during shutdown.
type Callback func(ctx context.Context) error

// NewManager creates a new ShutdownManager.
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		phases:  make(map[Phase]*PhaseHandler),
		timeout: timeout,
	}
}

// RegisterPhase registers a handler for a phase.
func (m *Manager) RegisterPhase(phase Phase, timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.phases[phase] = &PhaseHandler{
		phase:   phase,
		timeout: timeout,
	}
}

// AddCallback adds a callback to a phase.
func (m *Manager) AddCallback(phase Phase, callback Callback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handler, exists := m.phases[phase]
	if !exists {
		return fmt.Errorf("phase %s not registered", phase)
	}

	handler.callbacks = append(handler.callbacks, callback)
	return nil
}

// StartShutdown initiates the shutdown process.
func (m *Manager) StartShutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.currentPhase != 0 {
		m.mu.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}
	m.mu.Unlock()

	// Execute phases in order
	phases := []Phase{PhasePreShutdown, PhaseGraceful, PhaseForce, PhaseDone}

	for _, phase := range phases {
		m.mu.Lock()
		m.currentPhase = phase
		m.mu.Unlock()

		if err := m.executePhase(ctx, phase); err != nil {
			return fmt.Errorf("phase %s failed: %w", phase, err)
		}
	}

	return nil
}

// executePhase executes all callbacks for a phase.
func (m *Manager) executePhase(ctx context.Context, phase Phase) error {
	m.mu.RLock()
	handler, exists := m.phases[phase]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	if len(handler.callbacks) == 0 {
		return nil
	}

	phaseCtx, cancel := context.WithTimeout(ctx, handler.timeout)
	defer cancel()

	errChan := make(chan error, len(handler.callbacks))

	for _, callback := range handler.callbacks {
		m.wg.Add(1)
		go func(cb Callback) {
			defer m.wg.Done()
			if err := cb(phaseCtx); err != nil {
				errChan <- err
			}
		}(callback)
	}

	// Wait for all callbacks or timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
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
	case <-phaseCtx.Done():
		if handler.onTimeout != nil {
			handler.onTimeout()
		}
		return phaseCtx.Err()
	}
}

// SetOnTimeout sets the callback for phase timeout.
func (m *Manager) SetOnTimeout(phase Phase, fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if handler, exists := m.phases[phase]; exists {
		handler.onTimeout = fn
	}
}

// SetOnPanic sets the callback for panic during phase execution.
func (m *Manager) SetOnPanic(phase Phase, fn func(interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if handler, exists := m.phases[phase]; exists {
		handler.onPanic = fn
	}
}

// CurrentPhase returns the current shutdown phase.
func (m *Manager) CurrentPhase() Phase {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.currentPhase
}

// Wait waits for all in-progress operations to complete.
func (m *Manager) Wait() {
	m.wg.Wait()
}

// IsShutdown returns true if shutdown has started (past PhasePreShutdown).
func (m *Manager) IsShutdown() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Shutdown has started if we're past PhasePreShutdown
	return m.currentPhase > PhasePreShutdown
}
