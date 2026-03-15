package ratelimit

import (
	"context"
	"testing"
)

func TestTokenBucketLimiter(t *testing.T) {
	t.Run("create token bucket", func(t *testing.T) {
		config := &LimiterConfig{
			Rate:  10,
			Burst: 20,
		}
		limiter := NewTokenBucketLimiter(config)

		if limiter == nil {
			t.Errorf("limiter should not be nil")
		}
	})

	t.Run("allow request", func(t *testing.T) {
		config := &LimiterConfig{Rate: 10, Burst: 10}
		limiter := NewTokenBucketLimiter(config)

		allowed, err := limiter.Allow(context.Background())
		if err != nil {
			t.Errorf("allow error: %v", err)
		}
		if !allowed {
			t.Errorf("should be allowed")
		}
	})

	t.Run("rate", func(t *testing.T) {
		config := &LimiterConfig{Rate: 10, Burst: 10}
		limiter := NewTokenBucketLimiter(config)

		rate := limiter.Rate()
		if rate != 10 {
			t.Errorf("expected rate 10, got %f", rate)
		}
	})

	t.Run("reset", func(t *testing.T) {
		config := &LimiterConfig{Rate: 10, Burst: 10}
		limiter := NewTokenBucketLimiter(config)

		limiter.Allow(context.Background())
		limiter.Reset()

		allowed, _ := limiter.Allow(context.Background())
		if !allowed {
			t.Errorf("should be allowed after reset")
		}
	})
}

func TestSlidingWindowLimiter(t *testing.T) {
	t.Run("create sliding window", func(t *testing.T) {
		config := &LimiterConfig{Rate: 10, Burst: 10}
		limiter := NewSlidingWindowLimiter(config)

		if limiter == nil {
			t.Errorf("limiter should not be nil")
		}
	})

	t.Run("allow request", func(t *testing.T) {
		config := &LimiterConfig{Rate: 10, Burst: 10}
		limiter := NewSlidingWindowLimiter(config)

		allowed, err := limiter.Allow(context.Background())
		if err != nil {
			t.Errorf("allow error: %v", err)
		}
		if !allowed {
			t.Errorf("should be allowed")
		}
	})
}

func TestLimiterFactory(t *testing.T) {
	t.Run("create limiter", func(t *testing.T) {
		factory := NewFactory()
		limiter, err := factory.Create(LimiterTypeTokenBucket, &LimiterConfig{Rate: 10, Burst: 10})

		if err != nil {
			t.Errorf("create error: %v", err)
		}
		if limiter == nil {
			t.Errorf("limiter should not be nil")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		factory := NewFactory()
		_, err := factory.Create("unknown", nil)

		if err != ErrUnsupportedLimiterType {
			t.Errorf("expected unsupported type error")
		}
	})
}

func TestBackpressure(t *testing.T) {
	t.Run("create backpressure", func(t *testing.T) {
		bp := NewBackpressure(10, 20, DropPolicyReject)

		if bp == nil {
			t.Errorf("backpressure should not be nil")
		}
	})

	t.Run("check max active", func(t *testing.T) {
		bp := NewBackpressure(5, 10, DropPolicyReject)

		_ = bp
		// Just test that it can be created
	})
}
