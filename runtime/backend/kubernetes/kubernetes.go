// Package kubernetes provides a backend that executes code in Kubernetes pods/jobs.
// Best for scheduling, quotas, and multi-tenant controls; isolation depends on runtime class.
package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Kubernetes backend operations.
var (
	// ErrKubernetesNotAvailable is returned when Kubernetes is not available.
	ErrKubernetesNotAvailable = errors.New("kubernetes not available")

	// ErrPodCreationFailed is returned when pod creation fails.
	ErrPodCreationFailed = errors.New("pod creation failed")

	// ErrPodExecutionFailed is returned when pod execution fails.
	ErrPodExecutionFailed = errors.New("pod execution failed")
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

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code in Kubernetes pods/jobs.
type Backend struct {
	namespace        string
	image            string
	runtimeClassName string
	serviceAccount   string
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
		logger:           cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendKubernetes
}

// Execute runs code in a Kubernetes pod.
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
			Kind: runtime.BackendKubernetes,
			Details: map[string]any{
				"namespace":        b.namespace,
				"image":            b.image,
				"runtimeClassName": b.runtimeClassName,
			},
		},
	}

	return result, fmt.Errorf("%w: kubernetes backend not fully implemented", ErrKubernetesNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
