package errors

import (
	"errors"
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
			name:  "ErrInvalidArgument",
			error: ErrInvalidArgument,
			want:  "invalid argument",
		},
		{
			name:  "ErrNotFound",
			error: ErrNotFound,
			want:  "resource not found",
		},
		{
			name:  "ErrAlreadyExists",
			error: ErrAlreadyExists,
			want:  "resource already exists",
		},
		{
			name:  "ErrAccessDenied",
			error: ErrAccessDenied,
			want:  "access denied",
		},
		{
			name:  "ErrTimeout",
			error: ErrTimeout,
			want:  "operation timeout",
		},
		{
			name:  "ErrInternal",
			error: ErrInternal,
			want:  "internal server error",
		},
		{
			name:  "ErrNotImplemented",
			error: ErrNotImplemented,
			want:  "feature not implemented",
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
		ErrInvalidArgument,
		ErrNotFound,
		ErrAlreadyExists,
		ErrAccessDenied,
		ErrTimeout,
		ErrInternal,
		ErrNotImplemented,
	}

	for _, err := range errorVars {
		if err == nil {
			t.Errorf("error variable should not be nil")
		}
	}
}

// TestAppError_Error tests the Error method of AppError.
func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name string
		app  *AppError
		want string
	}{
		{
			name: "error with underlying error",
			app: &AppError{
				Code:    "TEST_CODE",
				Message: "test message",
				Err:     errors.New("underlying error"),
			},
			want: "TEST_CODE: test message: underlying error",
		},
		{
			name: "error without underlying error",
			app: &AppError{
				Code:    "TEST_CODE",
				Message: "test message",
				Err:     nil,
			},
			want: "TEST_CODE: test message",
		},
		{
			name: "error with empty fields",
			app: &AppError{
				Code:    "",
				Message: "",
				Err:     nil,
			},
			want: ": ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.app.Error()
			if got != tt.want {
				t.Errorf("AppError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAppError_Unwrap tests the Unwrap method of AppError.
func TestAppError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	appErr := &AppError{
		Code:    "TEST_CODE",
		Message: "test message",
		Err:     underlyingErr,
	}

	unwrapped := appErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("AppError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test with nil underlying error
	appErrNil := &AppError{
		Code:    "TEST_CODE",
		Message: "test message",
		Err:     nil,
	}

	unwrappedNil := appErrNil.Unwrap()
	if unwrappedNil != nil {
		t.Errorf("AppError.Unwrap() with nil Err = %v, want nil", unwrappedNil)
	}
}

// TestNewAppError tests the NewAppError function.
func TestNewAppError(t *testing.T) {
	underlyingErr := errors.New("underlying error")

	tests := []struct {
		name    string
		code    string
		message string
		err     error
	}{
		{
			name:    "create with all fields",
			code:    "ERR_001",
			message: "test error",
			err:     underlyingErr,
		},
		{
			name:    "create without underlying error",
			code:    "ERR_002",
			message: "test error without underlying",
			err:     nil,
		},
		{
			name:    "create with empty strings",
			code:    "",
			message: "",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppError(tt.code, tt.message, tt.err)

			if appErr.Code != tt.code {
				t.Errorf("NewAppError().Code = %q, want %q", appErr.Code, tt.code)
			}
			if appErr.Message != tt.message {
				t.Errorf("NewAppError().Message = %q, want %q", appErr.Message, tt.message)
			}
			if appErr.Err != tt.err {
				t.Errorf("NewAppError().Err = %v, want %v", appErr.Err, tt.err)
			}
			if appErr.Context == nil {
				t.Errorf("NewAppError().Context should be initialized, got nil")
			}
		})
	}
}

// TestAppError_WithContext tests the WithContext method of AppError.
func TestAppError_WithContext(t *testing.T) {
	appErr := NewAppError("ERR_001", "test error", nil)

	// Test adding context
	result := appErr.WithContext("key1", "value1")
	if result != appErr {
		t.Error("WithContext should return the same AppError instance for chaining")
	}

	if result.Context["key1"] != "value1" {
		t.Errorf("WithContext() context value = %v, want %v", result.Context["key1"], "value1")
	}

	// Test adding multiple contexts
	_ = result.WithContext("key2", 123).WithContext("key3", true)
	if result.Context["key2"] != 123 {
		t.Errorf("WithContext() second context value = %v, want %v", result.Context["key2"], 123)
	}
	if result.Context["key3"] != true {
		t.Errorf("WithContext() third context value = %v, want %v", result.Context["key3"], true)
	}

	// Test with nil context map
	appErrNilContext := &AppError{
		Code:    "ERR_002",
		Message: "test error",
		Err:     nil,
		Context: nil,
	}

	resultNil := appErrNilContext.WithContext("key", "value")
	if resultNil.Context == nil {
		t.Error("WithContext should initialize context map if nil")
	}
	if resultNil.Context["key"] != "value" {
		t.Errorf("WithContext() with nil context = %v, want %v", resultNil.Context["key"], "value")
	}
}

// TestWrap tests the Wrap function.
func TestWrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
		want    string
	}{
		{
			name:    "wrap existing error",
			err:     errors.New("base error"),
			message: "additional context",
			want:    "additional context: base error",
		},
		{
			name:    "wrap error with empty message",
			err:     errors.New("base error"),
			message: "",
			want:    ": base error",
		},
		{
			name:    "wrap nil error",
			err:     nil,
			message: "context",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.message)

			if tt.err == nil {
				if result != nil {
					t.Errorf("Wrap(nil, message) = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Error("Wrap(non-nil, message) should not return nil")
				}
				if result.Error() != tt.want {
					t.Errorf("Wrap() error message = %q, want %q", result.Error(), tt.want)
				}

				// Test unwrapping
				unwrapped := errors.Unwrap(result)
				if unwrapped != tt.err {
					t.Errorf("Wrap() unwrapped error = %v, want %v", unwrapped, tt.err)
				}
			}
		})
	}
}

// TestAppError_ErrorChain tests error chain functionality.
func TestAppError_ErrorChain(t *testing.T) {
	baseErr := errors.New("base error")
	middleErr := Wrap(baseErr, "middle layer")
	topErr := Wrap(middleErr, "top layer")

	// Verify error chain
	if !errors.Is(topErr, baseErr) {
		t.Error("errors.Is should find base error in chain")
	}

	// Unwrap step by step
	layer1 := errors.Unwrap(topErr)
	if layer1 == nil {
		t.Error("First unwrap should return middle error")
	}

	layer2 := errors.Unwrap(layer1)
	if layer2 == nil {
		t.Error("Second unwrap should return base error")
	}

	if layer2 != baseErr {
		t.Errorf("Final unwrapped error = %v, want %v", layer2, baseErr)
	}
}
