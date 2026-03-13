package sub

import (
	"context"
	"time"

	"goagent/internal/protocol/ahp"
)

// heartbeatSender sends heartbeat to leader.
type heartbeatSender struct {
	agentID      string
	interval     time.Duration
	stopCh       chan struct{}
	heartbeatMon *ahp.HeartbeatMonitor
}

// NewHeartbeatSender creates a new HeartbeatSender.
func NewHeartbeatSender(agentID string, interval time.Duration, hbMon *ahp.HeartbeatMonitor) *heartbeatSender {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &heartbeatSender{
		agentID:      agentID,
		interval:     interval,
		stopCh:       make(chan struct{}),
		heartbeatMon: hbMon,
	}
}

// Start starts sending heartbeats.
func (s *heartbeatSender) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if s.heartbeatMon != nil {
				s.heartbeatMon.RecordHeartbeat(s.agentID)
			}
		}
	}
}

// Stop stops sending heartbeats.
func (s *heartbeatSender) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
}
