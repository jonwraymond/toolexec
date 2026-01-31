// Package firecracker provides a backend that executes code in Firecracker microVMs.
// Provides strongest isolation; higher complexity and operational cost.
// Appropriate for high-risk multi-tenant execution.
package firecracker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Firecracker backend operations.
var (
	// ErrFirecrackerNotAvailable is returned when Firecracker is not available.
	ErrFirecrackerNotAvailable = errors.New("firecracker not available")

	// ErrMicroVMCreationFailed is returned when microVM creation fails.
	ErrMicroVMCreationFailed = errors.New("microvm creation failed")

	// ErrMicroVMExecutionFailed is returned when microVM execution fails.
	ErrMicroVMExecutionFailed = errors.New("microvm execution failed")
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

// Config configures a Firecracker backend.
type Config struct {
	// BinaryPath is the path to the firecracker binary.
	// Default: firecracker (uses PATH)
	BinaryPath string

	// KernelPath is the path to the guest kernel.
	// Required for execution.
	KernelPath string

	// RootfsPath is the path to the root filesystem image.
	// Required for execution.
	RootfsPath string

	// SocketPath is the path for the Firecracker API socket.
	// Default: auto-generated per VM
	SocketPath string

	// VCPUCount is the number of virtual CPUs.
	// Default: 1
	VCPUCount int

	// MemSizeMB is the memory size in megabytes.
	// Default: 128
	MemSizeMB int

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code in Firecracker microVMs.
type Backend struct {
	binaryPath string
	kernelPath string
	rootfsPath string
	socketPath string
	vcpuCount  int
	memSizeMB  int
	logger     Logger
}

// New creates a new Firecracker backend with the given configuration.
func New(cfg Config) *Backend {
	binaryPath := cfg.BinaryPath
	if binaryPath == "" {
		binaryPath = "firecracker"
	}

	vcpuCount := cfg.VCPUCount
	if vcpuCount <= 0 {
		vcpuCount = 1
	}

	memSizeMB := cfg.MemSizeMB
	if memSizeMB <= 0 {
		memSizeMB = 128
	}

	return &Backend{
		binaryPath: binaryPath,
		kernelPath: cfg.KernelPath,
		rootfsPath: cfg.RootfsPath,
		socketPath: cfg.SocketPath,
		vcpuCount:  vcpuCount,
		memSizeMB:  memSizeMB,
		logger:     cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendFirecracker
}

// Execute runs code in a Firecracker microVM.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := ctx.Err(); err != nil {
		return runtime.ExecuteResult{}, err
	}
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	start := time.Now()

	result := runtime.ExecuteResult{
		Duration: time.Since(start),
		Backend: runtime.BackendInfo{
			Kind: runtime.BackendFirecracker,
			Details: map[string]any{
				"vcpuCount": b.vcpuCount,
				"memSizeMB": b.memSizeMB,
			},
		},
	}

	return result, fmt.Errorf("%w: firecracker backend not fully implemented", ErrFirecrackerNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
