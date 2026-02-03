// Package temporal provides a backend that treats snippet execution as a Temporal workflow/activity.
// Useful for long-running or resumable executions.
// Note: Temporal is orchestration, not isolation - must compose with sandbox backends.
package temporal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for Temporal backend operations.
var (
	// ErrTemporalNotAvailable is returned when Temporal is not available.
	ErrTemporalNotAvailable = errors.New("temporal not available")

	// ErrWorkflowCreationFailed is returned when workflow creation fails.
	ErrWorkflowCreationFailed = errors.New("workflow creation failed")

	// ErrWorkflowExecutionFailed is returned when workflow execution fails.
	ErrWorkflowExecutionFailed = errors.New("workflow execution failed")

	// ErrMissingSandboxBackend is returned when no sandbox backend is configured.
	ErrMissingSandboxBackend = errors.New("temporal backend requires a sandbox backend for isolation")
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

// Config configures a Temporal backend.
type Config struct {
	// HostPort is the Temporal server address.
	// Default: localhost:7233
	HostPort string

	// Namespace is the Temporal namespace.
	// Default: default
	Namespace string

	// TaskQueue is the task queue for execution activities.
	// Default: toolruntime-execution
	TaskQueue string

	// SandboxBackend is the backend used for actual code execution.
	// Temporal provides orchestration; this backend provides isolation.
	// Required for secure execution.
	SandboxBackend runtime.Backend

	// WorkflowIDPrefix is the prefix for workflow IDs.
	// Default: toolruntime-
	WorkflowIDPrefix string

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code as Temporal workflows/activities.
type Backend struct {
	hostPort         string
	namespace        string
	taskQueue        string
	sandboxBackend   runtime.Backend
	workflowIDPrefix string
	logger           Logger
}

// New creates a new Temporal backend with the given configuration.
func New(cfg Config) *Backend {
	hostPort := cfg.HostPort
	if hostPort == "" {
		hostPort = "localhost:7233"
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	taskQueue := cfg.TaskQueue
	if taskQueue == "" {
		taskQueue = "toolruntime-execution"
	}

	workflowIDPrefix := cfg.WorkflowIDPrefix
	if workflowIDPrefix == "" {
		workflowIDPrefix = "toolruntime-"
	}

	return &Backend{
		hostPort:         hostPort,
		namespace:        namespace,
		taskQueue:        taskQueue,
		sandboxBackend:   cfg.SandboxBackend,
		workflowIDPrefix: workflowIDPrefix,
		logger:           cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendTemporal
}

// Execute runs code as a Temporal workflow.
// The actual code execution is delegated to the configured sandbox backend.
func (b *Backend) Execute(_ context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	// Temporal is orchestration, not isolation - must have a sandbox backend
	if b.sandboxBackend == nil {
		return runtime.ExecuteResult{}, ErrMissingSandboxBackend
	}

	start := time.Now()

	result := runtime.ExecuteResult{
		Duration: time.Since(start),
		Backend: runtime.BackendInfo{
			Kind:      runtime.BackendTemporal,
			Readiness: runtime.ReadinessStub,
			Details: map[string]any{
				"namespace":      b.namespace,
				"taskQueue":      b.taskQueue,
				"sandboxBackend": string(b.sandboxBackend.Kind()),
			},
		},
	}

	return result, fmt.Errorf("%w: temporal backend not fully implemented", ErrTemporalNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
