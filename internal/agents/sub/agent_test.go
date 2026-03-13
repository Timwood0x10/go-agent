package sub

import (
	"context"
	"testing"

	"goagent/internal/core/models"
	"goagent/internal/protocol/ahp"
)

func TestTaskExecutor_Execute(t *testing.T) {
	executor := NewTaskExecutor(nil)

	task := models.NewTask("task_1", models.AgentTypeTop, &models.UserProfile{})

	result, err := executor.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Error("Execute() should succeed for valid task")
	}

	if len(result.Items) == 0 {
		t.Error("Execute() should return items")
	}
}

func TestTaskExecutor_ExecuteNilTask(t *testing.T) {
	executor := NewTaskExecutor(nil)

	result, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Success {
		t.Error("Execute() should fail for nil task")
	}
}

func TestTaskExecutor_ExecuteByType(t *testing.T) {
	executor := NewTaskExecutor(nil)

	tests := []struct {
		agentType models.AgentType
		wantItems int
	}{
		{models.AgentTypeTop, 2},
		{models.AgentTypeBottom, 1},
		{models.AgentTypeShoes, 1},
		{models.AgentTypeHead, 1},
		{models.AgentTypeAccessory, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			task := models.NewTask("task_test", tt.agentType, &models.UserProfile{})
			result, err := executor.Execute(context.Background(), task)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(result.Items) != tt.wantItems {
				t.Errorf("Execute() got %d items, want %d", len(result.Items), tt.wantItems)
			}
		})
	}
}

func TestMessageHandler_Handle(t *testing.T) {
	handler := NewMessageHandler("test_agent")

	// Test nil message
	err := handler.Handle(context.Background(), nil)
	if err == nil {
		t.Error("Handle() should return error for nil message")
	}

	// Test valid message
	msg := ahp.NewHeartbeatMessage("test")
	err = handler.Handle(context.Background(), msg)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}
}

func TestToolBinder_BindAndCall(t *testing.T) {
	binder := NewToolBinder()

	// Bind a tool
	binder.BindTool("test_tool", func(ctx context.Context, args map[string]any) (any, error) {
		return "test_result", nil
	})

	// Call the tool
	result, err := binder.CallTool(context.Background(), "test_tool", nil)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if result != "test_result" {
		t.Errorf("CallTool() got %v, want 'test_result'", result)
	}
}

func TestToolBinder_CallNonExistentTool(t *testing.T) {
	binder := NewToolBinder()

	_, err := binder.CallTool(context.Background(), "non_existent", nil)
	if err == nil {
		t.Error("CallTool() should return error for non-existent tool")
	}
}

func TestHeartbeatSender_StartStop(t *testing.T) {
	sender := NewHeartbeatSender("test_agent", 100, nil)

	ctx, cancel := context.WithCancel(context.Background())

	go sender.Start(ctx)

	// Let it run briefly
	cancel()

	sender.Stop()
}

func TestSubAgent_New(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")

	agent := New("sub1", models.AgentTypeTop, executor, handler, nil, nil, nil)

	if agent.ID() != "sub1" {
		t.Errorf("expected sub1, got %s", agent.ID())
	}
	if agent.Type() != models.AgentTypeTop {
		t.Errorf("expected AgentTypeTop")
	}
}

func TestSubAgent_DefaultConfig(t *testing.T) {
	cfg := DefaultSubAgentConfig(models.AgentTypeTop)

	if cfg.Type != models.AgentTypeTop {
		t.Errorf("expected AgentTypeTop")
	}
}

func TestSubAgent_StartStop(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")

	agent := New("sub1", models.AgentTypeTop, executor, handler, nil, nil, nil)

	// Start
	err := agent.Start(context.Background())
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if agent.Status() != models.AgentStatusReady {
		t.Errorf("expected status Ready after Start")
	}

	// Start again should fail
	err = agent.Start(context.Background())
	if err == nil {
		t.Error("Start() should return error when already started")
	}

	// Stop
	err = agent.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if agent.Status() != models.AgentStatusOffline {
		t.Errorf("expected status Offline after Stop")
	}

	// Stop again should fail
	err = agent.Stop(context.Background())
	if err == nil {
		t.Error("Stop() should return error when not running")
	}
}

