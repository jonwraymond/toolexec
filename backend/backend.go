package backend

import (
	"context"
	"errors"

	"github.com/jonwraymond/toolfoundation/model"
)

// Common errors for backend operations.
var (
	ErrBackendNotFound    = errors.New("backend not found")
	ErrBackendDisabled    = errors.New("backend disabled")
	ErrToolNotFound       = errors.New("tool not found in backend")
	ErrBackendUnavailable = errors.New("backend unavailable")
)

// Backend defines a source of tools.
// Backends can be local handlers, MCP servers, HTTP APIs, or custom implementations.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods must honor cancellation/deadlines.
// - Errors: use ErrBackendNotFound/ErrBackendDisabled/ErrToolNotFound/ErrBackendUnavailable where applicable.
type Backend interface {
	// Kind returns the backend type (e.g., "local", "mcp", "http").
	Kind() string

	// Name returns the unique instance name for this backend.
	Name() string

	// Enabled returns whether this backend is currently enabled.
	Enabled() bool

	// ListTools returns all tools available from this backend.
	ListTools(ctx context.Context) ([]model.Tool, error)

	// Execute invokes a tool on this backend.
	Execute(ctx context.Context, tool string, args map[string]any) (any, error)

	// Start initializes the backend (connect to remote, start subprocess, etc.).
	Start(ctx context.Context) error

	// Stop gracefully shuts down the backend.
	Stop() error
}

// ConfigurableBackend can be configured from raw bytes (YAML/JSON).
//
// Contract:
// - Configure must validate config and return error on invalid input.
type ConfigurableBackend interface {
	Backend

	Configure(raw []byte) error
}

// StreamingBackend supports streaming responses.
//
// Contract:
// - If ExecuteStream returns nil error, the channel must be non-nil.
type StreamingBackend interface {
	Backend

	ExecuteStream(ctx context.Context, tool string, args map[string]any) (<-chan any, error)
}

// Factory creates backend instances.
type Factory func(name string) (Backend, error)

// Info contains metadata about a backend.
type Info struct {
	Kind        string
	Name        string
	Enabled     bool
	Description string
	Version     string
}
