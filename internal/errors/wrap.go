// Package errors provides high-performance error wrapping utilities.

package errors

import (
	"fmt"
	"strings"
)

// Wrap wraps an error with a context message without format string parsing.
// This is more efficient than fmt.Errorf for high-frequency error paths.
//
// Usage:
//
//	return Wrap(err, "operation name")
//	return Wrap(err, "operation name: additional context")
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	if message == "" {
		return err
	}
	return &wrappedError{
		msg: message,
		err: err,
	}
}

// Wrapf wraps an error with a formatted message (use sparingly).
// This should only be used when format string is necessary.
// For simple concatenation, use Wrap instead.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// wrappedError is a lightweight error wrapper.
type wrappedError struct {
	msg string
	err error
}

func (w *wrappedError) Error() string {
	var b strings.Builder
	b.Grow(len(w.msg) + 2 + len(w.err.Error()))
	b.WriteString(w.msg)
	b.WriteString(": ")
	b.WriteString(w.err.Error())
	return b.String()
}

func (w *wrappedError) Unwrap() error {
	return w.err
}
