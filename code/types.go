package code

import "time"

// ToolCallRecord captures information about a single tool invocation during
// code execution. It records the tool identifier, arguments, result, and
// timing information for observability and debugging.
type ToolCallRecord struct {
	// ToolID is the canonical identifier of the tool that was called.
	ToolID string `json:"toolId"`

	// Args contains the arguments passed to the tool.
	Args map[string]any `json:"args,omitempty"`

	// Structured contains the structured result from a successful tool execution.
	Structured any `json:"structured,omitempty"`

	// BackendKind indicates which backend executed the tool (mcp, provider, local).
	BackendKind string `json:"backendKind,omitempty"`

	// Error contains the error message if the tool call failed.
	Error string `json:"error,omitempty"`

	// ErrorOp indicates the operation that failed (e.g., "run", "chain").
	ErrorOp string `json:"errorOp,omitempty"`

	// DurationMs is the execution time in milliseconds.
	DurationMs int64 `json:"durationMs"`
}

// ExecuteParams specifies the parameters for executing a code snippet.
type ExecuteParams struct {
	// Language specifies the programming language of the code snippet.
	// If empty, the executor's default language is used.
	Language string `json:"language"`

	// Code is the source code to execute.
	Code string `json:"code"`

	// Timeout specifies the maximum duration for execution.
	// If zero, the executor's default timeout is used.
	Timeout time.Duration `json:"timeout"`

	// MaxToolCalls limits the number of tool invocations allowed.
	// If zero, the executor's configured limit applies (or unlimited if none).
	MaxToolCalls int `json:"maxToolCalls,omitempty"`
}

// ExecuteResult contains the outcome of executing a code snippet.
type ExecuteResult struct {
	// Value is the final result of the code execution, typically from the
	// __out variable convention.
	Value any `json:"value,omitempty"`

	// Stdout contains any output written via Println or similar.
	Stdout string `json:"stdout,omitempty"`

	// Stderr contains any error output from the execution.
	Stderr string `json:"stderr,omitempty"`

	// ToolCalls records all tool invocations made during execution.
	ToolCalls []ToolCallRecord `json:"toolCalls,omitempty"`

	// DurationMs is the total execution time in milliseconds.
	DurationMs int64 `json:"durationMs"`
}
