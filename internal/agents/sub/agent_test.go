package sub

import (
	"context"
	"testing"

	"styleagent/internal/core/models"
	"styleagent/internal/protocol/ahp"
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
