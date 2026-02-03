// Package containerd provides a backend that executes code via containerd.
// Similar to Docker but more infrastructure-native for servers/agents already using containerd.
package containerd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for containerd backend operations.
var (
	// ErrContainerdNotAvailable is returned when containerd is not available.
	ErrContainerdNotAvailable = errors.New("containerd not available")

	// ErrImageNotFound is returned when the execution image is not found.
	ErrImageNotFound = errors.New("image not found")

	// ErrContainerFailed is returned when container creation/execution fails.
	ErrContainerFailed = errors.New("container execution failed")

	// ErrClientNotConfigured is returned when no ContainerRunner is configured.
	ErrClientNotConfigured = errors.New("containerd client not configured")

	// ErrDaemonUnavailable is returned when the containerd daemon is not reachable.
	ErrDaemonUnavailable = errors.New("containerd daemon unavailable")

	// ErrSecurityViolation is returned when a security policy is violated.
	ErrSecurityViolation = errors.New("security policy violation")
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

// Config configures a containerd backend.
type Config struct {
	// ImageRef is the image reference to use for execution.
	// Default: toolruntime-sandbox:latest
	ImageRef string

	// Namespace is the containerd namespace to use.
	// Default: default
	Namespace string

	// SocketPath is the path to the containerd socket.
	// Default: /run/containerd/containerd.sock
	SocketPath string

	// Runtime is the containerd runtime to use.
	// Examples: "io.containerd.runc.v2", "io.containerd.runsc.v1", "io.containerd.kata.v2", "aws.firecracker".
	// Optional.
	Runtime string

	// SeccompPath is the path to a seccomp profile for hardened mode.
	SeccompPath string

	// Client executes container specs.
	// If nil, Execute() returns ErrClientNotConfigured.
	Client ContainerRunner

	// ImageResolver optionally resolves/pulls images before execution.
	ImageResolver ImageResolver

	// HealthChecker optionally verifies containerd availability.
	HealthChecker HealthChecker

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code via containerd with security isolation.
type Backend struct {
	imageRef   string
	namespace  string
	socketPath string
	runtime    string
	seccomp    string
	client     ContainerRunner
	resolver   ImageResolver
	health     HealthChecker
	logger     Logger
}

// New creates a new containerd backend with the given configuration.
func New(cfg Config) *Backend {
	imageRef := cfg.ImageRef
	if imageRef == "" {
		imageRef = "toolruntime-sandbox:latest"
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	socketPath := cfg.SocketPath
	if socketPath == "" {
		socketPath = "/run/containerd/containerd.sock"
	}

	return &Backend{
		imageRef:   imageRef,
		namespace:  namespace,
		socketPath: socketPath,
		runtime:    cfg.Runtime,
		seccomp:    cfg.SeccompPath,
		client:     cfg.Client,
		resolver:   cfg.ImageResolver,
		health:     cfg.HealthChecker,
		logger:     cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendContainerd
}

// Execute runs code via containerd with security isolation.
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

	image := b.imageRef
	if b.resolver != nil {
		resolved, err := b.resolver.Resolve(ctx, image)
		if err != nil {
			return runtime.ExecuteResult{}, err
		}
		image = resolved
	}

	profile := req.Profile
	if profile == "" {
		profile = runtime.ProfileStandard
	}

	spec, err := b.buildSpec(image, req, profile)
	if err != nil {
		return runtime.ExecuteResult{}, err
	}

	if b.logger != nil {
		b.logger.Info("executing in containerd",
			"profile", profile,
			"image", image,
			"runtime", b.runtime,
			"namespace", b.namespace)
	}

	containerResult, err := b.client.Run(ctx, spec)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(profile),
		}, err
	}

	return runtime.ExecuteResult{
		Value:    extractOutValue(containerResult.Stdout),
		Stdout:   containerResult.Stdout,
		Stderr:   containerResult.Stderr,
		Duration: containerResult.Duration,
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
		Kind:      runtime.BackendContainerd,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"imageRef":  b.imageRef,
			"namespace": b.namespace,
			"runtime":   b.runtime,
			"profile":   string(profile),
		},
	}
}

func (b *Backend) buildSpec(image string, req runtime.ExecuteRequest, profile runtime.SecurityProfile) (ContainerSpec, error) {
	opts := b.containerOptions(profile, req.Limits)

	spec := ContainerSpec{
		Image:   image,
		Runtime: b.runtime,
		Resources: ResourceSpec{
			MemoryBytes: opts.MemoryLimit,
			CPUQuota:    opts.CPUQuota,
			PidsLimit:   opts.PidsLimit,
			DiskBytes:   opts.DiskBytes,
		},
		Security: SecuritySpec{
			User:           opts.User,
			ReadOnlyRootfs: opts.ReadOnlyRootfs,
			NetworkMode:    opts.NetworkMode,
			SeccompProfile: opts.SeccompProfile,
		},
		Timeout: req.Timeout,
		Labels: map[string]string{
			"runtime.profile": string(profile),
			"runtime.backend": string(runtime.BackendContainerd),
		},
	}

	if err := spec.Validate(); err != nil {
		return ContainerSpec{}, err
	}
	return spec, nil
}

type containerOptions struct {
	NetworkMode    string
	ReadOnlyRootfs bool
	MemoryLimit    int64
	CPUQuota       int64
	PidsLimit      int64
	DiskBytes      int64
	SeccompProfile string
	User           string
}

func (b *Backend) containerOptions(profile runtime.SecurityProfile, limits runtime.Limits) containerOptions {
	opts := containerOptions{
		User: "nobody:nogroup",
	}

	switch profile {
	case runtime.ProfileDev:
		opts.NetworkMode = "bridge"
		opts.ReadOnlyRootfs = false
	case runtime.ProfileStandard:
		opts.NetworkMode = "none"
		opts.ReadOnlyRootfs = true
	case runtime.ProfileHardened:
		opts.NetworkMode = "none"
		opts.ReadOnlyRootfs = true
		if b.seccomp != "" {
			opts.SeccompProfile = b.seccomp
		}
	}

	if limits.MemoryBytes > 0 {
		opts.MemoryLimit = limits.MemoryBytes
	}
	if limits.CPUQuotaMillis > 0 {
		opts.CPUQuota = limits.CPUQuotaMillis * 1000
	}
	if limits.PidsMax > 0 {
		opts.PidsLimit = limits.PidsMax
	}
	if limits.DiskBytes > 0 {
		opts.DiskBytes = limits.DiskBytes
	}

	return opts
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
