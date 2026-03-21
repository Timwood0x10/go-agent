// Package api provides error definitions for API layer.
package api

import "errors"

var (
	// ErrInvalidConfig is returned when config is nil or invalid.
	ErrInvalidConfig = errors.New("invalid config")

	// ErrInitializationFailed is returned when component initialization fails.
	ErrInitializationFailed = errors.New("initialization failed")
)