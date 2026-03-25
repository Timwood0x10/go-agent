// Package errors provides tests for error wrapping utilities.

package errors

import (
	"errors"
	"fmt"
	"testing"
)

var (
	errTest = errors.New("test error")
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		wantErr bool
	}{
		{
			name:    "wrap nil error",
			err:     nil,
			message: "test",
			wantErr: false,
		},
		{
			name:    "wrap with empty message",
			err:     errTest,
			message: "",
			wantErr: true,
		},
		{
			name:    "wrap with message",
			err:     errTest,
			message: "operation",
			wantErr: true,
		},
		{
			name:    "wrap with complex message",
			err:     errTest,
			message: "operation: failed to process",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(tt.err, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				return
			}

			if err.Error() == "" {
				t.Error("Wrap() returned empty error message")
			}

			// Verify that the error can be unwrapped only when message is not empty.
			if tt.message != "" && errors.Unwrap(err) == nil {
				t.Error("Wrap() error cannot be unwrapped")
			}
		})
	}
}

func TestWrap_ErrorMessage(t *testing.T) {
	err := Wrap(errTest, "operation")
	expected := "operation: test error"

	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}
}

func TestWrap_Unwrap(t *testing.T) {
	err := Wrap(errTest, "operation")
	unwrapped := errors.Unwrap(err)

	if unwrapped != errTest {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, errTest)
	}
}

func TestWrap_NilErrorReturnsNil(t *testing.T) {
	err := Wrap(nil, "message")
	if err != nil {
		t.Errorf("Wrap(nil) should return nil, got %v", err)
	}
}

func TestWrap_EmptyMessageReturnsOriginal(t *testing.T) {
	err := Wrap(errTest, "")
	if err != errTest {
		t.Errorf("Wrap(err, '') should return original error, got %v", err)
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		format  string
		args    []any
		wantErr bool
	}{
		{
			name:    "wrapf nil error",
			err:     nil,
			format:  "test %s",
			args:    []any{"arg"},
			wantErr: false,
		},
		{
			name:    "wrapf with format",
			err:     errTest,
			format:  "operation %s",
			args:    []any{"test"},
			wantErr: true,
		},
		{
			name:    "wrapf without args",
			err:     errTest,
			format:  "operation",
			args:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrapf(tt.err, tt.format, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrapf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				return
			}

			if err.Error() == "" {
				t.Error("Wrapf() returned empty error message")
			}

			// Verify that the error can be unwrapped.
			if errors.Unwrap(err) == nil {
				t.Error("Wrapf() error cannot be unwrapped")
			}
		})
	}
}

func TestWrapf_ErrorMessage(t *testing.T) {
	err := Wrapf(errTest, "operation %s", "test")
	expected := "operation test: test error"

	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}
}

func TestWrapf_NilErrorReturnsNil(t *testing.T) {
	err := Wrapf(nil, "message %s", "arg")
	if err != nil {
		t.Errorf("Wrapf(nil) should return nil, got %v", err)
	}
}

// BenchmarkWrap compares performance of Wrap vs fmt.Errorf.
func BenchmarkWrap(b *testing.B) {
	err := errTest
	message := "operation failed"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Wrap(err, message)
	}
}

// BenchmarkFmtErrorfW compares performance of fmt.Errorf with %w.
func BenchmarkFmtErrorfW(b *testing.B) {
	err := errTest
	message := "operation failed"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("%s: %w", message, err)
	}
}

// BenchmarkWrapMultipleWraps tests performance of multiple wraps.
func BenchmarkWrapMultipleWraps(b *testing.B) {
	err := errTest
	message1 := "operation1"
	message2 := "operation2"
	message3 := "operation3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err1 := Wrap(err, message1)
		err2 := Wrap(err1, message2)
		_ = Wrap(err2, message3)
	}
}

// BenchmarkFmtErrorfMultipleWraps tests performance of multiple fmt.Errorf wraps.
func BenchmarkFmtErrorfMultipleWraps(b *testing.B) {
	err := errTest
	message1 := "operation1"
	message2 := "operation2"
	message3 := "operation3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err1 := fmt.Errorf("%s: %w", message1, err)
		err2 := fmt.Errorf("%s: %w", message2, err1)
		_ = fmt.Errorf("%s: %w", message3, err2)
	}
}
