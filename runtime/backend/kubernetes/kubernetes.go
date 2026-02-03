// Package kubernetes provides a backend that executes code in Kubernetes pods/jobs.
// Best for scheduling, quotas, and multi-tenant controls; isolation depends on runtime class.
package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Kubernetes backend operations.
var (
	// ErrKubernetesNotAvailable is returned when Kubernetes is not available.
	ErrKubernetesNotAvailable = errors.New("kubernetes not available")

	// ErrClientNotConfigured is returned when no PodRunner is configured.
	ErrClientNotConfigured = errors.New("kubernetes client not configured")

	// ErrClusterUnavailable is returned when the API server cannot be reached.
	ErrClusterUnavailable = errors.New("kubernetes cluster unavailable")

	// ErrPodCreationFailed is returned when pod creation fails.
	ErrPodCreationFailed = errors.New("pod creation failed")

	// ErrPodExecutionFailed is returned when pod execution fails.
	ErrPodExecutionFailed = errors.New("pod execution failed")

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

// Config configures a Kubernetes backend.
type Config struct {
	// Namespace is the Kubernetes namespace for execution pods.
	// Default: default
	Namespace string

	// Image is the container image to use for execution.
	// Default: toolruntime-sandbox:latest
	Image string

	// RuntimeClassName is the optional runtime class for stronger isolation.
	// Examples: gvisor, kata
	RuntimeClassName string

	// ServiceAccount is the service account for execution pods.
	ServiceAccount string

	// Client executes pod specs.
	// Required. Provide a PodRunner from an integration package.
	Client PodRunner

	// ImageResolver optionally resolves images before execution.
	ImageResolver ImageResolver

	// HealthChecker optionally verifies cluster availability.
	HealthChecker HealthChecker

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code in Kubernetes pods/jobs.
type Backend struct {
	namespace        string
	image            string
	runtimeClassName string
	serviceAccount   string
	client           PodRunner
	resolver         ImageResolver
	health           HealthChecker
	logger           Logger
}

// New creates a new Kubernetes backend with the given configuration.
func New(cfg Config) *Backend {
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	image := cfg.Image
	if image == "" {
		image = "toolruntime-sandbox:latest"
	}

	return &Backend{
		namespace:        namespace,
		image:            image,
		runtimeClassName: cfg.RuntimeClassName,
		serviceAccount:   cfg.ServiceAccount,
		client:           cfg.Client,
		resolver:         cfg.ImageResolver,
		health:           cfg.HealthChecker,
		logger:           cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendKubernetes
}

// Execute runs code in a Kubernetes pod.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	client, err := b.ensureClient()
	if err != nil {
		return runtime.ExecuteResult{}, err
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	if b.health != nil {
		if err := b.health.Ping(ctx); err != nil {
			return runtime.ExecuteResult{}, fmt.Errorf("%w: %v", ErrClusterUnavailable, err)
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
		b.logger.Info("executing in kubernetes",
			"profile", profile,
			"namespace", b.namespace,
			"runtimeClassName", b.runtimeClassName)
	}

	runResult, err := client.Run(ctx, spec)
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
			Pids:       false,
			Disk:       req.Limits.DiskBytes > 0,
			ToolCalls:  true,
			ChainSteps: true,
		},
	}, nil
}

var _ runtime.Backend = (*Backend)(nil)

func (b *Backend) ensureClient() (PodRunner, error) {
	if b.client != nil {
		if b.health == nil {
			if checker, ok := b.client.(HealthChecker); ok {
				b.health = checker
			}
		}
		return b.client, nil
	}
	return nil, ErrClientNotConfigured
}

func (b *Backend) backendInfo(profile runtime.SecurityProfile) runtime.BackendInfo {
	return runtime.BackendInfo{
		Kind:      runtime.BackendKubernetes,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"namespace":        b.namespace,
			"image":            b.image,
			"runtimeClassName": b.runtimeClassName,
			"profile":          string(profile),
		},
	}
}

func (b *Backend) buildSpec(image string, req runtime.ExecuteRequest, profile runtime.SecurityProfile) (PodSpec, error) {
	opts := b.podOptions(profile, req.Limits)
	spec := PodSpec{
		Namespace:        b.namespace,
		Image:            image,
		RuntimeClassName: b.runtimeClassName,
		ServiceAccount:   b.serviceAccount,
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
		},
		Timeout: req.Timeout,
		Labels: map[string]string{
			"runtime.profile": string(profile),
			"runtime.backend": string(runtime.BackendKubernetes),
		},
	}
	if err := spec.Validate(); err != nil {
		return PodSpec{}, err
	}
	return spec, nil
}

type podOptions struct {
	NetworkMode    string
	ReadOnlyRootfs bool
	MemoryLimit    int64
	CPUQuota       int64
	PidsLimit      int64
	DiskBytes      int64
	User           string
}

func (b *Backend) podOptions(profile runtime.SecurityProfile, limits runtime.Limits) podOptions {
	opts := podOptions{
		User: "65534", // nobody user id by default
	}

	switch profile {
	case runtime.ProfileDev:
		opts.NetworkMode = "default"
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
		opts.CPUQuota = limits.CPUQuotaMillis
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
