// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"goagent/internal/core/errors"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "closed"
	CircuitBreakerStateOpen     CircuitBreakerState = "open"
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreaker provides failure detection and automatic fallback for unreliable services.
// This implements the circuit breaker pattern to prevent cascading failures.
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int
	failureThreshold int
	successThreshold int
	lastFailureTime  time.Time
	openTimeout      time.Duration
	halfOpenSuccess  int
	halfOpenInflight atomic.Int32
}

// NewCircuitBreaker creates a new CircuitBreaker instance.
// Args:
// failureThreshold - number of failures before opening the circuit.
// openTimeout - time to wait before attempting half-open state.
// Returns new CircuitBreaker instance.
func NewCircuitBreaker(failureThreshold int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitBreakerStateClosed,
		failureThreshold: failureThreshold,
		successThreshold: 3,
		openTimeout:      openTimeout,
	}
}

// AllowRequest checks if a request should be allowed based on circuit breaker state.
// Returns error if circuit is open or enters open state.
func (cb *CircuitBreaker) AllowRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerStateClosed:
		return nil

	case CircuitBreakerStateOpen:
		if time.Since(cb.lastFailureTime) > cb.openTimeout {
			// Move to half-open state
			cb.state = CircuitBreakerStateHalfOpen
			cb.halfOpenSuccess = 0
			return nil
		}
		return errors.ErrCircuitBreakerOpen

	case CircuitBreakerStateHalfOpen:
		inflight := cb.halfOpenInflight.Add(1)
		if inflight > 1 {
			cb.halfOpenInflight.Add(-1)
			return errors.ErrCircuitBreakerOpen
		}
		return nil

	default:
		return errors.ErrInvalidState
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitBreakerStateHalfOpen {
		cb.halfOpenSuccess++
		cb.halfOpenInflight.Add(-1)
		if cb.halfOpenSuccess >= cb.successThreshold {
			cb.state = CircuitBreakerStateClosed
			cb.failureCount = 0
			cb.halfOpenSuccess = 0
		}
	}

	cb.failureCount = 0
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitBreakerStateHalfOpen {
		cb.halfOpenInflight.Add(-1)
		cb.state = CircuitBreakerStateOpen
		cb.lastFailureTime = time.Now()
		return
	}

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = CircuitBreakerStateOpen
	}
}

// State returns the current circuit breaker state.
// Returns current state.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitBreakerStateClosed
	cb.failureCount = 0
	cb.lastFailureTime = time.Time{}
	cb.halfOpenSuccess = 0
	cb.halfOpenInflight.Store(0)
}
