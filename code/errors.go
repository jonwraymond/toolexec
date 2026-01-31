package code

import (
	"errors"
	"fmt"
)

// Sentinel errors for error classification.
var (
	// ErrCodeExecution indicates an error during code snippet execution,
	// such as syntax errors or runtime exceptions in the snippet.
	ErrCodeExecution = errors.New("code execution error")

	// ErrConfiguration indicates an invalid or incomplete configuration.
	ErrConfiguration = errors.New("configuration error")

	// ErrLimitExceeded indicates that an execution limit was reached,
	// such as timeout or maximum tool calls.
	ErrLimitExceeded = errors.New("limit exceeded")
)

// CodeError represents an error that occurred during code snippet execution.
// It includes optional source location information for debugging.
type CodeError struct {
	// Message describes the error.
	Message string

	// Line is the 1-based line number where the error occurred.
	// Zero indicates the line is unknown.
	Line int

	// Column is the 1-based column number where the error occurred.
	// Zero indicates the column is unknown.
	Column int

	// Err is the underlying error, if any.
	Err error
}

// Error returns the error message, including line and column if available.
func (e *CodeError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s (line %d, col %d)", e.Message, e.Line, e.Column)
	}
	return e.Message
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *CodeError) Unwrap() error {
	return e.Err
}

// Is reports whether this error matches the target.
// CodeError matches ErrCodeExecution to allow sentinel-style error checking.
func (e *CodeError) Is(target error) bool {
	return target == ErrCodeExecution
}