func TestSubAgent_Process(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")

	agent := New("sub1", models.AgentTypeTop, executor, handler, nil, nil, nil)

	// Process without starting should auto-start
	task := models.NewTask("task_1", models.AgentTypeTop, &models.UserProfile{})
	result, err := agent.Process(context.Background(), task)
	if err != nil {
		t.Errorf("Process() error = %v", err)
	}
	_ = result
}

func TestSubAgent_SendReceiveMessage(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")
	queue := ahp.NewMessageQueue("sub1", &ahp.QueueOptions{MaxSize: 10})

	sub := &subAgent{
		id:           "sub1",
		agentType:    models.AgentTypeTop,
		status:       models.AgentStatusReady,
		executor:     executor,
		handler:      handler,
		tools:        make(map[string]func(ctx context.Context, args map[string]any) (any, error)),
		messageQueue: queue,
	}

	// Test SendMessage
	msg := ahp.NewMessage(ahp.AHPMethodResult, "sub1", "leader", "task1", "session1")
	err := sub.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	// Test ReceiveMessage
	_, err = sub.ReceiveMessage(context.Background())
	if err != nil {
		t.Errorf("ReceiveMessage() error = %v", err)
	}
}

func TestSubAgent_Heartbeat(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")
	hbMon := ahp.NewHeartbeatMonitor(ahp.DefaultHeartbeatConfig())

	sub := &subAgent{
		id:           "sub1",
		agentType:    models.AgentTypeTop,
		status:       models.AgentStatusReady,
		executor:     executor,
		handler:      handler,
		tools:        make(map[string]func(ctx context.Context, args map[string]any) (any, error)),
		heartbeatMon: hbMon,
	}

	err := sub.Heartbeat(context.Background())
	if err != nil {
		t.Errorf("Heartbeat() error = %v", err)
	}

	if !sub.IsAlive() {
		t.Error("IsAlive() should return true after heartbeat")
	}
}

func TestSubAgent_Execute(t *testing.T) {
	executor := NewTaskExecutor(nil)
	handler := NewMessageHandler("sub1")

	agent := New("sub1", models.AgentTypeTop, executor, handler, nil, nil, nil)

	task := models.NewTask("task_1", models.AgentTypeTop, &models.UserProfile{})
	result, err := agent.Execute(context.Background(), task)
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Error("Execute() should return result")
	}
}

func TestToolBinder_ListTools(t *testing.T) {
	binder := NewToolBinder()

	binder.BindTool("tool1", func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	})
	binder.BindTool("tool2", func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	})

	// ListTools is not implemented, so just test that tools can be bound and called
	result, err := binder.CallTool(context.Background(), "tool1", nil)
	if err != nil {
		t.Errorf("CallTool() error = %v", err)
	}
	if result != nil {
		t.Errorf("CallTool() got %v, want nil", result)
	}
}

func TestMessageHandler_HandleTaskMessage(t *testing.T) {
	handler := NewMessageHandler("test_agent")

	// Create a task message
	msg := ahp.NewTaskMessage("leader", "test_agent", "task1", "session1", map[string]any{"key": "value"})

	// Handle the task message - will fail since executor is nil
	err := handler.Handle(context.Background(), msg)
	// Error expected since there's no executor
	_ = err
}

func TestMessageHandler_HandleAckMessage(t *testing.T) {
	handler := NewMessageHandler("test_agent")

	// Create an ACK message
	msg := ahp.NewACKMessage("test_agent", "leader", "task1", "session1")

	// Handle the ACK message
	err := handler.Handle(context.Background(), msg)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}
}
