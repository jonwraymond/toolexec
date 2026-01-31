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

	// TLSSkipVerify skips TLS certificate verification.
	// WARNING: Only use for development.
	TLSSkipVerify bool

	// TimeoutOverhead is additional timeout added to account for network latency.
	// Default: 5s
	TimeoutOverhead time.Duration

	// MaxRetries is the maximum number of retries on transient failures.
	// Default: 3
	MaxRetries int

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code on a remote runtime service.
type Backend struct {
	endpoint        string
	authToken       string
	tlsSkipVerify   bool
	timeoutOverhead time.Duration
	maxRetries      int
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

	return &Backend{
		endpoint:        cfg.Endpoint,
		authToken:       cfg.AuthToken,
		tlsSkipVerify:   cfg.TLSSkipVerify,
		timeoutOverhead: timeoutOverhead,
		maxRetries:      maxRetries,
		logger:          cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendRemote
}

// Execute runs code on the remote runtime service.
func (b *Backend) Execute(_ context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	if b.endpoint == "" {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: endpoint not configured", ErrRemoteNotAvailable)
	}

	start := time.Now()

	result := runtime.ExecuteResult{
		Duration: time.Since(start),
		Backend: runtime.BackendInfo{
			Kind: runtime.BackendRemote,
			Details: map[string]any{
				"endpoint": b.endpoint,
			},
		},
	}

	return result, fmt.Errorf("%w: remote backend not fully implemented", ErrRemoteNotAvailable)
}

var _ runtime.Backend = (*Backend)(nil)
