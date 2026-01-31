package code

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Executor is the main entry point for executing code snippets.
// It orchestrates configuration, limits, and result collection.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines; deadline exceeded is wrapped with ErrLimitExceeded.
// - Errors: configuration failures return ErrConfiguration; execution failures propagate.
// - Ownership: params are read-only; returned ExecuteResult is caller-owned.
type Executor interface {
	// ExecuteCode runs a code snippet with the given parameters.
	// It applies configuration defaults, enforces limits, and collects
	// tool call traces and output.
	ExecuteCode(ctx context.Context, params ExecuteParams) (ExecuteResult, error)
}

// DefaultExecutor is the standard implementation of Executor.
type DefaultExecutor struct {
	cfg Config
}

// NewDefaultExecutor creates a new DefaultExecutor with the given configuration.
// Returns ErrConfiguration if any required field is missing.
func NewDefaultExecutor(cfg Config) (*DefaultExecutor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &DefaultExecutor{cfg: cfg}, nil
}

// ExecuteCode runs a code snippet with the given parameters.
func (e *DefaultExecutor) ExecuteCode(ctx context.Context, params ExecuteParams) (ExecuteResult, error) {
	// Apply defaults from config
	if params.Language == "" {
		params.Language = e.cfg.DefaultLanguage
	}
	if params.Timeout == 0 {
		params.Timeout = e.cfg.DefaultTimeout
	}

	// Resolve MaxToolCalls (params capped by config)
	maxCalls := params.MaxToolCalls
	if e.cfg.MaxToolCalls > 0 {
		if maxCalls == 0 || maxCalls > e.cfg.MaxToolCalls {
			maxCalls = e.cfg.MaxToolCalls
		}
	}

	// Create tools environment
	tools := newTools(&e.cfg, maxCalls, e.cfg.MaxChainSteps)

	// Create context with timeout
	var cancel context.CancelFunc
	if params.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, params.Timeout)
		defer cancel()
	}

	start := time.Now()
	result, err := e.cfg.Engine.Execute(ctx, params, tools)
	duration := time.Since(start).Milliseconds()

	// Collect captured data from tools
	result.ToolCalls = tools.GetToolCalls()
	result.Stdout = tools.GetStdout()
	result.DurationMs = duration

	// Log execution summary if logger present
	if e.cfg.Logger != nil {
		e.cfg.Logger.Logf("executed %d tool calls in %dms", len(result.ToolCalls), duration)
	}

	// Wrap timeout errors
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return result, fmt.Errorf("%w: timeout after %v", ErrLimitExceeded, params.Timeout)
	}

	return result, err
}
