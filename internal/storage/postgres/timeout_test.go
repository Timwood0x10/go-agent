package postgres

import (
	"context"
	"testing"
	"time"
)

func TestWithQueryTimeout(t *testing.T) {
	ctx, cancel := WithQueryTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected query timeout to be set")
	}
}

func TestWithInsertTimeout(t *testing.T) {
	ctx, cancel := WithInsertTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected insert timeout to be set")
	}
}

func TestWithUpdateTimeout(t *testing.T) {
	ctx, cancel := WithUpdateTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected update timeout to be set")
	}
}

func TestWithDeleteTimeout(t *testing.T) {
	ctx, cancel := WithDeleteTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected delete timeout to be set")
	}
}

func TestWithTransactionTimeout(t *testing.T) {
	ctx, cancel := WithTransactionTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected transaction timeout to be set")
	}
}

func TestWithVectorSearchTimeout(t *testing.T) {
	ctx, cancel := WithVectorSearchTimeout(context.Background())
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected vector search timeout to be set")
	}
}

func TestTimeoutRespectsExistingDeadline(t *testing.T) {
	originalCtx, originalCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer originalCancel()

	ctx, cancel := WithQueryTimeout(originalCtx)
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