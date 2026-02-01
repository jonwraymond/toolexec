package run

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/toolfoundation/model"
)

// StreamEventKind represents streaming and progress events.
type StreamEventKind string

const (
	// StreamEventProgress indicates a progress update.
	StreamEventProgress StreamEventKind = "progress"

	// StreamEventChunk indicates a partial result chunk.
	StreamEventChunk StreamEventKind = "chunk"

	// StreamEventDone indicates streaming has completed successfully.
	StreamEventDone StreamEventKind = "done"

	// StreamEventError indicates an error occurred during streaming.
	StreamEventError StreamEventKind = "error"
)

// StreamEvent is a transport-agnostic streaming envelope.
// It carries streaming events from tool execution including progress updates,
// partial chunks, completion signals, and errors.
type StreamEvent struct {
	// Kind indicates the type of streaming event.
	Kind StreamEventKind `json:"kind"`

	// ToolID is the canonical tool identifier (namespace:name or name).
	ToolID string `json:"toolId,omitempty"`

	// Data contains event-specific payload.
	// For progress events, this might contain percentage or status.
	// For chunk events, this contains partial result data.
	Data any `json:"data,omitempty"`

	// Err is set when Kind is StreamEventError.
	// Not serialized to JSON - callers should extract error information
	// from Data if needed for transmission.
	Err error `json:"-"`
}

// ProgressEvent represents coarse-grained progress during execution.
// Progress and Total are optional; when Total is zero, Progress should be treated
// as a best-effort signal rather than a precise fraction.
type ProgressEvent struct {
	Progress float64 `json:"progress"`
	Total    float64 `json:"total,omitempty"`
	Message  string  `json:"message,omitempty"`
}

// ChainStep defines one step in a sequential chain.
// Chains execute steps in order, with optional data passing between steps.
type ChainStep struct {
	// ToolID is the canonical tool identifier to execute.
	ToolID string `json:"toolId"`

	// Args are the arguments to pass to the tool.
	Args map[string]any `json:"args,omitempty"`

	// UsePrevious, when true, injects the previous step's structured result
	// into args["previous"], overwriting any existing value.
	UsePrevious bool `json:"usePrevious,omitempty"`
}

// StepResult captures what happened at a single chain step.
// It includes both the result and any error that occurred.
type StepResult struct {
	// ToolID is the canonical tool identifier that was executed.
	ToolID string `json:"toolId"`

	// Backend is the backend that was used for execution.
	Backend model.ToolBackend `json:"backend"`

	// Result contains the execution result.
	Result RunResult `json:"result"`

	// Err is set if the step failed.
	// Not serialized to JSON - callers should check this field explicitly.
	Err error `json:"-"`
}

// RunResult is the normalized result of a tool execution.
// Structured is the primary value used for chaining and validation.
type RunResult struct {
	// Tool is the resolved tool definition.
	Tool model.Tool `json:"tool"`

	// Backend is the backend that was used for execution.
	Backend model.ToolBackend `json:"backend"`

	// Structured is the normalized result value.
	// For MCP backends, this is either StructuredContent (preferred)
	// or a best-effort structured value derived from Content.
	// For provider/local backends, this is the executor/handler return value.
	Structured any `json:"structured,omitempty"`

	// MCPResult is the raw MCP CallToolResult when the backend was MCP.
	// Nil for provider and local backends unless they return MCP-native results.
	MCPResult *mcp.CallToolResult `json:"mcpResult,omitempty"`
}
