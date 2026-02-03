// Package gvisor provides a backend that executes code with gVisor (runsc).
// Provides stronger isolation than plain containers; appropriate for untrusted multi-tenant execution.
package gvisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	// ErrClientNotConfigured is returned when no SandboxRunner is configured.
	ErrClientNotConfigured = errors.New("gvisor runner not configured")

	// ErrDaemonUnavailable is returned when runsc is not reachable.
	ErrDaemonUnavailable = errors.New("gvisor daemon unavailable")

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

// Config configures a gVisor backend.
type Config struct {
	// RunscPath is the path to the runsc binary.
	// Default: runsc (uses PATH)
	RunscPath string

	// Image is the container image to use for execution.
	// Default: toolruntime-sandbox:latest
	Image string

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

	// Client executes sandbox specs.
	// If nil, Execute() returns ErrClientNotConfigured.
	Client SandboxRunner

	// ImageResolver optionally resolves/pulls images before execution.
	ImageResolver ImageResolver

	// HealthChecker optionally verifies gVisor availability.
	HealthChecker HealthChecker

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code with gVisor for stronger isolation.
type Backend struct {
	runscPath   string
	rootDir     string
	platform    string
	networkMode string
	image       string
	client      SandboxRunner
	resolver    ImageResolver
	health      HealthChecker
	logger      Logger
}

// New creates a new gVisor backend with the given configuration.
func New(cfg Config) *Backend {
	runscPath := cfg.RunscPath
	if runscPath == "" {
		runscPath = "runsc"
	}

	image := cfg.Image
	if image == "" {
		image = "toolruntime-sandbox:latest"
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
		image:       image,
		client:      cfg.Client,
		resolver:    cfg.ImageResolver,
		health:      cfg.HealthChecker,
		logger:      cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendGVisor
}

// Execute runs code with gVisor isolation.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
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

	image := b.image
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
		b.logger.Info("executing in gvisor",
			"profile", profile,
			"image", image,
			"platform", b.platform,
			"networkMode", spec.Security.NetworkMode)
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
		Kind:      runtime.BackendGVisor,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"platform":    b.platform,
			"networkMode": b.networkMode,
			"profile":     string(profile),
		},
	}
}

func (b *Backend) buildSpec(image string, req runtime.ExecuteRequest, profile runtime.SecurityProfile) (SandboxSpec, error) {
	opts := b.sandboxOptions(profile, req.Limits)

	spec := SandboxSpec{
		Image:      image,
		Platform:   b.platform,
		RunscPath:  b.runscPath,
		RootDir:    b.rootDir,
		Resources:  ResourceSpec{MemoryBytes: opts.MemoryLimit, CPUQuota: opts.CPUQuota, PidsLimit: opts.PidsLimit, DiskBytes: opts.DiskBytes},
		Security:   SecuritySpec{User: opts.User, ReadOnlyRootfs: opts.ReadOnlyRootfs, NetworkMode: opts.NetworkMode},
		Timeout:    req.Timeout,
		Labels:     map[string]string{"runtime.profile": string(profile), "runtime.backend": string(runtime.BackendGVisor)},
	}

	if err := spec.Validate(); err != nil {
		return SandboxSpec{}, err
	}
	return spec, nil
}

type sandboxOptions struct {
	NetworkMode    string
	ReadOnlyRootfs bool
	MemoryLimit    int64
	CPUQuota       int64
	PidsLimit      int64
	DiskBytes      int64
	User           string
}

func (b *Backend) sandboxOptions(profile runtime.SecurityProfile, limits runtime.Limits) sandboxOptions {
	opts := sandboxOptions{
		User: "nobody:nogroup",
	}

	switch profile {
	case runtime.ProfileDev:
		opts.NetworkMode = b.networkMode
		opts.ReadOnlyRootfs = false
	case runtime.ProfileStandard:
		opts.NetworkMode = "none"
		opts.ReadOnlyRootfs = true
	case runtime.ProfileHardened:
		opts.NetworkMode = "none"
		opts.ReadOnlyRootfs = true
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
