package exec

import (
	"context"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
)

// Handler is the function signature for local tool handlers.
// It matches run.LocalHandler and local.HandlerFunc for compatibility.
type Handler func(ctx context.Context, args map[string]any) (any, error)

// Result represents the outcome of a single tool execution.
type Result struct {
	// Value is the return value from the tool.
	Value any

	// ToolID is the canonical ID of the executed tool.
	ToolID string

	// Duration is how long the tool took to execute.
	Duration time.Duration

	// Error is non-nil if the tool execution failed.
	// This is set when the tool itself returns an error,
	// not for resolution or validation errors (which are
	// returned from RunTool directly).
	Error error
}

// OK returns true if the result has no error.
func (r Result) OK() bool {
	return r.Error == nil
}

// StepResult represents the outcome of a single step in a chain execution.
type StepResult struct {
	// StepIndex is the zero-based index of this step in the chain.
	StepIndex int

	// ToolID is the canonical ID of the executed tool.
	ToolID string

	// Args are the arguments passed to this step.
	Args map[string]any

	// Value is the return value from this step.
	Value any

	// Duration is how long this step took to execute.
	Duration time.Duration

	// Error is non-nil if this step failed.
	Error error

	// Skipped is true if this step was skipped due to a prior failure.
	Skipped bool
}

// OK returns true if the step completed successfully.
func (s StepResult) OK() bool {
	return s.Error == nil && !s.Skipped
}

// CodeResult represents the outcome of code execution with tool access.
type CodeResult struct {
	// Value is the final return value from the code.
	Value any

	// ToolCalls contains information about each tool call made during execution.
	ToolCalls []ToolCall

	// Duration is the total execution time.
	Duration time.Duration

	// Stdout contains captured standard output.
	Stdout string

	// Stderr contains captured standard error.
	Stderr string

	// Error is non-nil if execution failed.
	Error error
}

// OK returns true if code execution succeeded.
func (c CodeResult) OK() bool {
	return c.Error == nil
}

// ToolCall represents a tool invocation made during code execution.
type ToolCall struct {
	// ToolID is the canonical ID of the called tool.
	ToolID string

	// Args are the arguments passed to the tool.
	Args map[string]any

	// Result is the tool's return value.
	Result any

	// Duration is how long the tool call took.
	Duration time.Duration

	// Error is non-nil if the tool call failed.
	Error error
}

// ToolSummary is an alias to index.Summary for search results.
type ToolSummary = index.Summary
