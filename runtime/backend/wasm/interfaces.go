package wasm

import "context"

// ModuleLoader compiles and caches WASM modules.
// This is an optional interface - backends may compile on-demand.
type ModuleLoader interface {
	// Load compiles a WASM binary into a reusable module.
	// The returned module reference can be used in Spec.Module.
	Load(ctx context.Context, binary []byte) (CompiledModule, error)

	// Close releases all cached modules.
	Close(ctx context.Context) error
}

// CompiledModule represents a pre-compiled WASM module.
type CompiledModule interface {
	// Name returns the module name if available.
	Name() string

	// Exports lists exported functions.
	Exports() []string

	// Close releases the compiled module.
	Close(ctx context.Context) error
}

// HealthChecker verifies WASM runtime availability.
// This is an optional interface - backends may skip health checks.
type HealthChecker interface {
	// Ping checks if the WASM runtime is operational.
	Ping(ctx context.Context) error

	// Info returns runtime information.
	Info(ctx context.Context) (RuntimeInfo, error)
}

// StreamRunner provides streaming execution for long-running modules.
// This is an optional extension to Runner.
type StreamRunner interface {
	Runner

	// RunStream executes and streams stdout/stderr as events.
	// The returned channel is closed when execution completes.
	// Callers should drain the channel to receive the exit event.
	RunStream(ctx context.Context, spec Spec) (<-chan StreamEvent, error)
}
