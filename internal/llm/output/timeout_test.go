// nolint: errcheck // Test code may ignore return values
package output

import (
	"context"
	"testing"
	"time"
)

func TestWithDefaultTimeout(t *testing.T) {
	t.Run("No deadline", func(t *testing.T) {
		ctx := context.Background()
		newCtx, cancel := WithDefaultTimeout(ctx, 5*time.Second)
		defer cancel()

		_, hasDeadline := newCtx.Deadline()
		if !hasDeadline {
			t.Error("Expected deadline to be set")
		}
	})

	t.Run("Has deadline preserved", func(t *testing.T) {
		originalCtx, originalCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer originalCancel()

		newCtx, cancel := WithDefaultTimeout(originalCtx, 5*time.Second)
		defer cancel()

		// Both should have deadlines
		deadline, hasDeadline := newCtx.Deadline()
		originalDeadline, _ := originalCtx.Deadline()

		if !hasDeadline {
			t.Error("Expected deadline to be set")
		}

		// Verify original deadline is preserved
		if !deadline.Equal(originalDeadline) {
			t.Errorf("Expected deadline to be preserved, got %v vs %v", deadline, originalDeadline)
		}
	})
}

func TestWithLLMTimeout(t *testing.T) {
	ctx, cancel := WithLLMTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected LLM timeout to be set")
	}
}

func TestWithDatabaseTimeout(t *testing.T) {
	ctx, cancel := WithDatabaseTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected database timeout to be set")
	}
}

func TestTimeoutRespectsExistingDeadline(t *testing.T) {
	originalCtx, originalCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer originalCancel()

	ctx, cancel := WithDefaultTimeout(originalCtx, 5*time.Second)
	defer cancel()

	// Verify original deadline is preserved
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected deadline to be preserved")
	}

	originalDeadline, _ := originalCtx.Deadline()
	if !deadline.Equal(originalDeadline) {
		t.Errorf("Expected deadline to be preserved, got %v vs %v", deadline, originalDeadline)
	}
}

// nolint: errcheck // Test code may ignore return values
// nolint: errcheck // Test code may ignore return values
