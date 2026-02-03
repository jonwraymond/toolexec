// Package remote provides a backend that executes code on a remote runtime service.
// Generic target for dedicated runtime services, batch systems, or job runners.
package remote

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for remote backend operations.
var (
	// ErrRemoteNotAvailable is returned when the remote service is not available.
	ErrRemoteNotAvailable = errors.New("remote service not available")

	// ErrConnectionFailed is returned when connection to remote service fails.
	ErrConnectionFailed = errors.New("connection to remote service failed")

	// ErrRemoteExecutionFailed is returned when remote execution fails.
	ErrRemoteExecutionFailed = errors.New("remote execution failed")

	// ErrClientNotConfigured is returned when no remote client is configured.
	ErrClientNotConfigured = errors.New("remote client not configured")
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

// RemoteClient executes remote requests.
//
// Contract:
// - Concurrency: Implementations must be safe for concurrent use.
// - Context: Execute must honor cancellation and deadlines.
type RemoteClient interface {
	Execute(ctx context.Context, req RemoteRequest) (RemoteResponse, error)
}

// EndpointProvider optionally exposes the configured endpoint for diagnostics.
type EndpointProvider interface {
	Endpoint() string
}

// Config configures a remote backend.
type Config struct {
	// Client executes remote requests.
	// Required. Provide a RemoteClient from an integration package.
	Client RemoteClient

	// GatewayEndpoint is the URL of the tool gateway available to the remote runtime.
	// Optional, but recommended when remote code needs tool access.
	GatewayEndpoint string

	// GatewayToken is an optional token to authorize gateway access.
	GatewayToken string

	// TimeoutOverhead is additional timeout added to account for network latency.
	// Default: 5s
	TimeoutOverhead time.Duration

	// EnableStreaming enables SSE streaming when supported by the remote service.
	EnableStreaming bool

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code on a remote runtime service.
type Backend struct {
	client          RemoteClient
	gatewayEndpoint string
	gatewayToken    string
	timeoutOverhead time.Duration
	enableStreaming bool
	logger          Logger
}

// New creates a new remote backend with the given configuration.
func New(cfg Config) *Backend {
	timeoutOverhead := cfg.TimeoutOverhead
	if timeoutOverhead == 0 {
		timeoutOverhead = 5 * time.Second
	}

	return &Backend{
		client:          cfg.Client,
		gatewayEndpoint: cfg.GatewayEndpoint,
		gatewayToken:    cfg.GatewayToken,
		timeoutOverhead: timeoutOverhead,
		enableStreaming: cfg.EnableStreaming,
		logger:          cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendRemote
}

// Execute runs code on the remote runtime service.
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

	ctx, cancel := context.WithTimeout(ctx, timeout+b.timeoutOverhead)
	defer cancel()

	start := time.Now()

	payload := RemoteRequest{
		Request: buildExecutePayload(req),
		Gateway: buildGatewayDescriptor(b.gatewayEndpoint, b.gatewayToken),
		Stream:  b.enableStreaming,
	}

	response, err := b.client.Execute(ctx, payload)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(),
		}, err
	}
	if response.Error != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(),
		}, fmt.Errorf("%w: %s", ErrRemoteExecutionFailed, response.Error.Message)
	}
	if response.Result == nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(),
		}, fmt.Errorf("%w: missing result", ErrRemoteExecutionFailed)
	}

	result := mapRemoteResult(*response.Result)
	if result.Duration == 0 {
		result.Duration = time.Since(start)
	}
	result.Backend = b.backendInfo()
	return result, nil
}

var _ runtime.Backend = (*Backend)(nil)

// RemoteRequest is the wire request to a remote runtime.
type RemoteRequest struct {
	Request ExecutePayload     `json:"request"`
	Gateway *GatewayDescriptor `json:"gateway,omitempty"`
	Stream  bool               `json:"stream,omitempty"`
}

// GatewayDescriptor describes the tool gateway accessible to the runtime.
type GatewayDescriptor struct {
	URL      string `json:"url"`
	Token    string `json:"token,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// ExecutePayload defines the code execution request payload.
type ExecutePayload struct {
	Language       string         `json:"language,omitempty"`
	Code           string         `json:"code"`
	TimeoutMillis  int64          `json:"timeout_ms,omitempty"`
	Limits         LimitsPayload  `json:"limits,omitempty"`
	Profile        string         `json:"profile,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	EnableTracing  bool           `json:"enable_tracing,omitempty"`
	RequestedScope string         `json:"requested_scope,omitempty"`
}

