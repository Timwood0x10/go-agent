# RateLimiter 设计文档

## 1. 概述

RateLimiter 模块提供限流和背压机制，用于控制请求速率、防止系统过载。

## 2. 限流策略

| 策略 | 适用场景 | 实现方式 |
|------|----------|----------|
| 令牌桶 | LLM 请求限流 | 平滑速率 |
| 滑动窗口 | 全局 QPS 控制 | 精确计数 |
| 信号量 | Agent 并发控制 | 资源池 |
| 队列长度 | 任务队列限流 | 队列容量 |

## 3. 核心接口

```go
type Limiter interface {
    // Allow 检查是否允许通过
    Allow() (bool, error)
    
    // Wait 等待获取令牌
    Wait(ctx context.Context) error
    
    // Reset 重置限流器
    Reset()
    
    // Stats 获取统计信息
    Stats() *LimiterStats
}

type LimiterStats struct {
    TotalRequests  int64   `json:"total_requests"`
    AllowedRequests int64  `json:"allowed_requests"`
    RejectedRequests int64 `json:"rejected_requests"`
    CurrentRate    float64 `json:"current_rate"`
}
```

## 4. 限流器实现

### 4.1 令牌桶

```go
type TokenBucket struct {
    rate       float64     // 每秒令牌数
    capacity   int         // 桶容量
    tokens     float64    // 当前令牌数
    lastUpdate time.Time   // 上次更新时间
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

### 4.2 滑动窗口

```go
type SlidingWindow struct {
    windowSize  time.Duration // 窗口大小
    maxRequests int           // 窗口内最大请求数
    requests   []time.Time   // 请求时间戳
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
    
    // 清理过期的请求
    var valid []time.Time
    for _, t := range sw.requests {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }
    sw.requests = valid
    
    // 检查是否超过限制
    if len(sw.requests) >= sw.maxRequests {
        return false, nil
    }
    
    sw.requests = append(sw.requests, now)
    return true, nil
}
```

### 4.3 信号量

```go
type SemaphoreLimiter struct {
    sem *semaphore.Weighted
    mu  sync.Mutex
}

func NewSemaphoreLimiter permits int) *SemaphoreLimiter {
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

## 5. 背压机制

```go
type Backpressure struct {
    queueLimit    int
    currentLoad   atomic.Int32
    rejectionRate float64
    
    // 响应头
    RetryAfter   time.Duration
    RetryCount   int
    
    // 告警回调
    OnThreshold  func(load int)
}

func (bp *Backpressure) Check() (bool, int) {
    load := int(bp.currentLoad.Load())
    percentage := float64(load) / float64(bp.queueLimit)
    
    switch {
    case percentage >= 1.0:
        // 队列满，拒绝新任务
        return false, http.StatusServiceUnavailable
    case percentage >= 0.9:
        // 90% 告警
        if bp.OnThreshold != nil {
            bp.OnThreshold(load)
        }
        return false, http.StatusTooManyRequests
    case percentage >= 0.8:
        // 80% 告警
        if bp.OnThreshold != nil {
            bp.OnThreshold(load)
        }
    }
    
    return true, http.StatusOK
}
```

## 6. 使用示例

```go
// 创建限流器
llmLimiter := ratelimit.NewTokenBucket(10, 50)      // LLM: 10 req/s
agentLimiter := ratelimit.NewSemaphoreLimiter(10)    // Agent: 10 并发
globalLimiter := ratelimit.NewSlidingWindow(1*time.Second, 100) // 全局: 100 QPS

// 在请求处理中使用
func handleRequest(ctx context.Context) error {
    // 1. 全局限流
    if ok, _ := globalLimiter.Allow(); !ok {
        return ErrRateLimitExceeded
    }
    
    // 2. LLM 限流
    if err := llmLimiter.Wait(ctx); err != nil {
        return err
    }
    
    // 3. Agent 限流
    if err := agentLimiter.Acquire(ctx); err != nil {
        return err
    }
    defer agentLimiter.Release()
    
    // 处理请求
    return doProcess(ctx)
}
```

## 7. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| llm_rate | 10 | LLM 每秒请求数 |
| llm_burst | 50 | LLM 突发容量 |
| agent_concurrency | 10 | Agent 最大并发 |
| global_qps | 100 | 全局 QPS 限制 |
| queue_threshold | 0.8 | 队列阈值 |
| backoff_base | 1s | 退避基础时间 |
