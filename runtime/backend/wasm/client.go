package wasm

import "context"

// Runner is the primary interface for WASM module execution.
// Implementations may use wazero, wasmer, wasmtime, or mocks.
//
// The interface is intentionally minimal following Go best practices:
// "The bigger the interface, the weaker the abstraction."
//
// Implementations are expected to:
//   - Compile/load the WASM module
//   - Configure memory limits and WASI
//   - Execute the module with provided input
//   - Capture stdout/stderr
//   - Respect context cancellation and spec timeout
type Runner interface {
	// Run executes code in a WASM sandbox and returns the result.
	// The module lifecycle (compile, instantiate, execute, cleanup) is atomic.
	//
	// The implementation must:
	//   - Validate the spec before execution
	//   - Respect ctx cancellation
	//   - Respect spec.Timeout if set
	//   - Capture stdout and stderr
	//   - Return the exit code
	//   - Clean up the module instance on completion or error
	Run(ctx context.Context, spec Spec) (Result, error)
}
