// Package remote provides a backend that executes code on a remote runtime service.
// Generic target for dedicated runtime services, batch systems, or job runners.
package remote

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

// Config configures a remote backend.
type Config struct {
	// Endpoint is the URL of the remote runtime service.
	// Required.
	Endpoint string

	// AuthToken is the authentication token for the remote service.
	AuthToken string

	// GatewayEndpoint is the URL of the tool gateway available to the remote runtime.
	// Optional, but recommended when remote code needs tool access.
	GatewayEndpoint string

	// GatewayToken is an optional token to authorize gateway access.
	GatewayToken string

	// TLSSkipVerify skips TLS certificate verification.
	// WARNING: Only use for development.
	TLSSkipVerify bool

	// TimeoutOverhead is additional timeout added to account for network latency.
	// Default: 5s
	TimeoutOverhead time.Duration

	// MaxRetries is the maximum number of retries on transient failures.
	// Default: 3
	MaxRetries int

	// HTTPClient overrides the default HTTP client.
	HTTPClient *http.Client

	// EnableStreaming enables SSE streaming when supported by the remote service.
	EnableStreaming bool

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code on a remote runtime service.
type Backend struct {
	endpoint        string
	authToken       string
	gatewayEndpoint string
	gatewayToken    string
	tlsSkipVerify   bool
	timeoutOverhead time.Duration
	maxRetries      int
	httpClient      *http.Client
	enableStreaming bool
	logger          Logger
}

// New creates a new remote backend with the given configuration.
func New(cfg Config) *Backend {
	timeoutOverhead := cfg.TimeoutOverhead
	if timeoutOverhead == 0 {
		timeoutOverhead = 5 * time.Second
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	client := cfg.HTTPClient
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.TLSSkipVerify {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   0,
		}
	}

	return &Backend{
		endpoint:        cfg.Endpoint,
		authToken:       cfg.AuthToken,
		gatewayEndpoint: cfg.GatewayEndpoint,
		gatewayToken:    cfg.GatewayToken,
		tlsSkipVerify:   cfg.TLSSkipVerify,
		timeoutOverhead: timeoutOverhead,
		maxRetries:      maxRetries,
		httpClient:      client,
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

	if b.endpoint == "" {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: endpoint not configured", ErrRemoteNotAvailable)
	}

	endpointURL, err := url.Parse(b.endpoint)
	if err != nil {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: invalid endpoint: %v", ErrRemoteNotAvailable, err)
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout+b.timeoutOverhead)
	defer cancel()

	start := time.Now()

	payload := remoteRequest{
		Request: buildExecutePayload(req),
		Gateway: buildGatewayDescriptor(b.gatewayEndpoint, b.gatewayToken),
		Stream:  b.enableStreaming,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: marshal request: %v", ErrRemoteExecutionFailed, err)
	}

	response, err := b.doRequest(ctx, endpointURL, data, payload.Stream)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(),
		}, err
	}

	result := mapRemoteResult(response)
	if result.Duration == 0 {
		result.Duration = time.Since(start)
	}
	result.Backend = b.backendInfo()
	return result, nil
}

var _ runtime.Backend = (*Backend)(nil)

type remoteRequest struct {
	Request executePayload     `json:"request"`
	Gateway *gatewayDescriptor `json:"gateway,omitempty"`
	Stream  bool               `json:"stream,omitempty"`
}

type gatewayDescriptor struct {
	URL      string `json:"url"`
	Token    string `json:"token,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type executePayload struct {
	Language       string         `json:"language,omitempty"`
	Code           string         `json:"code"`
	TimeoutMillis  int64          `json:"timeout_ms,omitempty"`
	Limits         limitsPayload  `json:"limits,omitempty"`
	Profile        string         `json:"profile,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	EnableTracing  bool           `json:"enable_tracing,omitempty"`
	RequestedScope string         `json:"requested_scope,omitempty"`
}

type limitsPayload struct {
	MaxToolCalls   int   `json:"max_tool_calls,omitempty"`
	MaxChainSteps  int   `json:"max_chain_steps,omitempty"`
	CPUQuotaMillis int64 `json:"cpu_quota_millis,omitempty"`
	MemoryBytes    int64 `json:"memory_bytes,omitempty"`
	PidsMax        int64 `json:"pids_max,omitempty"`
	DiskBytes      int64 `json:"disk_bytes,omitempty"`
}

type remoteResponse struct {
	Result *executeResultPayload `json:"result,omitempty"`
	Error  *remoteError          `json:"error,omitempty"`
}

type remoteError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type executeResultPayload struct {
	Value          any                    `json:"value,omitempty"`
	Stdout         string                 `json:"stdout,omitempty"`
	Stderr         string                 `json:"stderr,omitempty"`
	ToolCalls      []toolCallPayload      `json:"tool_calls,omitempty"`
	DurationMillis int64                  `json:"duration_ms,omitempty"`
	LimitsEnforced runtime.LimitsEnforced `json:"limits_enforced,omitempty"`
}

