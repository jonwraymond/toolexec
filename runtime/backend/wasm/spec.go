package wasm

import "time"

// Spec defines what to execute in a WASM sandbox and how.
type Spec struct {
	// Module is the compiled WASM binary (required).
	// This can be raw .wasm bytes or a precompiled module reference.
	Module []byte

	// EntryPoint is the exported function to call (default: "_start" for WASI).
	EntryPoint string

	// Args are command-line arguments passed to the module.
	Args []string

	// Env contains environment variables in KEY=value format.
	Env []string

	// Stdin provides input to the module's stdin.
	Stdin []byte

	// WorkingDir is the working directory for WASI filesystem operations.
	WorkingDir string

	// Mounts defines filesystem mounts for WASI.
	Mounts []Mount

	// Resources defines resource limits.
	Resources ResourceSpec

	// Security defines security settings.
	Security SecuritySpec

	// Timeout is the maximum execution duration.
	Timeout time.Duration

	// Labels are metadata labels for tracking.
	Labels map[string]string
}

// Mount defines a filesystem mount for WASI.
type Mount struct {
	// HostPath is the path on the host filesystem.
	HostPath string

	// GuestPath is the path inside the WASM module.
	GuestPath string

	// ReadOnly mounts the path as read-only.
	ReadOnly bool
}

// ResourceSpec defines WASM resource limits.
type ResourceSpec struct {
	// MemoryPages is the maximum memory in 64KB pages.
	// Zero uses runtime default (typically 256 = 16MB).
	MemoryPages uint32

	// FuelLimit is the maximum fuel for metered execution.
	// Zero means unlimited (no metering).
	FuelLimit uint64

	// StackSize is the maximum call stack size in bytes.
	// Zero uses runtime default.
	StackSize uint32
}

// SecuritySpec defines WASM security settings.
type SecuritySpec struct {
	// EnableWASI enables WebAssembly System Interface.
	// Default: true for most use cases.
	EnableWASI bool

	// AllowedHostFunctions lists host functions the module can import.
	// Empty means no host functions allowed (maximum isolation).
	AllowedHostFunctions []string

	// EnableNetwork allows WASI network access.
	// Default: false for sandbox isolation.
	EnableNetwork bool

	// EnableClock allows access to system clock.
	// Default: true (needed for timing operations).
	EnableClock bool
}

// Result captures the output of WASM execution.
type Result struct {
	// ExitCode is the module's exit code (0 = success).
	ExitCode int

	// Stdout contains the module's stdout output.
	Stdout string

	// Stderr contains the module's stderr output.
	Stderr string

	// Duration is the execution time.
	Duration time.Duration

	// FuelConsumed is the fuel used (if metering enabled).
	FuelConsumed uint64

	// MemoryUsed is peak memory usage in bytes.
	MemoryUsed uint64
}

// StreamEventType identifies the type of streaming event.
type StreamEventType string

const (
	// StreamEventStdout indicates stdout data.
	StreamEventStdout StreamEventType = "stdout"

	// StreamEventStderr indicates stderr data.
	StreamEventStderr StreamEventType = "stderr"

	// StreamEventExit indicates module exit.
	StreamEventExit StreamEventType = "exit"

	// StreamEventError indicates an error occurred.
	StreamEventError StreamEventType = "error"
)

// StreamEvent represents a streaming output event from WASM execution.
type StreamEvent struct {
	// Type identifies the event type.
	Type StreamEventType

	// Data contains the event payload (stdout/stderr bytes).
	Data []byte

	// ExitCode is set when Type is StreamEventExit.
	ExitCode int

	// Error is set when Type is StreamEventError.
	Error error
}

// RuntimeInfo contains WASM runtime metadata.
type RuntimeInfo struct {
	// Name is the runtime name (wazero, wasmer, wasmtime).
	Name string

	// Version is the runtime version.
	Version string

	// Features lists supported features.
	Features []string
}
