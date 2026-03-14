package shutdown

import (
	"context"
	"testing"
	"time"
)

func TestShutdownManager(t *testing.T) {
	t.Run("create manager", func(t *testing.T) {
		manager := NewManager(10 * time.Second)

		if manager == nil {
			t.Errorf("manager should not be nil")
		}
		if manager.timeout != 10*time.Second {
			t.Errorf("expected 10s timeout")
		}
	})

	t.Run("register phase", func(t *testing.T) {
		manager := NewManager(10 * time.Second)
		manager.RegisterPhase(PhaseGraceful, 5*time.Second)

		if manager.CurrentPhase() != 0 {
			t.Errorf("phase should start at 0")
		}
	})

	t.Run("add callback", func(t *testing.T) {
		manager := NewManager(10 * time.Second)
		manager.RegisterPhase(PhaseGraceful, 5*time.Second)

		err := manager.AddCallback(PhaseGraceful, func(ctx context.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("add callback error: %v", err)
		}
	})

	t.Run("is shutdown", func(t *testing.T) {
		manager := NewManager(10 * time.Second)

		if manager.IsShutdown() {
			t.Errorf("should not be shutdown initially")
		}
	})
}

func TestPhase(t *testing.T) {
	t.Run("phase constants", func(t *testing.T) {
		if PhasePreShutdown.String() != "pre-shutdown" {
			t.Errorf("unexpected pre-shutdown phase")
		}
		if PhaseGraceful.String() != "graceful" {
			t.Errorf("unexpected graceful phase")
		}
		if PhaseDone.String() != "done" {
			t.Errorf("unexpected done phase")
		}
	})

	t.Run("is valid", func(t *testing.T) {
		if !PhaseGraceful.IsValid() {
			t.Errorf("graceful should be valid")
		}
		if Phase(100).IsValid() {
			t.Errorf("invalid phase should not be valid")
		}
	})
}

func TestPhaseExecutor(t *testing.T) {
	t.Run("create executor", func(t *testing.T) {
		executor := NewPhaseExecutor(PhaseGraceful, 3)

		if executor.Phase() != PhaseGraceful {
			t.Errorf("expected graceful phase")
		}
		if executor.Retries() != 0 {
			t.Errorf("expected 0 retries")
		}
	})

	t.Run("state", func(t *testing.T) {
		executor := NewPhaseExecutor(PhaseGraceful, 3)

		if executor.State() != PhaseStatePending {
			t.Errorf("expected pending state")
		}
	})
}
