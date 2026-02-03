// Package firecracker provides a backend that executes code in Firecracker microVMs.
// Provides strongest isolation; higher complexity and operational cost.
// Appropriate for high-risk multi-tenant execution.
package firecracker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	// ErrClientNotConfigured is returned when no MicroVMRunner is configured.
	ErrClientNotConfigured = errors.New("firecracker runner not configured")

	// ErrDaemonUnavailable is returned when Firecracker is not reachable.
	ErrDaemonUnavailable = errors.New("firecracker daemon unavailable")
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

	// Image is the container image to use for execution when supported.
	// Default: toolruntime-sandbox:latest
	Image string

	// Client executes microVM specs.
	// If nil, Execute() returns ErrClientNotConfigured.
	Client MicroVMRunner

	// HealthChecker optionally verifies Firecracker availability.
	HealthChecker HealthChecker

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
	image      string
	client     MicroVMRunner
	health     HealthChecker
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

	image := cfg.Image
	if image == "" {
		image = "toolruntime-sandbox:latest"
	}

	return &Backend{
		binaryPath: binaryPath,
		kernelPath: cfg.KernelPath,
		rootfsPath: cfg.RootfsPath,
		socketPath: cfg.SocketPath,
		vcpuCount:  vcpuCount,
		memSizeMB:  memSizeMB,
		image:      image,
		client:     cfg.Client,
		health:     cfg.HealthChecker,
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

	if b.client == nil {
		return runtime.ExecuteResult{}, ErrClientNotConfigured
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	if b.health != nil {
		if err := b.health.Ping(ctx); err != nil {
			return runtime.ExecuteResult{}, fmt.Errorf("%w: %v", ErrDaemonUnavailable, err)
		}
	}

	profile := req.Profile
	if profile == "" {
		profile = runtime.ProfileStandard
	}

	spec, err := b.buildSpec(req)
	if err != nil {
		return runtime.ExecuteResult{}, err
	}

	if b.logger != nil {
		b.logger.Info("executing in firecracker",
			"profile", profile,
			"kernelPath", b.kernelPath,
			"rootfsPath", b.rootfsPath)
	}

	runResult, err := b.client.Run(ctx, spec)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(profile),
		}, err
	}

	return runtime.ExecuteResult{
		Value:    extractOutValue(runResult.Stdout),
		Stdout:   runResult.Stdout,
		Stderr:   runResult.Stderr,
		Duration: runResult.Duration,
		Backend:  b.backendInfo(profile),
		LimitsEnforced: runtime.LimitsEnforced{
			Timeout:    true,
			Memory:     req.Limits.MemoryBytes > 0,
			CPU:        req.Limits.CPUQuotaMillis > 0,
			Pids:       req.Limits.PidsMax > 0,
			Disk:       req.Limits.DiskBytes > 0,
			ToolCalls:  true,
			ChainSteps: true,
		},
	}, nil
}

var _ runtime.Backend = (*Backend)(nil)

func (b *Backend) backendInfo(profile runtime.SecurityProfile) runtime.BackendInfo {
	return runtime.BackendInfo{
		Kind:      runtime.BackendFirecracker,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"vcpuCount": b.vcpuCount,
			"memSizeMB": b.memSizeMB,
			"profile":   string(profile),
		},
	}
}

func (b *Backend) buildSpec(req runtime.ExecuteRequest) (MicroVMSpec, error) {
	spec := MicroVMSpec{
		Image:      b.image,
		Resources:  VMResourceSpec{VCPUCount: b.vcpuCount, MemSizeMB: b.memSizeMB},
		Config:     VMConfig{KernelPath: b.kernelPath, RootfsPath: b.rootfsPath, SocketPath: b.socketPath},
		Timeout:    req.Timeout,
		Labels:     map[string]string{"runtime.backend": string(runtime.BackendFirecracker)},
	}
	if err := spec.Validate(); err != nil {
		return MicroVMSpec{}, err
	}
	return spec, nil
}

// extractOutValue extracts the __out value from stdout if present.
func extractOutValue(stdout string) any {
	lines := strings.Split(stdout, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "__OUT__:") {
			jsonStr := strings.TrimPrefix(line, "__OUT__:")
			var value any
			if err := json.Unmarshal([]byte(jsonStr), &value); err == nil {
				return value
			}
			return jsonStr
		}
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var payload map[string]any
			if err := json.Unmarshal([]byte(line), &payload); err == nil {
				if value, ok := payload["__out"]; ok {
					return value
				}
			}
		}
	}
	return nil
}