// LimitsPayload encodes execution limits for remote runtimes.
type LimitsPayload struct {
	MaxToolCalls   int   `json:"max_tool_calls,omitempty"`
	MaxChainSteps  int   `json:"max_chain_steps,omitempty"`
	CPUQuotaMillis int64 `json:"cpu_quota_millis,omitempty"`
	MemoryBytes    int64 `json:"memory_bytes,omitempty"`
	PidsMax        int64 `json:"pids_max,omitempty"`
	DiskBytes      int64 `json:"disk_bytes,omitempty"`
}

// RemoteResponse is the wire response from a remote runtime.
type RemoteResponse struct {
	Result *ExecuteResultPayload `json:"result,omitempty"`
	Error  *RemoteError          `json:"error,omitempty"`
}

// RemoteError describes a remote runtime error.
type RemoteError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ExecuteResultPayload is the remote execution result payload.
type ExecuteResultPayload struct {
	Value          any                    `json:"value,omitempty"`
	Stdout         string                 `json:"stdout,omitempty"`
	Stderr         string                 `json:"stderr,omitempty"`
	ToolCalls      []ToolCallPayload      `json:"tool_calls,omitempty"`
	DurationMillis int64                  `json:"duration_ms,omitempty"`
	LimitsEnforced runtime.LimitsEnforced `json:"limits_enforced,omitempty"`
}

// ToolCallPayload records tool call metadata from a remote execution.
type ToolCallPayload struct {
	ToolID      string `json:"tool_id"`
	BackendKind string `json:"backend_kind"`
	DurationMs  int64  `json:"duration_ms"`
	ErrorOp     string `json:"error_op,omitempty"`
}

func buildExecutePayload(req runtime.ExecuteRequest) ExecutePayload {
	payload := ExecutePayload{
		Language: req.Language,
		Code:     req.Code,
		Profile:  string(req.Profile),
		Metadata: req.Metadata,
	}
	if req.Timeout > 0 {
		payload.TimeoutMillis = req.Timeout.Milliseconds()
	}
	payload.Limits = LimitsPayload{
		MaxToolCalls:   req.Limits.MaxToolCalls,
		MaxChainSteps:  req.Limits.MaxChainSteps,
		CPUQuotaMillis: req.Limits.CPUQuotaMillis,
		MemoryBytes:    req.Limits.MemoryBytes,
		PidsMax:        req.Limits.PidsMax,
		DiskBytes:      req.Limits.DiskBytes,
	}
	return payload
}

func buildGatewayDescriptor(endpoint, token string) *GatewayDescriptor {
	if endpoint == "" {
		return nil
	}
	return &GatewayDescriptor{
		URL:      endpoint,
		Token:    token,
		Protocol: "toolruntime-gateway-http/v1",
	}
}

func mapRemoteResult(payload ExecuteResultPayload) runtime.ExecuteResult {
	result := runtime.ExecuteResult{
		Value:    payload.Value,
		Stdout:   payload.Stdout,
		Stderr:   payload.Stderr,
		Duration: time.Duration(payload.DurationMillis) * time.Millisecond,
		LimitsEnforced: runtime.LimitsEnforced{
			Timeout:    payload.LimitsEnforced.Timeout,
			ToolCalls:  payload.LimitsEnforced.ToolCalls,
			ChainSteps: payload.LimitsEnforced.ChainSteps,
			Memory:     payload.LimitsEnforced.Memory,
			CPU:        payload.LimitsEnforced.CPU,
			Pids:       payload.LimitsEnforced.Pids,
			Disk:       payload.LimitsEnforced.Disk,
		},
	}

	if len(payload.ToolCalls) > 0 {
		result.ToolCalls = make([]runtime.ToolCallRecord, len(payload.ToolCalls))
		for i, call := range payload.ToolCalls {
			result.ToolCalls[i] = runtime.ToolCallRecord{
				ToolID:      call.ToolID,
				BackendKind: call.BackendKind,
				Duration:    time.Duration(call.DurationMs) * time.Millisecond,
				ErrorOp:     call.ErrorOp,
			}
		}
	}

	return result
}

func (b *Backend) backendInfo() runtime.BackendInfo {
	details := map[string]any{}
	if provider, ok := b.client.(EndpointProvider); ok {
		if endpoint := provider.Endpoint(); endpoint != "" {
			details["endpoint"] = endpoint
		}
	}
	return runtime.BackendInfo{
		Kind:      runtime.BackendRemote,
		Readiness: runtime.ReadinessBeta,
		Details:   details,
	}
}
