// Package containerd provides a backend that executes code via containerd.
// Similar to Docker but more infrastructure-native for servers/agents already using containerd.
package containerd

import (
	"context"
	"errors"
	"fmt"
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

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code via containerd with security isolation.
type Backend struct {
	imageRef   string
	namespace  string
	socketPath string
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

	start := time.Now()

	result := runtime.ExecuteResult{
		Duration: time.Since(start),
		Backend: runtime.BackendInfo{
			Kind: runtime.BackendContainerd,
			Details: map[string]any{
				"imageRef":  b.imageRef,
				"namespace": b.namespace,
			},
		},
	}

	return result, fmt.Errorf("%w: containerd backend not fully implemented", ErrContainerdNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
