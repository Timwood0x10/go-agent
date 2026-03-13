package ratelimit

import (
	"context"
	"time"
)

// Limiter defines the rate limiter interface.
type Limiter interface {
	// Allow checks if a request is allowed.
	Allow(ctx context.Context) (bool, error)
	// Wait blocks until a request can be processed or context is cancelled.
	Wait(ctx context.Context) error
	// Reset resets the limiter state.
	Reset()
	// Rate returns the current rate.
	Rate() float64
}

// LimiterConfig holds limiter configuration.
type LimiterConfig struct {
	Rate       float64       // requests per second
	Burst      int           // maximum burst size
	Timeout    time.Duration // wait timeout
	RefillRate time.Duration // token refill interval
}

// DefaultConfig returns default configuration.
func DefaultConfig() *LimiterConfig {
	return &LimiterConfig{
		Rate:       10,
		Burst:      20,
		Timeout:    5 * time.Second,
		RefillRate: time.Second,
	}
}

// LimiterType represents the type of limiter.
type LimiterType string

const (
	LimiterTypeTokenBucket   LimiterType = "token_bucket"
	LimiterTypeSlidingWindow LimiterType = "sliding_window"
	LimiterTypeSemaphore     LimiterType = "semaphore"
)

// Factory creates limiters based on type.
type Factory struct {
	creators map[LimiterType]func(*LimiterConfig) Limiter
}

// NewFactory creates a new Factory.
func NewFactory() *Factory {
	f := &Factory{
		creators: make(map[LimiterType]func(*LimiterConfig) Limiter),
	}

	f.Register(LimiterTypeTokenBucket, func(config *LimiterConfig) Limiter {
		return NewTokenBucketLimiter(config)
	})

	f.Register(LimiterTypeSlidingWindow, func(config *LimiterConfig) Limiter {
		return NewSlidingWindowLimiter(config)
	})

	f.Register(LimiterTypeSemaphore, func(config *LimiterConfig) Limiter {
		return NewSemaphoreLimiter(config)
	})

	return f
}

// Register registers a limiter creator.
func (f *Factory) Register(limiterType LimiterType, creator func(*LimiterConfig) Limiter) {
	f.creators[limiterType] = creator
}

// Create creates a limiter by type.
func (f *Factory) Create(limiterType LimiterType, config *LimiterConfig) (Limiter, error) {
	creator, exists := f.creators[limiterType]
	if !exists {
		return nil, ErrUnsupportedLimiterType
	}

	if config == nil {
		config = DefaultConfig()
	}

	return creator(config), nil
}

// DefaultFactory is the default limiter factory.
var DefaultFactory = NewFactory()

// CreateLimiter creates a limiter using the default factory.
func CreateLimiter(limiterType LimiterType, config *LimiterConfig) (Limiter, error) {
	return DefaultFactory.Create(limiterType, config)
}

// Limiter errors.
var (
	ErrUnsupportedLimiterType = &LimiterError{"unsupported limiter type"}
)

// LimiterError represents a limiter error.
type LimiterError struct {
	msg string
}

func (e *LimiterError) Error() string {
	return e.msg
}
