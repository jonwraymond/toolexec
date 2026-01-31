package runtime

import (
	"context"
	"fmt"
	"sync"
)

// Runtime is the main interface for code execution.
// It manages backends and routes execution requests to the appropriate backend
// based on the security profile.
//
// Contract:
//   - Concurrency: implementations must be safe for concurrent use.
//   - Context: must honor cancellation/deadlines and return ctx.Err() when canceled.
//   - Errors: request validation should return ErrMissingGateway/ErrInvalidRequest;
//     backend selection failures return ErrRuntimeUnavailable/ErrBackendDenied.
//   - Ownership: requests are read-only; results are caller-owned snapshots.
type Runtime interface {
	// Execute runs code with the given request parameters.
	// It selects the appropriate backend based on the security profile
	// and delegates execution.
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}

// RuntimeConfig configures a DefaultRuntime instance.
type RuntimeConfig struct {
	// Backends maps security profiles to their backend implementations.
	Backends map[SecurityProfile]Backend

	// DenyUnsafeProfiles lists profiles that cannot use the unsafe backend.
	// If a profile is listed here and only the unsafe backend is available,
	// execution will be denied.
	DenyUnsafeProfiles []SecurityProfile

	// DefaultProfile is the profile to use when none is specified in the request.
	// If empty and no profile is specified, execution will fail.
	DefaultProfile SecurityProfile

	// Logger is an optional logger for runtime events.
	Logger Logger
}

// Logger is an optional interface for logging.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Errors: logging must be best-effort and must not panic.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// DefaultRuntime is the default implementation of Runtime.
// It routes requests to backends based on security profiles.
type DefaultRuntime struct {
	mu                 sync.RWMutex
	backends           map[SecurityProfile]Backend
	denyUnsafeProfiles map[SecurityProfile]bool
	defaultProfile     SecurityProfile
	logger             Logger
}

// toolCallRecorder is an optional interface implemented by gateways that
// can expose a tool call trace.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Ownership: returned slice is caller-owned; records are read-only snapshots.
type toolCallRecorder interface {
	GetToolCalls() []ToolCallRecord
}

// NewDefaultRuntime creates a new DefaultRuntime with the given configuration.
func NewDefaultRuntime(cfg RuntimeConfig) *DefaultRuntime {
	denyMap := make(map[SecurityProfile]bool)
	for _, p := range cfg.DenyUnsafeProfiles {
		denyMap[p] = true
	}

	return &DefaultRuntime{
		backends:           cfg.Backends,
		denyUnsafeProfiles: denyMap,
		defaultProfile:     cfg.DefaultProfile,
		logger:             cfg.Logger,
	}
}

// Execute implements the Runtime interface.
func (r *DefaultRuntime) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	// Check context first
	if ctx.Err() != nil {
		return ExecuteResult{}, ctx.Err()
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return ExecuteResult{}, err
	}

	// Determine profile
	profile := req.Profile
	if profile == "" {
		profile = r.defaultProfile
	}

	// Get backend for profile
	r.mu.RLock()
	backend, ok := r.backends[profile]
	isDenied := r.denyUnsafeProfiles[profile]
	r.mu.RUnlock()

	if !ok {
		return ExecuteResult{}, fmt.Errorf("%w: no backend for profile %q", ErrRuntimeUnavailable, profile)
	}

	// Check if unsafe backend is denied for this profile
	if isDenied && backend.Kind() == BackendUnsafeHost {
		return ExecuteResult{}, fmt.Errorf("%w: unsafe backend denied for profile %q", ErrBackendDenied, profile)
	}

	// Log execution start
	if r.logger != nil {
		r.logger.Info("executing code", "profile", profile, "backend", backend.Kind())
	}

	// Delegate to backend
	result, err := backend.Execute(ctx, req)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("execution failed", "profile", profile, "error", err)
		}
		return result, err
	}

	// If the backend did not populate tool calls but the gateway can, capture them.
	if len(result.ToolCalls) == 0 {
		if recorder, ok := req.Gateway.(toolCallRecorder); ok {
			result.ToolCalls = recorder.GetToolCalls()
		}
	}

	if r.logger != nil {
		r.logger.Info("execution completed", "profile", profile, "duration", result.Duration)
	}

	return result, nil
}

// RegisterBackend registers a backend for a security profile.
// This is thread-safe and can be called at runtime.
func (r *DefaultRuntime) RegisterBackend(profile SecurityProfile, backend Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.backends == nil {
		r.backends = make(map[SecurityProfile]Backend)
	}
	r.backends[profile] = backend
}

// UnregisterBackend removes a backend for a security profile.
// This is thread-safe and can be called at runtime.
func (r *DefaultRuntime) UnregisterBackend(profile SecurityProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.backends, profile)
}
