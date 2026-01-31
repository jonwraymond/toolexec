package runtime

import "context"

// Backend is the interface for code execution backends.
// Each backend provides a different level of isolation and security.
//
// Contract:
//   - Concurrency: implementations must be safe for concurrent use.
//   - Context: must honor cancellation/deadlines and return ctx.Err() when canceled.
//   - Errors: validation errors should use ErrInvalidRequest; runtime errors should
//     return ErrExecutionFailed (see errors.go) where applicable.
//   - Ownership: requests are read-only; results are caller-owned snapshots.
type Backend interface {
	// Kind returns the backend kind identifier.
	Kind() BackendKind

	// Execute runs code with the given request parameters.
	// It validates the request, executes the code, and returns the result.
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}
