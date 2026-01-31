package code

import "context"

// Engine is the pluggable code execution engine that runs code snippets
// with access to the Tools environment. Implementations are responsible
// for parsing and executing the code in the specified language.
//
// The Engine should:
//   - Execute the code with access to the provided Tools
//   - Capture the final result (typically via __out variable convention)
//   - Return any stdout/stderr captured during execution
//   - Wrap execution errors in CodeError with line/column info when available
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines and return ctx.Err() when canceled.
// - Errors: execution failures should return CodeError where possible; callers use errors.Is.
// - Ownership: params and Tools are read-only; returned ExecuteResult is caller-owned.
type Engine interface {
	// Execute runs a code snippet with access to the tools environment.
	// It returns the execution result including the final value, output,
	// and any errors encountered.
	Execute(ctx context.Context, params ExecuteParams, tools Tools) (ExecuteResult, error)
}
