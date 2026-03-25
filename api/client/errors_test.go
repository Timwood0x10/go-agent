package client

import (
	"testing"
)

// TestErrorVariables tests that all error variables are properly defined.
func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name  string
		error error
		want  string
	}{
		{
			name:  "ErrInvalidConfig",
			error: ErrInvalidConfig,
			want:  "invalid config",
		},
		{
			name:  "ErrAgentNotConfigured",
			error: ErrAgentNotConfigured,
			want:  "agent service not configured",
		},
		{
			name:  "ErrMemoryNotConfigured",
			error: ErrMemoryNotConfigured,
			want:  "memory service not configured",
		},
		{
			name:  "ErrRetrievalNotConfigured",
			error: ErrRetrievalNotConfigured,
			want:  "retrieval service not configured",
		},
		{
			name:  "ErrLLMNotConfigured",
			error: ErrLLMNotConfigured,
			want:  "LLM service not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("expected error variable to be non-nil")
			}
			if tt.error.Error() != tt.want {
				t.Errorf("error message mismatch: got %q, want %q", tt.error.Error(), tt.want)
			}
		})
	}
}

// TestErrorNilCheck ensures error variables are not nil.
func TestErrorNilCheck(t *testing.T) {
	errorVars := []error{
		ErrInvalidConfig,
		ErrAgentNotConfigured,
		ErrMemoryNotConfigured,
		ErrRetrievalNotConfigured,
		ErrLLMNotConfigured,
	}

	for _, err := range errorVars {
		if err == nil {
			t.Errorf("error variable should not be nil")
		}
	}
}

// TestErrorUniqueness ensures all error messages are unique.
func TestErrorUniqueness(t *testing.T) {
	errorMessages := map[string]bool{
		ErrInvalidConfig.Error():          true,
		ErrAgentNotConfigured.Error():     true,
		ErrMemoryNotConfigured.Error():    true,
		ErrRetrievalNotConfigured.Error(): true,
		ErrLLMNotConfigured.Error():       true,
	}

	if len(errorMessages) != 5 {
		t.Errorf("expected 5 unique error messages, got %d", len(errorMessages))
	}
}
