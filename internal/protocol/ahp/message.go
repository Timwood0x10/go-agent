package ahp

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"goagent/internal/core/models"
)

// messageIDCounter is used to generate unique message IDs.
var messageIDCounter uint64

// AHPMethod represents the type of AHP message.
type AHPMethod string

const (
	AHPMethodTask      AHPMethod = "TASK"
	AHPMethodResult    AHPMethod = "RESULT"
	AHPMethodProgress  AHPMethod = "PROGRESS"
	AHPMethodACK       AHPMethod = "ACK"
	AHPMethodHeartbeat AHPMethod = "HEARTBEAT"
)

// AHPMessage represents the message structure for Agent communication.
type AHPMessage struct {
	MessageID   string         `json:"message_id"`
	Method      AHPMethod      `json:"method"`
	AgentID     string         `json:"agent_id"`
	TargetAgent string         `json:"target_agent"`
	TaskID      string         `json:"task_id"`
	SessionID   string         `json:"session_id"`
	Payload     map[string]any `json:"payload"`
	Timestamp   time.Time      `json:"timestamp"`
}

// NewMessage creates a new AHPMessage.
func NewMessage(method AHPMethod, agentID, targetAgent, taskID, sessionID string) *AHPMessage {
	return &AHPMessage{
		MessageID:   generateMessageID(),
		Method:      method,
		AgentID:     agentID,
		TargetAgent: targetAgent,
		TaskID:      taskID,
		SessionID:   sessionID,
		Payload:     make(map[string]any),
		Timestamp:   time.Now(),
	}
}

// NewTaskMessage creates a new TASK message.
func NewTaskMessage(agentID, targetAgent, taskID, sessionID string, payload map[string]any) *AHPMessage {
	msg := NewMessage(AHPMethodTask, agentID, targetAgent, taskID, sessionID)
	msg.Payload = payload
	return msg
}

// NewResultMessage creates a new RESULT message.
func NewResultMessage(agentID, targetAgent, taskID, sessionID string, result *models.TaskResult) *AHPMessage {
	msg := NewMessage(AHPMethodResult, agentID, targetAgent, taskID, sessionID)
	msg.Payload = map[string]any{
		"result": result,
	}
	return msg
}

// NewProgressMessage creates a new PROGRESS message.
func NewProgressMessage(agentID, targetAgent, taskID, sessionID string, progress float64) *AHPMessage {
	msg := NewMessage(AHPMethodProgress, agentID, targetAgent, taskID, sessionID)
	msg.Payload = map[string]any{
		"progress": progress,
	}
	return msg
}

// NewACKMessage creates a new ACK message.
func NewACKMessage(agentID, targetAgent, taskID, sessionID string) *AHPMessage {
	return NewMessage(AHPMethodACK, agentID, targetAgent, taskID, sessionID)
}

// NewHeartbeatMessage creates a new HEARTBEAT message.
func NewHeartbeatMessage(agentID string) *AHPMessage {
	return &AHPMessage{
		MessageID: generateMessageID(),
		Method:    AHPMethodHeartbeat,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}
}

// IsTask checks if the message is a TASK message.
func (m *AHPMessage) IsTask() bool {
	return m.Method == AHPMethodTask
}

// IsResult checks if the message is a RESULT message.
func (m *AHPMessage) IsResult() bool {
	return m.Method == AHPMethodResult
}

// IsHeartbeat checks if the message is a HEARTBEAT message.
func (m *AHPMessage) IsHeartbeat() bool {
	return m.Method == AHPMethodHeartbeat
}

// GetResult extracts TaskResult from payload.
func (m *AHPMessage) GetResult() (*models.TaskResult, bool) {
	if m.Method != AHPMethodResult {
		return nil, false
	}
	result, ok := m.Payload["result"].(*models.TaskResult)
	return result, ok
}

// GetProgress extracts progress from payload.
func (m *AHPMessage) GetProgress() (float64, bool) {
	if m.Method != AHPMethodProgress {
		return 0, false
	}
	progress, ok := m.Payload["progress"].(float64)
	return progress, ok
}

func generateMessageID() string {
	id := atomic.AddUint64(&messageIDCounter, 1)
	return fmt.Sprintf("%s.%d", time.Now().Format("20060102150405.000000"), id)
}

// MarshalJSON implements custom JSON marshaling.
func (m *AHPMessage) MarshalJSON() ([]byte, error) {
	type Alias AHPMessage
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}
