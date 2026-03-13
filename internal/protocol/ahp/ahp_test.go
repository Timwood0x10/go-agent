package ahp

import (
	"context"
	"testing"
	"time"

	"goagent/internal/core/models"
)

func TestAHPMessage(t *testing.T) {
	t.Run("create message", func(t *testing.T) {
		msg := NewMessage(AHPMethodTask, "agent1", "agent2", "task1", "session1")

		if msg.Method != AHPMethodTask {
			t.Errorf("expected task method, got %s", msg.Method)
		}
		if msg.AgentID != "agent1" {
			t.Errorf("expected agent1, got %s", msg.AgentID)
		}
		if msg.TargetAgent != "agent2" {
			t.Errorf("expected agent2, got %s", msg.TargetAgent)
		}
	})
}

func TestMessageQueue(t *testing.T) {
	t.Run("enqueue and dequeue", func(t *testing.T) {
		queue := NewMessageQueue("agent1", nil)
		msg := NewMessage(AHPMethodTask, "agent1", "agent2", "task1", "session1")

		err := queue.Enqueue(context.Background(), msg)
		if err != nil {
			t.Errorf("enqueue error: %v", err)
		}

		dequeued, err := queue.Dequeue(context.Background())
		if err != nil {
			t.Errorf("dequeue error: %v", err)
		}
		if dequeued == nil {
			t.Errorf("expected message, got nil")
		}
	})

	t.Run("dequeue with timeout", func(t *testing.T) {
		queue := NewMessageQueue("agent1", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		msg, err := queue.Dequeue(ctx)
		if err == nil {
			t.Errorf("expected timeout error")
		}
		if msg != nil {
			t.Errorf("expected nil message")
		}
	})
}

func TestDLQ(t *testing.T) {
	t.Run("add to dlq", func(t *testing.T) {
		dlq := NewDLQ(10)
		msg := NewMessage(AHPMethodTask, "agent1", "agent2", "task1", "session1")

		dlq.Add(msg, nil, "test reason")

		if dlq.Size() != 1 {
			t.Errorf("expected size 1, got %d", dlq.Size())
		}
	})

	t.Run("get all from dlq", func(t *testing.T) {
		dlq := NewDLQ(10)
		msg := NewMessage(AHPMethodTask, "agent1", "agent2", "task1", "session1")

		dlq.Add(msg, nil, "test reason")
		entries := dlq.GetAll()

		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
	})
}

func TestCodec(t *testing.T) {
	t.Run("encode and decode", func(t *testing.T) {
		codec := NewJSONCodec()
		msg := NewMessage(AHPMethodTask, "agent1", "agent2", "task1", "session1")
		msg.Payload = map[string]any{"key": "value"}

		data, err := codec.Encode(msg)
		if err != nil {
			t.Errorf("encode error: %v", err)
		}

		decoded, err := codec.Decode(data)
		if err != nil {
			t.Errorf("decode error: %v", err)
		}
		if decoded.MessageID != msg.MessageID {
			t.Errorf("expected %s, got %s", msg.MessageID, decoded.MessageID)
		}
	})
}

func TestProtocol(t *testing.T) {
	t.Run("create task message", func(t *testing.T) {
		msg := NewTaskMessage("agent1", "agent2", "task1", "session1", nil)

		if msg.Method != AHPMethodTask {
			t.Errorf("expected task method, got %s", msg.Method)
		}
	})

	t.Run("create result message", func(t *testing.T) {
		result := &models.TaskResult{
			TaskID: "task1",
		}
		msg := NewResultMessage("agent1", "agent2", "task1", "session1", result)

		if msg.Method != AHPMethodResult {
			t.Errorf("expected result method, got %s", msg.Method)
		}
	})

	t.Run("create progress message", func(t *testing.T) {
		msg := NewProgressMessage("agent1", "agent2", "task1", "session1", 50)

		if msg.Method != AHPMethodProgress {
			t.Errorf("expected progress method, got %s", msg.Method)
		}
	})

	t.Run("create ack message", func(t *testing.T) {
		msg := NewACKMessage("agent1", "agent2", "task1", "session1")

		if msg.Method != AHPMethodACK {
			t.Errorf("expected ack method, got %s", msg.Method)
		}
	})

	t.Run("create heartbeat message", func(t *testing.T) {
		msg := NewHeartbeatMessage("agent1")

		if msg.Method != AHPMethodHeartbeat {
			t.Errorf("expected heartbeat method, got %s", msg.Method)
		}
	})
}

func TestHeartbeatMonitor(t *testing.T) {
	t.Run("create heartbeat monitor", func(t *testing.T) {
		monitor := NewHeartbeatMonitor(nil)

		if monitor == nil {
			t.Errorf("expected monitor, got nil")
		}
	})

	t.Run("list agents", func(t *testing.T) {
		monitor := NewHeartbeatMonitor(nil)
		monitor.RecordHeartbeat("agent1")

		agents := monitor.ListAgents()
		if len(agents) != 1 {
			t.Errorf("expected 1 agent, got %d", len(agents))
		}
	})
}
