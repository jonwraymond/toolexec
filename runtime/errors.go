package runtime

import (
	"errors"
	"fmt"
)

// Sentinel errors for toolruntime operations.
var (
	// ErrRuntimeUnavailable is returned when no suitable runtime/backend is available.
	ErrRuntimeUnavailable = errors.New("runtime unavailable")

	// ErrBackendDenied is returned when a backend is denied by security policy.
	ErrBackendDenied = errors.New("backend denied by policy")

	// ErrSandboxViolation is returned when sandboxed code violates security policy.
	ErrSandboxViolation = errors.New("sandbox policy violation")

	// ErrTimeout is returned when execution exceeds the configured timeout.
	ErrTimeout = errors.New("execution timeout")

	// ErrResourceLimit is returned when a resource limit is exceeded.
	ErrResourceLimit = errors.New("resource limit exceeded")

	// ErrMissingGateway is returned when ExecuteRequest has no Gateway.
	ErrMissingGateway = errors.New("gateway is required")

	// ErrMissingCode is returned when ExecuteRequest has no Code.
	ErrMissingCode = errors.New("code is required")

	// ErrInvalidLimits is returned when Limits validation fails.
	ErrInvalidLimits = errors.New("invalid limits")
)

// RuntimeError wraps an error with execution context information.
// It provides the operation that failed and the backend that was in use.
type RuntimeError struct {
	// Err is the underlying error.
	Err error

	// Op is the operation that failed (e.g., "execute", "container_create").
	Op string

	// Backend is the backend kind that was in use when the error occurred.
	Backend BackendKind

	// Retryable indicates whether the operation can be retried.
	// True for transient errors like timeouts; false for policy violations.
	Retryable bool
}

// Error returns the error message with context.
func (e *RuntimeError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s: unknown error", e.Backend, e.Op)
	}
	return fmt.Sprintf("%s: %s: %v", e.Backend, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As.
func (e *RuntimeError) Unwrap() error {
	return e.Err
}
