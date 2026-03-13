# Shutdown Design Document

## 1. Overview

The Shutdown module is responsible for graceful system shutdown, ensuring that when the service stops it can:
- Stop accepting new requests
- Process current tasks
- Save system state
- Clean up resources

## 2. Shutdown Phases

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Shutdown Signal Flow                               │
└─────────────────────────────────────────────────────────────────────────┘

                       SIGTERM / SIGINT
                              │
                              ▼
                 ┌────────────────────────┐
                 │  Phase 1: Stop Accept  │
                 │  Stop accepting new    │
                 └───────────┬────────────┘
                             │
                             │ Wait 30s (grace period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 2: Cancel       │
                 │  Cancel all Agent      │
                 │  contexts              │
                 └───────────┬────────────┘
                             │
                             │ Wait 60s (shutdown period)
                             ▼
                 ┌────────────────────────┐
                 │  Phase 3: Drain Queues │
                 │  Process pending       │
                 │  messages              │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 4: Save State   │
                 │  Persist memory state  │
                 └───────────┬────────────┘
                             │
                             ▼
                 ┌────────────────────────┐
                 │  Phase 5: Close        │
                 │  Close DB/Connection  │
                 └───────────┬────────────┘
                             │
                             ▼
                          EXIT 0
```

## 3. Core Structure

```go
type ShutdownManager struct {
    gracePeriod     time.Duration     // 30s, graceful period
    shutdownPeriod  time.Duration     // 60s, shutdown period
    forceTimeout    time.Duration     // 90s, force timeout
    
    phase    atomic.Int32              // Current phase
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    
    // Callbacks
    onStopAccept   []func()           // Stop accept callback
    onCancel        []func()           // Cancel callback
    onDrain        []func()           // Drain callback
    onSaveState    []func()           // Save state callback
    onClose        []func()           // Close callback
}

type Phase int

const (
    PhaseRunning    Phase = 0  // Running
    PhaseStopping   Phase = 1  // Stop accepting
    PhaseDraining   Phase = 2  // Draining
    PhaseExiting    Phase = 3  // Exiting
)
```

## 4. Implementation

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

// Start starts the shutdown manager
func (sm *ShutdownManager) Start() error {
    // Wait for system signal
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

// Shutdown executes the shutdown process
func (sm *ShutdownManager) Shutdown() error {
    // Phase 1: Stop accepting
    sm.phase.Store(int32(PhaseStopping))
    log.Info("Phase 1: Stop accepting new requests")
    sm.notifyCallbacks(sm.onStopAccept)
    
    // Wait for grace period
    time.Sleep(sm.gracePeriod)
    
    // Phase 2: Cancel all
    log.Info("Phase 2: Cancel all agent contexts")
    sm.cancel()
    sm.notifyCallbacks(sm.onCancel)
    
    // Wait for shutdown period
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

// waitForDrain wait for queue drain
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

## 5. Agent Integration

```go
type AgentWithShutdown struct {
    agent  Agent
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}

func (a *AgentWithShutdown) Start(ctx context.Context) error {
    a.ctx, a.cancel = context.WithCancel(ctx)
    // Start agent
    return a.agent.Start(a.ctx)
}

func (a *AgentWithShutdown) Stop() error {
    a.cancel()
    <-a.done
    return a.agent.Stop()
}

// Register to ShutdownManager
func (sm *ShutdownManager) RegisterAgent(agent *AgentWithShutdown) {
    sm.wg.Add(1)
    go func() {
        defer sm.wg.Done()
        <-agent.ctx.Done()
        // Wait for task completion
        agent.done <- struct{}{}
    }()
}
```

## 6. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| grace_period | 30s | Grace period |
| shutdown_period | 60s | Shutdown period |
| force_timeout | 90s | Force timeout |
| signal_handled | SIGTERM,SIGINT | Handled signals |
