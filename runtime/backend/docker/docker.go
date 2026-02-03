// Package docker provides a backend that executes code in Docker containers
// with configurable security profiles and resource limits.
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Docker backend operations.
var (
	// ErrDockerNotAvailable is returned when Docker is not available.
	ErrDockerNotAvailable = errors.New("docker not available")

	// ErrImageNotFound is returned when the execution image is not found.
	ErrImageNotFound = errors.New("docker image not found")

	// ErrContainerFailed is returned when container creation/execution fails.
	ErrContainerFailed = errors.New("container execution failed")

	// ErrClientNotConfigured is returned when no ContainerRunner is configured.
	ErrClientNotConfigured = errors.New("docker client not configured")

	// ErrContainerCreate is returned when container creation fails.
	ErrContainerCreate = errors.New("container creation failed")

	// ErrContainerStart is returned when container start fails.
	ErrContainerStart = errors.New("container start failed")

	// ErrContainerWait is returned when waiting for container completion fails.
	ErrContainerWait = errors.New("container wait failed")

	// ErrImagePull is returned when image pull fails.
	ErrImagePull = errors.New("image pull failed")

	// ErrDaemonUnavailable is returned when the Docker daemon is not reachable.
	ErrDaemonUnavailable = errors.New("docker daemon unavailable")

	// ErrResourceLimit is returned when a resource limit is exceeded.
	ErrResourceLimit = errors.New("resource limit exceeded")

	// ErrSecurityViolation is returned when a security policy is violated.
	ErrSecurityViolation = errors.New("security policy violation")
)

// ClientError wraps client operation errors with context.
type ClientError struct {
	// Op is the operation that failed: "create", "start", "wait", "pull".
	Op string

	// Image is the image reference.
	Image string

	// ContainerID is the container ID if available.
	ContainerID string

	// Err is the underlying error.
	Err error
}

func (e *ClientError) Error() string {
	if e.ContainerID != "" {
		return fmt.Sprintf("docker %s %s (%s): %v", e.Op, e.Image, e.ContainerID, e.Err)
	}
	return fmt.Sprintf("docker %s %s: %v", e.Op, e.Image, e.Err)
}

func (e *ClientError) Unwrap() error { return e.Err }

// ContainerOptions represents container configuration for execution.
type ContainerOptions struct {
	// NetworkDisabled disables network access.
	NetworkDisabled bool

	// ReadOnlyRootfs makes the root filesystem read-only.
	ReadOnlyRootfs bool

	// MemoryLimit is the memory limit in bytes.
	MemoryLimit int64

	// CPUQuota is the CPU quota in microseconds.
	CPUQuota int64

	// PidsLimit is the maximum number of processes.
	PidsLimit int64

	// SeccompProfile is the path to a seccomp profile.
	SeccompProfile string

	// User is the user to run as (non-root).
	User string
}

// Config configures a Docker backend.
type Config struct {
	// ImageName is the Docker image to use for execution.
	// Default: toolruntime-sandbox:latest
	ImageName string

	// SeccompPath is the path to a custom seccomp profile for hardened mode.
	SeccompPath string

	// Client is the container runner implementation.
	// If nil, Execute() returns ErrClientNotConfigured.
	Client ContainerRunner

	// ImageResolver optionally resolves/pulls images before execution.
	// If nil, images are assumed to exist locally.
	ImageResolver ImageResolver

	// HealthChecker optionally verifies daemon health before execution.
	// If nil, health checks are skipped.
	HealthChecker HealthChecker

	// Logger is an optional logger for backend events.
	Logger Logger
}

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

// Backend executes code in Docker containers with security isolation.
type Backend struct {
	imageName     string
	seccompPath   string
	client        ContainerRunner
	imageResolver ImageResolver
	healthChecker HealthChecker
	logger        Logger
}

