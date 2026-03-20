package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Backpressure provides backpressure mechanisms.
type Backpressure struct {
	queue      chan *BackpressureRequest
	mu         sync.RWMutex
	active     int
	maxActive  int
	queueSize  int
	dropPolicy DropPolicy
	metrics    *BackpressureMetrics
}

// BackpressureRequest represents a request in the queue.
type BackpressureRequest struct {
	Ctx    context.Context
	Key    string
	Weight int
	Result chan *BackpressureResult
}

// BackpressureResult represents the result of a backpressure request.
type BackpressureResult struct {
	Allowed bool
	Error   error
}

// DropPolicy defines how to handle queue overflow.
type DropPolicy string

const (
	DropPolicyReject  DropPolicy = "reject"
	DropPolicyDropOld DropPolicy = "drop_oldest"
	DropPolicyBlock   DropPolicy = "block"
)

// BackpressureMetrics holds backpressure statistics.
type BackpressureMetrics struct {
	Accepted     int64
	Rejected     int64
	Dropped      int64
	Queued       int64
	Processed    int64
	AvgQueueTime time.Duration
}

// NewBackpressure creates a new Backpressure instance.
func NewBackpressure(maxActive, queueSize int, dropPolicy DropPolicy) *Backpressure {
	bp := &Backpressure{
		queue:      make(chan *BackpressureRequest, queueSize),
		maxActive:  maxActive,
		queueSize:  queueSize,
		dropPolicy: dropPolicy,
		metrics:    &BackpressureMetrics{},
	}

	return bp
}

// Submit submits a request for processing.
func (bp *Backpressure) Submit(ctx context.Context, key string, weight int) (*BackpressureResult, error) {
	result := &BackpressureResult{
		Allowed: false,
	}

	bp.mu.Lock()

	// Check if we can process immediately
	if bp.active < bp.maxActive {
		bp.active += weight
		bp.metrics.Accepted++
		bp.mu.Unlock()

		result.Allowed = true
		return result, nil
	}

	bp.mu.Unlock()

	// Try to queue
	bp.metrics.Queued++

	resultChan := make(chan *BackpressureResult, 1)

	select {
	case bp.queue <- &BackpressureRequest{
		Ctx:    ctx,
		Key:    key,
		Weight: weight,
		Result: resultChan,
	}:
		// Wait for actual processing result
		select {
		case bpResult := <-resultChan:
			return bpResult, bpResult.Error
		case <-ctx.Done():
			bp.mu.Lock()
			bp.metrics.Rejected++
			bp.mu.Unlock()
			result.Error = ErrQueueFull
			return result, result.Error
		}
	case <-ctx.Done():
		bp.mu.Lock()
		bp.metrics.Rejected++
		bp.mu.Unlock()
		result.Error = ErrQueueFull
		return result, result.Error
	}
}

// Start starts the processing loop.
func (bp *Backpressure) Start(ctx context.Context, handler func(context.Context, string) error) {
	go bp.processLoop(ctx, handler)
}

// processLoop processes queued requests.
func (bp *Backpressure) processLoop(ctx context.Context, handler func(context.Context, string) error) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-bp.queue:
			bp.mu.Lock()
			bp.active += req.Weight
			bp.metrics.Processed++
			bp.mu.Unlock()

			err := handler(req.Ctx, req.Key)

			bp.mu.Lock()
			bp.active -= req.Weight
			bp.mu.Unlock()

			req.Result <- &BackpressureResult{
				Allowed: err == nil,
				Error:   err,
			}
		}
	}
}

// Metrics returns current metrics.
func (bp *Backpressure) Metrics() *BackpressureMetrics {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	metrics := *bp.metrics
	metrics.Queued = int64(len(bp.queue))
	return &metrics
}

// Reset resets the backpressure state.
func (bp *Backpressure) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.metrics = &BackpressureMetrics{}
	bp.active = 0

	// Drain queue
	for len(bp.queue) > 0 {
		select {
		case req := <-bp.queue:
			req.Result <- &BackpressureResult{
				Allowed: false,
				Error:   ErrReset,
			}
		default:
		}
	}
}

// Active returns the number of active requests.
func (bp *Backpressure) Active() int {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return bp.active
}

// Queued returns the number of queued requests.
func (bp *Backpressure) Queued() int {
	return len(bp.queue)
}

// Backpressure errors.
var (
	ErrQueueFull = errors.New("queue full")
	ErrReset     = errors.New("backpressure reset")
)

// AdaptiveLimiter adjusts rate based on current load.
type AdaptiveLimiter struct {
	limiter        *TokenBucketLimiter
	mu             sync.RWMutex
	minRate        float64
	maxRate        float64
	currentRate    float64
	decreaseFactor float64
	increaseFactor float64
}

// NewAdaptiveLimiter creates a new AdaptiveLimiter.
func NewAdaptiveLimiter(base Limiter, minRate, maxRate float64) *AdaptiveLimiter {
	// Extract current rate from base limiter
	rate := base.Rate()
	burst := int(rate) // Default burst to rate

	config := &LimiterConfig{
		Rate:    rate,
		Burst:   burst,
		Timeout: 30 * time.Second,
	}

	return &AdaptiveLimiter{
		limiter:        NewTokenBucketLimiter(config),
		minRate:        minRate,
		maxRate:        maxRate,
		currentRate:    rate,
		decreaseFactor: 0.9,
		increaseFactor: 1.1,
	}
}

// Allow checks if request is allowed.
func (a *AdaptiveLimiter) Allow(ctx context.Context) (bool, error) {
	a.mu.RLock()
	limiter := a.limiter
	a.mu.RUnlock()
	return limiter.Allow(ctx)
}

// Wait blocks until request can be processed.
func (a *AdaptiveLimiter) Wait(ctx context.Context) error {
	a.mu.RLock()
	limiter := a.limiter
	a.mu.RUnlock()
	return limiter.Wait(ctx)
}

// Reset resets the limiter.
func (a *AdaptiveLimiter) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.limiter.Reset()
	a.currentRate = a.maxRate
}

// Rate returns current rate.
func (a *AdaptiveLimiter) Rate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentRate
}

// Increase increases the rate.
func (a *AdaptiveLimiter) Increase() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.currentRate *= a.increaseFactor
	if a.currentRate > a.maxRate {
		a.currentRate = a.maxRate
	}

	// Update limiter rate
	config := &LimiterConfig{
		Rate:    a.currentRate,
		Burst:   int(a.currentRate),
		Timeout: 30 * time.Second,
	}
	a.limiter = NewTokenBucketLimiter(config)
}

// Decrease decreases the rate.
func (a *AdaptiveLimiter) Decrease() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.currentRate *= a.decreaseFactor
	if a.currentRate < a.minRate {
		a.currentRate = a.minRate
	}

	// Update limiter rate
	config := &LimiterConfig{
		Rate:    a.currentRate,
		Burst:   int(a.currentRate),
		Timeout: 30 * time.Second,
	}
	a.limiter = NewTokenBucketLimiter(config)
}
