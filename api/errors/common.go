// Package errors provides unified error definitions for API layer.
package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig is returned when config is nil or invalid.
	ErrInvalidConfig = errors.New("invalid config")

	// ErrInvalidArgument is returned when an argument is invalid.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrAccessDenied is returned when access to a resource is denied.
	ErrAccessDenied = errors.New("access denied")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timeout")

	// ErrInternal is returned for internal server errors.
	ErrInternal = errors.New("internal server error")

	// ErrNotImplemented is returned when a feature is not implemented.
	ErrNotImplemented = errors.New("feature not implemented")
)

// AppError represents an application error with additional context.
type AppError struct {
	// Code is the error code.
	Code string
	// Message is the error message.
	Message string
	// Err is the underlying error.
	Err error
	// Context is additional context.
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error.
// Args:
// code - error code.
// message - error message.
// err - underlying error.
// Returns new application error.
func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error.
// Args:
// key - context key.
// value - context value.
// Returns the error for chaining.
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Wrap wraps an error with additional context.
// Args:
// err - the error to wrap.
// message - additional message.
// Returns wrapped error.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}