// New creates a new Docker backend with the given configuration.
func New(cfg Config) *Backend {
	imageName := cfg.ImageName
	if imageName == "" {
		imageName = "toolruntime-sandbox:latest"
	}

	return &Backend{
		imageName:     imageName,
		seccompPath:   cfg.SeccompPath,
		client:        cfg.Client,
		imageResolver: cfg.ImageResolver,
		healthChecker: cfg.HealthChecker,
		logger:        cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendDocker
}

// Execute runs code in a Docker container with security isolation.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	// Check client is configured
	if b.client == nil {
		return runtime.ExecuteResult{}, ErrClientNotConfigured
	}

	// Apply timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return runtime.ExecuteResult{}, ctx.Err()
	default:
	}

	start := time.Now()

	// Get container options based on profile and limits
	profile := req.Profile
	if profile == "" {
		profile = runtime.ProfileStandard
	}

	// Optional health check
	if b.healthChecker != nil {
		if err := b.healthChecker.Ping(ctx); err != nil {
			return runtime.ExecuteResult{}, fmt.Errorf("%w: %v", ErrDaemonUnavailable, err)
		}
	}

	// Optional image resolution
	image := b.imageName
	if b.imageResolver != nil {
		resolved, err := b.imageResolver.Resolve(ctx, image)
		if err != nil {
			return runtime.ExecuteResult{}, err
		}
		image = resolved
	}

	// Build container spec from request
	spec, err := b.buildSpec(image, req, profile)
	if err != nil {
		return runtime.ExecuteResult{}, err
	}

	// Log execution
	if b.logger != nil {
		b.logger.Info("executing in Docker container",
			"profile", profile,
			"image", image,
			"networkDisabled", spec.Security.NetworkMode == "none",
			"readOnlyRootfs", spec.Security.ReadOnlyRootfs)
	}

	// Execute via client
	containerResult, err := b.client.Run(ctx, spec)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(profile),
		}, err
	}

	// Convert to ExecuteResult
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
			ToolCalls:  true, // Enforced by gateway
			ChainSteps: true, // Enforced by gateway
		},
	}, nil
}

// buildSpec creates a ContainerSpec from an ExecuteRequest.
func (b *Backend) buildSpec(image string, req runtime.ExecuteRequest, profile runtime.SecurityProfile) (ContainerSpec, error) {
	opts := b.containerOptions(profile, req.Limits)

	builder := NewSpecBuilder(image).
		WithTimeout(req.Timeout).
		WithSecurity(SecuritySpec{
			User:           opts.User,
			ReadOnlyRootfs: opts.ReadOnlyRootfs,
			NetworkMode:    b.networkMode(opts),
			SeccompProfile: opts.SeccompProfile,
		}).
		WithResources(ResourceSpec{
			MemoryBytes: opts.MemoryLimit,
			CPUQuota:    opts.CPUQuota,
			PidsLimit:   opts.PidsLimit,
		}).
		WithLabel("runtime.profile", string(profile)).
		WithLabel("runtime.backend", string(runtime.BackendDocker))

	return builder.Build()
}

// networkMode converts ContainerOptions to a network mode string.
func (b *Backend) networkMode(opts ContainerOptions) string {
	if opts.NetworkDisabled {
		return "none"
	}
	return "bridge"
}

// backendInfo returns BackendInfo for the given profile.
func (b *Backend) backendInfo(profile runtime.SecurityProfile) runtime.BackendInfo {
	return runtime.BackendInfo{
		Kind:      runtime.BackendDocker,
		Readiness: runtime.ReadinessProd,
		Details: map[string]any{
			"image":   b.imageName,
			"profile": string(profile),
		},
	}
}

// extractOutValue extracts the __out value from stdout if present.
// This follows the toolruntime convention for capturing return values.
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

// containerOptions returns ContainerOptions based on the security profile and limits.
func (b *Backend) containerOptions(profile runtime.SecurityProfile, limits runtime.Limits) ContainerOptions {
	opts := ContainerOptions{
		User: "nobody:nogroup", // Always run as non-root
	}

	switch profile {
	case runtime.ProfileDev:
		// Dev mode: minimal restrictions
		opts.NetworkDisabled = false
		opts.ReadOnlyRootfs = false

	case runtime.ProfileStandard:
		// Standard: no network, read-only rootfs
		opts.NetworkDisabled = true
		opts.ReadOnlyRootfs = true

	case runtime.ProfileHardened:
		// Hardened: all restrictions plus seccomp
		opts.NetworkDisabled = true
		opts.ReadOnlyRootfs = true
		if b.seccompPath != "" {
			opts.SeccompProfile = b.seccompPath
		}
	}

	// Apply resource limits
	if limits.MemoryBytes > 0 {
		opts.MemoryLimit = limits.MemoryBytes
	}
	if limits.CPUQuotaMillis > 0 {
		// Convert milliseconds to microseconds for Docker
		opts.CPUQuota = limits.CPUQuotaMillis * 1000
	}
	if limits.PidsMax > 0 {
		opts.PidsLimit = limits.PidsMax
	}

	return opts
}