type toolCallPayload struct {
	ToolID      string `json:"tool_id"`
	BackendKind string `json:"backend_kind"`
	DurationMs  int64  `json:"duration_ms"`
	ErrorOp     string `json:"error_op,omitempty"`
}

type remoteResponseEnvelope struct {
	Result executeResultPayload
	Err    error
}

func buildExecutePayload(req runtime.ExecuteRequest) executePayload {
	payload := executePayload{
		Language: req.Language,
		Code:     req.Code,
		Profile:  string(req.Profile),
		Metadata: req.Metadata,
	}
	if req.Timeout > 0 {
		payload.TimeoutMillis = req.Timeout.Milliseconds()
	}
	payload.Limits = limitsPayload{
		MaxToolCalls:   req.Limits.MaxToolCalls,
		MaxChainSteps:  req.Limits.MaxChainSteps,
		CPUQuotaMillis: req.Limits.CPUQuotaMillis,
		MemoryBytes:    req.Limits.MemoryBytes,
		PidsMax:        req.Limits.PidsMax,
		DiskBytes:      req.Limits.DiskBytes,
	}
	return payload
}

func buildGatewayDescriptor(endpoint, token string) *gatewayDescriptor {
	if endpoint == "" {
		return nil
	}
	return &gatewayDescriptor{
		URL:      endpoint,
		Token:    token,
		Protocol: "toolruntime-gateway-http/v1",
	}
}

func (b *Backend) doRequest(ctx context.Context, endpoint *url.URL, payload []byte, stream bool) (remoteResponseEnvelope, error) {
	for attempt := 0; attempt <= b.maxRetries; attempt++ {
		resp, err := b.executeRequest(ctx, endpoint, payload, stream)
		if err == nil {
			return resp, nil
		}
		if !isRetryable(err) || attempt == b.maxRetries {
			return remoteResponseEnvelope{}, err
		}
		if b.logger != nil {
			b.logger.Warn("remote execution retry", "attempt", attempt+1, "error", err)
		}
	}
	return remoteResponseEnvelope{}, fmt.Errorf("%w: retries exhausted", ErrRemoteExecutionFailed)
}

func (b *Backend) executeRequest(ctx context.Context, endpoint *url.URL, payload []byte, stream bool) (remoteResponseEnvelope, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(string(payload)))
	if err != nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: build request: %v", ErrConnectionFailed, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	if b.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+b.authToken)
		signRequest(req, payload, b.authToken)
	}

	if b.logger != nil {
		b.logger.Info("remote execution request", "endpoint", endpoint.String(), "stream", stream)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return remoteResponseEnvelope{}, fmt.Errorf("%w: server error %d: %s", ErrRemoteExecutionFailed, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return remoteResponseEnvelope{}, fmt.Errorf("%w: status %d: %s", ErrRemoteExecutionFailed, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if stream && strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return b.readStream(resp.Body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: read response: %v", ErrRemoteExecutionFailed, err)
	}

	var response remoteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: decode response: %v", ErrRemoteExecutionFailed, err)
	}

	if response.Error != nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: %s", ErrRemoteExecutionFailed, response.Error.Message)
	}
	if response.Result == nil {
		return remoteResponseEnvelope{}, fmt.Errorf("%w: missing result", ErrRemoteExecutionFailed)
	}

	return remoteResponseEnvelope{Result: *response.Result}, nil
}

func (b *Backend) readStream(body io.Reader) (remoteResponseEnvelope, error) {
	decoder := newSSEDecoder(body)
	var result executeResultPayload
	for {
		event, err := decoder.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return remoteResponseEnvelope{}, fmt.Errorf("%w: stream decode: %v", ErrRemoteExecutionFailed, err)
		}
		switch event.Name {
		case "stdout":
			result.Stdout += event.Data
		case "stderr":
			result.Stderr += event.Data
		case "result":
			var payload executeResultPayload
			if err := json.Unmarshal([]byte(event.Data), &payload); err == nil {
				if payload.Stdout == "" {
					payload.Stdout = result.Stdout
				}
				if payload.Stderr == "" {
					payload.Stderr = result.Stderr
				}
				if len(payload.ToolCalls) == 0 {
					payload.ToolCalls = result.ToolCalls
				}
				result = payload
			}
		case "error":
			return remoteResponseEnvelope{}, fmt.Errorf("%w: %s", ErrRemoteExecutionFailed, event.Data)
		}
	}
	return remoteResponseEnvelope{Result: result}, nil
}

func signRequest(req *http.Request, payload []byte, token string) {
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Toolruntime-Timestamp", timestamp)
	req.Header.Set("X-Toolruntime-Signature", signature)
}

func mapRemoteResult(response remoteResponseEnvelope) runtime.ExecuteResult {
	payload := response.Result

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
	return runtime.BackendInfo{
		Kind:      runtime.BackendRemote,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"endpoint": b.endpoint,
		},
	}
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	return true
}
