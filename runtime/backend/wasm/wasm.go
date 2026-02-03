// Package wasm provides a backend that executes code compiled to WebAssembly.
// Provides strong in-process isolation; requires constrained SDK surface.
package wasm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for WASM backend operations.
var (
	// ErrWASMRuntimeNotAvailable is returned when WASM runtime is not available.
	ErrWASMRuntimeNotAvailable = errors.New("wasm runtime not available")

	// ErrModuleCompilationFailed is returned when WASM module compilation fails.
	ErrModuleCompilationFailed = errors.New("wasm module compilation failed")

	// ErrModuleExecutionFailed is returned when WASM module execution fails.
	ErrModuleExecutionFailed = errors.New("wasm module execution failed")

	// ErrUnsupportedLanguage is returned when the language cannot be compiled to WASM.
	ErrUnsupportedLanguage = errors.New("language not supported for wasm compilation")

	// ErrClientNotConfigured is returned when no Runner is configured.
	ErrClientNotConfigured = errors.New("wasm client not configured")

	// ErrInvalidModule is returned when the WASM module is invalid.
	ErrInvalidModule = errors.New("invalid wasm module")

	// ErrMemoryExceeded is returned when the memory limit is exceeded.
	ErrMemoryExceeded = errors.New("memory limit exceeded")

	// ErrFuelExhausted is returned when the fuel limit is exhausted.
	ErrFuelExhausted = errors.New("fuel limit exhausted")
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

// Config configures a WASM backend.
type Config struct {
	// Runtime specifies the WASM runtime to use.
	// Options: wasmtime, wasmer, wazero
	// Default: wazero
	Runtime string

	// MaxMemoryPages is the maximum memory pages (64KB each).
	// Default: 256 (16MB)
	MaxMemoryPages int

	// EnableWASI enables WASI (WebAssembly System Interface).
	// Default: true
	EnableWASI bool

	// AllowedHostFunctions lists host functions the WASM module can call.
	AllowedHostFunctions []string

	// Client is the WASM runner implementation.
	// If nil, Execute() returns ErrClientNotConfigured.
	Client Runner

	// ModuleLoader optionally pre-compiles modules.
	// If nil, modules are compiled on-demand.
	ModuleLoader ModuleLoader

	// HealthChecker optionally verifies runtime health.
	// If nil, health checks are skipped.
	HealthChecker HealthChecker

	// Logger is an optional logger for backend events.
	Logger Logger
}

// Backend executes code compiled to WebAssembly.
type Backend struct {
	runtime              string
	maxMemoryPages       int
	enableWASI           bool
	allowedHostFunctions []string
	client               Runner
	moduleLoader         ModuleLoader
	healthChecker        HealthChecker
	logger               Logger
}

// New creates a new WASM backend with the given configuration.
func New(cfg Config) *Backend {
	runtime := cfg.Runtime
	if runtime == "" {
		runtime = "wazero"
	}

	maxMemoryPages := cfg.MaxMemoryPages
	if maxMemoryPages <= 0 {
		maxMemoryPages = 256 // 16MB
	}

	return &Backend{
		runtime:              runtime,
		maxMemoryPages:       maxMemoryPages,
		enableWASI:           cfg.EnableWASI,
		allowedHostFunctions: cfg.AllowedHostFunctions,
		client:               cfg.Client,
		moduleLoader:         cfg.ModuleLoader,
		healthChecker:        cfg.HealthChecker,
		logger:               cfg.Logger,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendWASM
}

// Execute runs code compiled to WebAssembly.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	// Check client is configured
	if b.client == nil {
		return runtime.ExecuteResult{}, ErrClientNotConfigured
	}

	// Apply timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check context before proceeding
	select {
	case <-ctx.Done():
		return runtime.ExecuteResult{}, ctx.Err()
	default:
	}

	start := time.Now()

	// Get security profile
	profile := req.Profile
	if profile == "" {
		profile = runtime.ProfileStandard
	}

	// Optional health check
	if b.healthChecker != nil {
		if err := b.healthChecker.Ping(ctx); err != nil {
			return runtime.ExecuteResult{}, fmt.Errorf("%w: %v", ErrWASMRuntimeNotAvailable, err)
		}
	}

	// Build WASM spec from request
	spec := b.buildSpec(req, profile)

	// Log execution
	if b.logger != nil {
		b.logger.Info("executing in WASM sandbox",
			"profile", profile,
			"runtime", b.runtime,
			"enableWASI", b.enableWASI,
			"memoryPages", b.maxMemoryPages)
	}

	// Execute via client
	wasmResult, err := b.client.Run(ctx, spec)
	if err != nil {
		return runtime.ExecuteResult{
			Duration: time.Since(start),
			Backend:  b.backendInfo(profile),
		}, err
	}

	// Convert to ExecuteResult
	return runtime.ExecuteResult{
		Value:    extractOutValue(wasmResult.Stdout),
		Stdout:   wasmResult.Stdout,
		Stderr:   wasmResult.Stderr,
		Duration: wasmResult.Duration,
		Backend:  b.backendInfo(profile),
		LimitsEnforced: runtime.LimitsEnforced{
			Timeout:    true,
			Memory:     spec.Resources.MemoryPages > 0,
			CPU:        spec.Resources.FuelLimit > 0, // Fuel serves as CPU limiting
			Pids:       false,                        // WASM doesn't have process model
			ToolCalls:  true,                         // Enforced by gateway
			ChainSteps: true,                         // Enforced by gateway
		},
	}, nil
}

// buildSpec creates a Spec from an ExecuteRequest.
func (b *Backend) buildSpec(req runtime.ExecuteRequest, profile runtime.SecurityProfile) Spec {
	memoryPages := uint32(0)
	if b.maxMemoryPages > 0 {
		// #nosec G115 -- b.maxMemoryPages is clamped to uint32 below.
		maxPages := uint64(b.maxMemoryPages)
		memoryPages = clampUint32(maxPages)
	}

	spec := Spec{
		// Note: Module bytes would need to be provided by the execution framework
		// This is typically handled by a code compiler step before execution
		Timeout: req.Timeout,
		Security: SecuritySpec{
			EnableWASI:           b.enableWASI,
			AllowedHostFunctions: b.allowedHostFunctions,
			EnableNetwork:        false, // Always disabled for sandbox
			EnableClock:          true,  // Allow timing operations
		},
		Resources: ResourceSpec{
			MemoryPages: memoryPages,
		},
		Labels: map[string]string{
			"runtime.profile": string(profile),
			"runtime.backend": string(runtime.BackendWASM),
		},
	}

	// Apply profile-specific settings
	switch profile {
	case runtime.ProfileDev:
		// Dev mode: more permissive
		spec.Security.EnableNetwork = false // Still no network in WASM
		spec.Security.EnableClock = true

	case runtime.ProfileStandard:
		// Standard: default restrictions
		spec.Security.EnableNetwork = false
		spec.Security.EnableClock = true

	case runtime.ProfileHardened:
		// Hardened: maximum restrictions
		spec.Security.EnableNetwork = false
		spec.Security.EnableClock = false // Disable clock for timing attacks
		spec.Security.AllowedHostFunctions = nil
	}

	// Apply resource limits from request
	if req.Limits.MemoryBytes > 0 {
		// Convert bytes to 64KB pages
		pages := req.Limits.MemoryBytes / (64 * 1024)
		if pages > 0 {
			// #nosec G115 -- pages is positive and clamped to uint32 range.
			spec.Resources.MemoryPages = clampUint32(uint64(pages))
		}
	}

	return spec
}

// backendInfo returns BackendInfo for the given profile.
func (b *Backend) backendInfo(profile runtime.SecurityProfile) runtime.BackendInfo {
	return runtime.BackendInfo{
		Kind:      runtime.BackendWASM,
		Readiness: runtime.ReadinessBeta,
		Details: map[string]any{
			"runtime":        b.runtime,
			"profile":        string(profile),
			"maxMemoryPages": b.maxMemoryPages,
			"enableWASI":     b.enableWASI,
		},
	}
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

func clampUint32(value uint64) uint32 {
	if value > math.MaxUint32 {
		return math.MaxUint32
	}
	// #nosec G115 -- value bounded to MaxUint32.
	return uint32(value)
}

var _ runtime.Backend = (*Backend)(nil)
