# RateLimiter Design Document

## 1. Overview

The RateLimiter module provides rate limiting and backpressure mechanisms to control request rates and prevent system overload.

## 2. Rate Limiting Strategies

| Strategy | Use Case | Implementation |
|----------|----------|----------------|
| Token Bucket | LLM request rate limiting | Smooth rate |
| Sliding Window | Global QPS control | Precise counting |
| Semaphore | Agent concurrency control | Resource pool |
| Queue Length | Task queue rate limiting | Queue capacity |

## 3. Core Interfaces

```go
type Limiter interface {
    // Allow check if allowed
    Allow() (bool, error)
    
    // Wait wait for token
    Wait(ctx context.Context) error
    
    // Reset reset limiter
    Reset()
    
    // Stats get statistics
    Stats() *LimiterStats
}

type LimiterStats struct {
    TotalRequests  int64   `json:"total_requests"`
    AllowedRequests int64  `json:"allowed_requests"`
    RejectedRequests int64 `json:"rejected_requests"`
    CurrentRate    float64 `json:"current_rate"`
}
```

## 4. Limiter Implementations

### 4.1 Token Bucket

```go
type TokenBucket struct {
    rate       float64     // Tokens per second
    capacity   int         // Bucket capacity
    tokens     float64    // Current tokens
    lastUpdate time.Time   // Last update time
    mu         sync.Mutex
}

func NewTokenBucket(rate float64, capacity int) *TokenBucket {
    return &TokenBucket{
        rate:       rate,
        capacity:   capacity,
        tokens:     float64(capacity),
        lastUpdate: time.Now(),
    }
}

func (tb *TokenBucket) Allow() (bool, error) {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    tb.refill()
    
    if tb.tokens >= 1 {
        tb.tokens--
        return true, nil
    }
    return false, nil
}

func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastUpdate).Seconds()
    tb.tokens = math.Min(float64(tb.capacity), tb.tokens+elapsed*tb.rate)
    tb.lastUpdate = now
}
```

### 4.2 Sliding Window

```go
type SlidingWindow struct {
    windowSize  time.Duration // Window size
    maxRequests int           // Max requests in window
    requests   []time.Time   // Request timestamps
    mu         sync.Mutex
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int) *SlidingWindow {
    return &SlidingWindow{
        windowSize:  windowSize,
        maxRequests: maxRequests,
        requests:    make([]time.Time, 0, maxRequests),
    }
}

func (sw *SlidingWindow) Allow() (bool, error) {
    sw.mu.Lock()
    defer sw.mu.Unlock()
    
    now := time.Now()
    cutoff := now.Add(-sw.windowSize)
    
    // Clean up expired requests
    var valid []time.Time
    for _, t := range sw.requests {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }
    sw.requests = valid
    
    // Check if limit exceeded
    if len(sw.requests) >= sw.maxRequests {
        return false, nil
    }
    
    sw.requests = append(sw.requests, now)
    return true, nil
}
```

### 4.3 Semaphore

```go
type SemaphoreLimiter struct {
    sem *semaphore.Weighted
    mu  sync.Mutex
}

func NewSemaphoreLimiter(permits int) *SemaphoreLimiter {
    return &SemaphoreLimiter{
        sem: semaphore.NewWeighted(int64(permits)),
    }
}

func (s *SemaphoreLimiter) Acquire(ctx context.Context) error {
    return s.sem.Acquire(ctx, 1)
}

func (s *SemaphoreLimiter) Release() {
    s.sem.Release(1)
}
```

## 5. Backpressure Mechanism

```go
type Backpressure struct {
    queueLimit    int
    currentLoad   atomic.Int32
    rejectionRate float64
    
    // Response headers
    RetryAfter   time.Duration
    RetryCount   int
    
    // Alert callback
    OnThreshold  func(load int)
}

func (bp *Backpressure) Check() (bool, int) {
    load := int(bp.currentLoad.Load())
    percentage := float64(load) / float64(bp.queueLimit)
    
    switch {
    case percentage >= 1.0:
        // Queue full, reject new tasks
        return false, http.StatusServiceUnavailable
    case percentage >= 0.9:
        // 90% alert
        if bp.OnThreshold != nil {
            bp.OnThreshold(load)
        }
        return false, http.StatusTooManyRequests
    case percentage >= 0.8:
        // 80% alert
        if bp.OnThreshold != nil {
            bp.OnThreshold(load)
        }
    }
    
    return true, http.StatusOK
}
```

## 6. Usage Example

```go
// Create limiters
llmLimiter := ratelimit.NewTokenBucket(10, 50)      // LLM: 10 req/s
agentLimiter := ratelimit.NewSemaphoreLimiter(10)   // Agent: 10 concurrent
globalLimiter := ratelimit.NewSlidingWindow(1*time.Second, 100) // Global: 100 QPS

// Use in request handling
func handleRequest(ctx context.Context) error {
    // 1. Global limit
    if ok, _ := globalLimiter.Allow(); !ok {
        return ErrRateLimitExceeded
    }
    
    // 2. LLM limit
    if err := llmLimiter.Wait(ctx); err != nil {
        return err
    }
    
    // 3. Agent limit
    if err := agentLimiter.Acquire(ctx); err != nil {
        return err
    }
    defer agentLimiter.Release()
    
    // Process request
    return doProcess(ctx)
}
```

## 7. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| llm_rate | 10 | LLM requests per second |
| llm_burst | 50 | LLM burst capacity |
| agent_concurrency | 10 | Agent max concurrency |
| global_qps | 100 | Global QPS limit |
| queue_threshold | 0.8 | Queue threshold |
| backoff_base | 1s | Backoff base time |
