// Package kata provides a backend that executes code in Kata Containers.
// Provides VM-level isolation stronger than plain containers.
package kata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Kata backend operations.
var (
	// ErrKataNotAvailable is returned when Kata Containers is not available.
	ErrKataNotAvailable = errors.New("kata containers not available")

	// ErrVMCreationFailed is returned when VM creation fails.
	ErrVMCreationFailed = errors.New("vm creation failed")

	// ErrVMExecutionFailed is returned when VM execution fails.
	ErrVMExecutionFailed = errors.New("vm execution failed")
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

// Config configures a Kata backend.
type Config struct {
	// RuntimePath is the path to the kata-runtime binary.
	// Default: kata-runtime (uses PATH)
	RuntimePath string

	// Hypervisor specifies the hypervisor to use.
	// Options: qemu, cloud-hypervisor, firecracker
	// Default: qemu
	Hypervisor string

	// KernelPath is the path to the guest kernel.
	KernelPath string

	// ImagePath is the path to the guest image/rootfs.
	ImagePath string

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code in Kata Containers for VM-level isolation.
type Backend struct {
	runtimePath string
	hypervisor  string
	kernelPath  string
	imagePath   string
	logger      Logger
}

// New creates a new Kata backend with the given configuration.
func New(cfg Config) *Backend {
	runtimePath := cfg.RuntimePath
	if runtimePath == "" {
		runtimePath = "kata-runtime"
	}

	hypervisor := cfg.Hypervisor
	if hypervisor == "" {
		hypervisor = "qemu"
	}

	return &Backend{
		runtimePath: runtimePath,
		hypervisor:  hypervisor,
		kernelPath:  cfg.KernelPath,
		imagePath:   cfg.ImagePath,
		logger:      cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendKata
}

// Execute runs code in a Kata Container with VM-level isolation.
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
			Kind: runtime.BackendKata,
			Details: map[string]any{
				"hypervisor": b.hypervisor,
			},
		},
	}

	return result, fmt.Errorf("%w: kata backend not fully implemented", ErrKataNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
