// Package proxmox provides a backend that executes code via a Proxmox LXC runtime.
// The backend ensures an LXC container is running and delegates execution to a
// runtime service inside the container via the remote backend.
package proxmox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
	"github.com/jonwraymond/toolexec/runtime/backend/remote"
)

// Errors for Proxmox backend operations.
var (
	// ErrProxmoxNotAvailable is returned when Proxmox is not available.
	ErrProxmoxNotAvailable = errors.New("proxmox not available")

	// ErrClientNotConfigured is returned when no API client is configured.
	ErrClientNotConfigured = errors.New("proxmox client not configured")

	// ErrAuthNotConfigured is returned when no API token is configured.
	ErrAuthNotConfigured = errors.New("proxmox api token not configured")

	// ErrRuntimeNotConfigured is returned when no runtime endpoint is configured.
	ErrRuntimeNotConfigured = errors.New("runtime endpoint not configured")

	// ErrLXCNotRunning is returned when the LXC container is not running.
	ErrLXCNotRunning = errors.New("lxc container not running")
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

// Config configures a Proxmox LXC backend.
type Config struct {
	// Endpoint is the Proxmox API base URL (https://host:8006/api2/json).
	Endpoint string

	// TokenID is the user@realm!tokenid portion of the API token.
	TokenID string

	// TokenSecret is the API token secret UUID.
	TokenSecret string

	// Node is the Proxmox node name.
	Node string

	// VMID is the LXC container ID.
	VMID int

	// RuntimeEndpoint is the HTTP endpoint of the runtime service inside the LXC container.
	RuntimeEndpoint string

	// RuntimeToken is an optional token for the runtime service.
	RuntimeToken string

	// TLSSkipVerify disables TLS verification for the Proxmox API.
	TLSSkipVerify bool

	// AutoStart controls whether the backend starts the LXC container if stopped.
	// Default: true
	AutoStart *bool

	// AutoStop controls whether the backend stops the container after execution.
	// Default: false
	AutoStop *bool

	// StartTimeout is the maximum time to wait for the container to start.
	StartTimeout time.Duration

	// PollInterval controls how often to poll for container status.
	PollInterval time.Duration

	// Client overrides the default API client.
	Client APIClient

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code via Proxmox LXC using a runtime service inside the container.
type Backend struct {
	clientConfig *ClientConfig
	client       APIClient
	runtime      *remote.Backend
	node         string
	vmid         int
	runtimeURL   string
	runtimeToken string
	autoStart    bool
	autoStop     bool
	startTimeout time.Duration
	pollInterval time.Duration
	logger       Logger
}

// New creates a new Proxmox LXC backend with the given configuration.
func New(cfg Config) *Backend {
	autoStart := boolValue(cfg.AutoStart, true)
	autoStop := boolValue(cfg.AutoStop, false)
	startTimeout := cfg.StartTimeout
	if startTimeout == 0 {
		startTimeout = 2 * time.Minute
	}
	poll := cfg.PollInterval
	if poll == 0 {
		poll = 2 * time.Second
	}

	var clientConfig *ClientConfig
	if cfg.Client == nil && cfg.Endpoint != "" {
		clientConfig = &ClientConfig{
			Endpoint:      cfg.Endpoint,
			TokenID:       cfg.TokenID,
			TokenSecret:   cfg.TokenSecret,
			TLSSkipVerify: cfg.TLSSkipVerify,
		}
	}

	return &Backend{
		clientConfig: clientConfig,
		client:       cfg.Client,
		node:         cfg.Node,
		vmid:         cfg.VMID,
		runtimeURL:   cfg.RuntimeEndpoint,
		runtimeToken: cfg.RuntimeToken,
		autoStart:    autoStart,
		autoStop:     autoStop,
		startTimeout: startTimeout,
		pollInterval: poll,
		logger:       cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendProxmoxLXC
}

// Execute runs code in an LXC-backed runtime service.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}
	if b.runtimeURL == "" {
		return runtime.ExecuteResult{}, ErrRuntimeNotConfigured
	}

	client, err := b.ensureClient()
	if err != nil {
		return runtime.ExecuteResult{}, err
	}

	if b.autoStart {
		if err := b.ensureRunning(ctx, client); err != nil {
			return runtime.ExecuteResult{}, err
		}
	}

	if b.runtime == nil {
		b.runtime = remote.New(remote.Config{
			Endpoint:        b.runtimeURL,
			AuthToken:       b.runtimeToken,
			EnableStreaming: true,
		})
	}

	result, err := b.runtime.Execute(ctx, req)
	result.Backend = b.backendInfo(req.Profile)

	if b.autoStop {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = client.Stop(stopCtx, b.node, b.vmid)
		cancel()
	}

	return result, err
}

var _ runtime.Backend = (*Backend)(nil)

func (b *Backend) ensureClient() (APIClient, error) {
	if b.client != nil {
		return b.client, nil
	}
	if b.clientConfig == nil {
		return nil, ErrClientNotConfigured
	}
	client, err := NewClient(*b.clientConfig, b.logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProxmoxNotAvailable, err)
	}
	b.client = client
	return b.client, nil
}

func (b *Backend) ensureRunning(ctx context.Context, client APIClient) error {
	if b.node == "" || b.vmid == 0 {
		return ErrProxmoxNotAvailable
	}

	status, err := client.Status(ctx, b.node, b.vmid)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProxmoxNotAvailable, err)
	}
	if status.Status == "running" {
		return nil
	}
	if !b.autoStart {
		return ErrLXCNotRunning
	}

	if b.logger != nil {
		b.logger.Info("starting proxmox lxc", "node", b.node, "vmid", b.vmid)
	}

	if err := client.Start(ctx, b.node, b.vmid); err != nil {
		return fmt.Errorf("%w: %v", ErrProxmoxNotAvailable, err)
	}

	startCtx, cancel := context.WithTimeout(ctx, b.startTimeout)
	defer cancel()

	for {
		status, err := client.Status(startCtx, b.node, b.vmid)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProxmoxNotAvailable, err)
		}
		if status.Status == "running" {
			return nil
		}
		select {
		case <-startCtx.Done():
			return startCtx.Err()
		case <-time.After(b.pollInterval):
		}
	}
}

func (b *Backend) backendInfo(profile runtime.SecurityProfile) runtime.BackendInfo {
	return runtime.BackendInfo{
		Kind:      runtime.BackendProxmoxLXC,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"node":     b.node,
			"vmid":     b.vmid,
			"endpoint": b.runtimeURL,
			"profile":  string(profile),
		},
	}
}

func boolValue(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}
