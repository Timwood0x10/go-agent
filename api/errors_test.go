package api

import (
	"testing"
)

// TestErrorVariables verifies that all error variables are properly defined.
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
			name:  "ErrInitializationFailed",
			error: ErrInitializationFailed,
			want:  "initialization failed",
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
	if ErrInvalidConfig == nil {
		t.Error("ErrInvalidConfig should not be nil")
	}
	if ErrInitializationFailed == nil {
		t.Error("ErrInitializationFailed should not be nil")
	}
}
