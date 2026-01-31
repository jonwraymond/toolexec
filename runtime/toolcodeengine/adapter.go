// Package toolcodeengine provides an adapter that implements code.Engine
// using runtime.Runtime for execution.
package toolcodeengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonwraymond/toolexec/code"
	"github.com/jonwraymond/toolexec/runtime"
)

// Config configures an Engine.
type Config struct {
	// Runtime is the runtime.Runtime to use for execution.
	Runtime runtime.Runtime

	// Profile is the security profile to use for execution.
	Profile runtime.SecurityProfile
}

// Engine implements code.Engine using a runtime.Runtime backend.
type Engine struct {
	runtime runtime.Runtime
	profile runtime.SecurityProfile
}

// New creates a new Engine with the given configuration.
func New(cfg Config) (*Engine, error) {
	if cfg.Runtime == nil {
		return nil, runtime.ErrRuntimeUnavailable
	}

	profile := cfg.Profile
	if profile == "" {
		profile = runtime.ProfileStandard
	}

	return &Engine{
		runtime: cfg.Runtime,
		profile: profile,
	}, nil
}

// Execute implements code.Engine by delegating to the underlying runtime.
func (e *Engine) Execute(ctx context.Context, params code.ExecuteParams, tools code.Tools) (code.ExecuteResult, error) {
	if e.runtime == nil {
		return code.ExecuteResult{}, runtime.ErrRuntimeUnavailable
	}

	// Wrap Tools into a ToolGateway
	gateway := WrapTools(tools)

	// Map code.ExecuteParams to runtime.ExecuteRequest
	req := runtime.ExecuteRequest{
		Language: params.Language,
		Code:     params.Code,
		Timeout:  params.Timeout,
		Limits: runtime.Limits{
			MaxToolCalls: params.MaxToolCalls,
		},
		Profile: e.profile,
		Gateway: gateway,
	}

	// Execute via the runtime
	result, err := e.runtime.Execute(ctx, req)

	// Map errors
	if err != nil {
		return mapResult(result), mapError(err)
	}

	return mapResult(result), nil
}

// mapResult converts runtime.ExecuteResult to code.ExecuteResult.
func mapResult(r runtime.ExecuteResult) code.ExecuteResult {
	toolCalls := make([]code.ToolCallRecord, len(r.ToolCalls))
	for i, tc := range r.ToolCalls {
		toolCalls[i] = code.ToolCallRecord{
			ToolID:      tc.ToolID,
			BackendKind: tc.BackendKind,
			DurationMs:  tc.Duration.Milliseconds(),
			ErrorOp:     tc.ErrorOp,
		}
	}

	return code.ExecuteResult{
		Value:      r.Value,
		Stdout:     r.Stdout,
		Stderr:     r.Stderr,
		ToolCalls:  toolCalls,
		DurationMs: r.Duration.Milliseconds(),
	}
}

// mapError converts toolruntime errors to toolcode errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Map timeout and resource limit errors to ErrLimitExceeded
	if errors.Is(err, runtime.ErrTimeout) {
		return fmt.Errorf("%w: %v", code.ErrLimitExceeded, err)
	}
	if errors.Is(err, runtime.ErrResourceLimit) {
		return fmt.Errorf("%w: %v", code.ErrLimitExceeded, err)
	}

	// Map sandbox violation to ErrCodeExecution
	if errors.Is(err, runtime.ErrSandboxViolation) {
		return fmt.Errorf("%w: %v", code.ErrCodeExecution, err)
	}

	// Return other errors as-is
	return err
}
