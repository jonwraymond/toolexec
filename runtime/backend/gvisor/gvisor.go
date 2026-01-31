// Package gvisor provides a backend that executes code with gVisor (runsc).
// Provides stronger isolation than plain containers; appropriate for untrusted multi-tenant execution.
package gvisor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for gVisor backend operations.
var (
	// ErrGVisorNotAvailable is returned when gVisor/runsc is not available.
	ErrGVisorNotAvailable = errors.New("gvisor not available")

	// ErrSandboxCreationFailed is returned when sandbox creation fails.
	ErrSandboxCreationFailed = errors.New("sandbox creation failed")

	// ErrSandboxExecutionFailed is returned when sandbox execution fails.
	ErrSandboxExecutionFailed = errors.New("sandbox execution failed")
)

// Logger is the interface for logging.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Errors: logging must be best-effort and must not panic.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Config configures a gVisor backend.
type Config struct {
	// RunscPath is the path to the runsc binary.
	// Default: runsc (uses PATH)
	RunscPath string

	// RootDir is the root directory for gVisor state.
	// Default: /var/run/gvisor
	RootDir string

	// Platform is the gVisor platform to use.
	// Options: ptrace, kvm, systrap
	// Default: systrap
	Platform string

	// NetworkMode specifies the network configuration.
	// Options: none, sandbox, host
	// Default: none
	NetworkMode string

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code with gVisor for stronger isolation.
type Backend struct {
	runscPath   string
	rootDir     string
	platform    string
	networkMode string
	logger      Logger
}

// New creates a new gVisor backend with the given configuration.
func New(cfg Config) *Backend {
	runscPath := cfg.RunscPath
	if runscPath == "" {
		runscPath = "runsc"
	}

	rootDir := cfg.RootDir
	if rootDir == "" {
		rootDir = "/var/run/gvisor"
	}

	platform := cfg.Platform
	if platform == "" {
		platform = "systrap"
	}

	networkMode := cfg.NetworkMode
	if networkMode == "" {
		networkMode = "none"
	}

	return &Backend{
		runscPath:   runscPath,
		rootDir:     rootDir,
		platform:    platform,
		networkMode: networkMode,
		logger:      cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendGVisor
}

// Execute runs code with gVisor isolation.
func (b *Backend) Execute(_ context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	start := time.Now()

	result := runtime.ExecuteResult{
		Duration: time.Since(start),
		Backend: runtime.BackendInfo{
			Kind: runtime.BackendGVisor,
			Details: map[string]any{
				"platform":    b.platform,
				"networkMode": b.networkMode,
			},
		},
	}

	return result, fmt.Errorf("%w: gvisor backend not fully implemented", ErrGVisorNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
