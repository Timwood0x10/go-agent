// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"goagent/internal/core/errors"
	"sync"
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
	halfOpenMaxCalls int
	halfOpenSuccess  int
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
		successThreshold: 3, // Number of successes needed to close circuit in half-open state
		openTimeout:      openTimeout,
		halfOpenMaxCalls: 5,
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
		if cb.halfOpenSuccess >= cb.halfOpenMaxCalls {
			// Circuit recovered, close it
			cb.state = CircuitBreakerStateClosed
			cb.failureCount = 0
			cb.halfOpenSuccess = 0
		}
		return nil

	default:
		return errors.ErrInvalidState
	}
}

// RecordSuccess records a successful operation.
// This is called after a request completes successfully.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitBreakerStateHalfOpen {
		cb.halfOpenSuccess++
	}

	cb.failureCount = 0
}

// RecordFailure records a failed operation.
// This is called when a request fails and may trigger circuit opening.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	// Open circuit if failure threshold reached
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
// This is primarily used for testing purposes.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitBreakerStateClosed
	cb.failureCount = 0
	cb.lastFailureTime = time.Time{}
	cb.halfOpenSuccess = 0
}
