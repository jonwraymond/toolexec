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
	ErrRuntimeNotConfigured = errors.New("runtime client not configured")

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
	// Node is the Proxmox node name.
	Node string

	// VMID is the LXC container ID.
	VMID int

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
	// Required. Provide an APIClient from an integration package.
	Client APIClient

	// RuntimeClient executes code in the LXC runtime service.
	// Required. Provide a RemoteClient from an integration package.
	RuntimeClient remote.RemoteClient

	// RuntimeGatewayEndpoint is the tool gateway URL the runtime can use.
	RuntimeGatewayEndpoint string

	// RuntimeGatewayToken is an optional token for the tool gateway.
	RuntimeGatewayToken string

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code via Proxmox LXC using a runtime service inside the container.
type Backend struct {
	client                 APIClient
	runtime                *remote.Backend
	runtimeClient          remote.RemoteClient
	runtimeGatewayEndpoint string
	runtimeGatewayToken    string
	node                   string
	vmid                   int
	autoStart              bool
	autoStop               bool
	startTimeout           time.Duration
	pollInterval           time.Duration
	logger                 Logger
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

	return &Backend{
		client:                 cfg.Client,
		runtimeClient:          cfg.RuntimeClient,
		runtimeGatewayEndpoint: cfg.RuntimeGatewayEndpoint,
		runtimeGatewayToken:    cfg.RuntimeGatewayToken,
		node:                   cfg.Node,
		vmid:                   cfg.VMID,
		autoStart:              autoStart,
		autoStop:               autoStop,
		startTimeout:           startTimeout,
		pollInterval:           poll,
		logger:                 cfg.Logger,
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
	if b.runtimeClient == nil {
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
			Client:          b.runtimeClient,
			GatewayEndpoint: b.runtimeGatewayEndpoint,
			GatewayToken:    b.runtimeGatewayToken,
			EnableStreaming: true,
			Logger:          b.logger,
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
	return nil, ErrClientNotConfigured
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
	details := map[string]any{
		"node":    b.node,
		"vmid":    b.vmid,
		"profile": string(profile),
	}
	if provider, ok := b.runtimeClient.(interface{ Endpoint() string }); ok {
		if endpoint := provider.Endpoint(); endpoint != "" {
			details["endpoint"] = endpoint
		}
	}
	return runtime.BackendInfo{
		Kind:      runtime.BackendProxmoxLXC,
		Readiness: runtime.ReadinessBeta,
		Details:   details,
	}
}

func boolValue(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}
