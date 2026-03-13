package errors

import (
	"errors"
	"testing"
	"time"
)

func TestErrorCode(t *testing.T) {
	t.Run("create error code", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error message", "test", true, 3, time.Second, 500)

		if code.Code != "TEST_ERROR" {
			t.Errorf("expected TEST_ERROR, got %s", code.Code)
		}
		if code.Message != "test error message" {
			t.Errorf("expected test error message, got %s", code.Message)
		}
		if !code.Retry {
			t.Errorf("expected retry to be true")
		}
		if code.RetryMax != 3 {
			t.Errorf("expected retry max 3, got %d", code.RetryMax)
		}
	})
}

func TestAppError(t *testing.T) {
	t.Run("create app error", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error", "test", false, 0, 0, 500)
		err := New(code)

		if err.Code != code {
			t.Errorf("expected error code, got %v", err.Code)
		}
	})

	t.Run("error to string", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error", "test", false, 0, 0, 500)
		err := New(code)

		str := err.Error()
		if str == "" {
			t.Errorf("error string should not be empty")
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		cause := errors.New("original error")
		code := NewErrorCode("TEST_ERROR", "test error", "test", false, 0, 0, 500)
		err := Wrap(cause, code)

		unwrapped := err.Unwrap()
		if unwrapped != cause {
			t.Errorf("expected to unwrap cause error")
		}
	})

	t.Run("with context", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error", "test", false, 0, 0, 500)
		err := New(code).WithContext("key", "value")

		if err.Context == nil {
			t.Errorf("expected context to be set")
		}
		if err.Context["key"] != "value" {
			t.Errorf("expected key value in context")
		}
	})

	t.Run("is retryable", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error", "test", true, 3, time.Second, 500)
		err := New(code)

		if !err.IsRetryable() {
			t.Errorf("expected error to be retryable")
		}
	})

	t.Run("should retry", func(t *testing.T) {
		code := NewErrorCode("TEST_ERROR", "test error", "test", true, 3, time.Second, 500)
		err := New(code)

		if !err.ShouldRetry(1) {
			t.Errorf("expected should retry at attempt 1")
		}
		if err.ShouldRetry(3) {
			t.Errorf("expected should not retry at attempt 3 (exceeds max)")
		}
	})
}

func TestWrap(t *testing.T) {
	t.Run("wrap error", func(t *testing.T) {
		cause := errors.New("original error")
		code := NewErrorCode("TEST_ERROR", "test error", "test", false, 0, 0, 500)
		err := Wrap(cause, code)

		if err.Err != cause {
			t.Errorf("expected to wrap cause error")
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"agent not found", ErrAgentNotFound},
		{"agent timeout", ErrAgentTimeout},
		{"agent panic", ErrAgentPanic},
		{"task queue full", ErrTaskQueueFull},
		{"dependency cycle", ErrDependencyCycle},
		{"invalid message", ErrInvalidMessage},
		{"message timeout", ErrMessageTimeout},
		{"heartbeat missed", ErrHeartbeatMissed},
		{"queue full", ErrQueueFull},
		{"queue empty", ErrQueueEmpty},
		{"db connection failed", ErrDBConnectionFailed},
		{"query failed", ErrQueryFailed},
		{"record not found", ErrRecordNotFound},
		{"llm request failed", ErrLLMRequestFailed},
		{"invalid user id", ErrInvalidUserID},
		{"invalid age", ErrInvalidAge},
		{"invalid budget", ErrInvalidBudget},
		{"invalid input", ErrInvalidInput},
		{"workflow not found", ErrWorkflowNotFound},
		{"rate limit exceeded", ErrRateLimitExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("sentinel error %s should not be nil", tt.name)
			}
		})
	}
}
