# Shutdown 设计文档

## 1. 概述

Shutdown 模块负责系统的优雅退出，确保在服务停止时能够：
- 停止接收新请求
- 处理完当前任务
- 保存系统状态
- 清理资源

## 2. 退出阶段

```
┌─────────────────────────────────────────────────────────────────┐
│                      Shutdown Signal Flow                       │
└─────────────────────────────────────────────────────────────────┘

                       SIGTERM / SIGINT
                              │
                              ▼
                 ┌────────────────────────┐
                 │  Phase 1: Stop Accept  │
                 │  停止接收新请求/任务     │
                 └───────────┬────────────┘
                             │
                             │ 等待 30s (grace period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 2: Cancel       │
                 │  取消所有 Agent 上下文   │
                 └───────────┬────────────┘
                             │
                             │ 等待 60s (shutdown period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 3: Drain Queues │
                 │  处理完积压消息          │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 4: Save State   │
                 │  落盘内存状态            │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 5: Close        │
                 │  关闭 DB/连接池         │
                 └───────────┬────────────┘
                             │
                             ▼
                          EXIT 0
```

## 3. 核心结构

```go
type ShutdownManager struct {
    gracePeriod     time.Duration     // 30s, 优雅期
    shutdownPeriod  time.Duration     // 60s, 关闭期
    forceTimeout    time.Duration     // 90s, 强制超时
    
    phase    atomic.Int32              // 当前阶段
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    
    // 回调函数
    onStopAccept   []func()           // 停止接收回调
    onCancel        []func()           // 取消回调
    onDrain        []func()           // 排空回调
    onSaveState    []func()           // 保存状态回调
    onClose        []func()           // 关闭回调
}

type Phase int

const (
    PhaseRunning    Phase = 0  // 运行中
    PhaseStopping   Phase = 1  // 停止接收
    PhaseDraining   Phase = 2  // 排空中
    PhaseExiting    Phase = 3  // 退出中
)
```

## 4. 实现代码

```go
func NewShutdownManager(gracePeriod, shutdownPeriod time.Duration) *ShutdownManager {
    ctx, cancel := context.WithCancel(context.Background())
    return &ShutdownManager{
        gracePeriod:    gracePeriod,
        shutdownPeriod: shutdownPeriod,
        forceTimeout:   gracePeriod + shutdownPeriod + 30*time.Second,
        phase:          atomic.Int32{},
        ctx:            ctx,
        cancel:         cancel,
    }
}

// Start 启动关闭管理器
func (sm *ShutdownManager) Start() error {
    // 等待系统信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    select {
    case sig := <-sigChan:
        log.Infof("Received signal: %v", sig)
        return sm.Shutdown()
    case <-sm.ctx.Done():
        return nil
    }
}

// Shutdown 执行关闭流程
func (sm *ShutdownManager) Shutdown() error {
    // Phase 1: Stop accepting
    sm.phase.Store(int32(PhaseStopping))
    log.Info("Phase 1: Stop accepting new requests")
    sm.notifyCallbacks(sm.onStopAccept)
    
    // 等待优雅期
    time.Sleep(sm.gracePeriod)
    
    // Phase 2: Cancel all
    log.Info("Phase 2: Cancel all agent contexts")
    sm.cancel()
    sm.notifyCallbacks(sm.onCancel)
    
    // 等待关闭期
    time.Sleep(sm.gracePeriod)
    
    // Phase 3: Drain queues
    sm.phase.Store(int32(PhaseDraining))
    log.Info("Phase 3: Drain queues")
    if !sm.waitForDrain(sm.shutdownPeriod) {
        log.Warn("Queue drain timeout, forcing shutdown")
    }
    sm.notifyCallbacks(sm.onDrain)
    
    // Phase 4: Save state
    log.Info("Phase 4: Save state")
    sm.notifyCallbacks(sm.onSaveState)
    
    // Phase 5: Close
    sm.phase.Store(int32(PhaseExiting))
    log.Info("Phase 5: Close resources")
    sm.notifyCallbacks(sm.onClose)
    
    return nil
}

// waitForDrain 等待队列清空
func (sm *ShutdownManager) waitForDrain(timeout time.Duration) bool {
    done := make(chan bool, 1)
    go func() {
        sm.wg.Wait()
        done <- true
    }()
    
    select {
    case <-done:
        return true
    case <-time.After(timeout):
        return false
    }
}
```

## 5. Agent 集成

```go
type AgentWithShutdown struct {
    agent  Agent
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}

func (a *AgentWithShutdown) Start(ctx context.Context) error {
    a.ctx, a.cancel = context.WithCancel(ctx)
    // 启动 agent
    return a.agent.Start(a.ctx)
}

func (a *AgentWithShutdown) Stop() error {
    a.cancel()
    <-a.done
    return a.agent.Stop()
}

// 注册到 ShutdownManager
func (sm *ShutdownManager) RegisterAgent(agent *AgentWithShutdown) {
    sm.wg.Add(1)
    go func() {
        defer sm.wg.Done()
        <-agent.ctx.Done()
        // 等待任务完成
        agent.done <- struct{}{}
    }()
}
```

## 6. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| grace_period | 30s | 优雅期时间 |
| shutdown_period | 60s | 关闭期时间 |
| force_timeout | 90s | 强制超时时间 |
| signal_handled | SIGTERM,SIGINT | 处理的信号 |
