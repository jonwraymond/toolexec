package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonwraymond/toolfoundation/model"
)

// DefaultRunner is the standard Runner implementation.
// It uses the configured Index, resolvers, validators, and executors
// to resolve, validate, and execute tools.
type DefaultRunner struct {
	cfg Config
}

// NewRunner creates a new DefaultRunner with the given options.
// By default, validation is enabled for both input and output.
func NewRunner(opts ...ConfigOption) *DefaultRunner {
	cfg := Config{
		ValidateInput:  true, // Default on
		ValidateOutput: true, // Default on
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.applyDefaults()
	return &DefaultRunner{cfg: cfg}
}

// Run executes a single tool and returns the normalized result.
func (r *DefaultRunner) Run(ctx context.Context, toolID string, args map[string]any) (RunResult, error) {
	if err := ctx.Err(); err != nil {
		return RunResult{}, err
	}
	if toolID == "" {
		return RunResult{}, WrapError(toolID, nil, "validate_tool_id", ErrInvalidToolID)
	}
	// 1. Resolve tool + backends
	resolved, err := r.resolveTool(ctx, toolID)
	if err != nil {
		return RunResult{}, WrapError(toolID, nil, "resolve", err)
	}

	// 2. Select backend
	backend, err := r.selectBackend(resolved.backends)
	if err != nil {
		return RunResult{}, WrapError(toolID, nil, "select_backend", err)
	}

	// 3. Validate input
	if r.cfg.ValidateInput {
		if err := r.cfg.Validator.ValidateInput(&resolved.tool, args); err != nil {
			return RunResult{}, WrapError(toolID, &backend, "validate_input", fmt.Errorf("%w: %v", ErrValidation, err))
		}
	}

	// 4. Dispatch
	dispatchResult, err := r.dispatch(ctx, resolved.tool, backend, args)
	if err != nil {
		return RunResult{}, WrapError(toolID, &backend, "execute", fmt.Errorf("%w: %v", ErrExecution, err))
	}

	// 5. Normalize
	result := r.normalize(resolved.tool, backend, dispatchResult)

	// 6. Validate output
	if r.cfg.ValidateOutput {
		if err := r.cfg.Validator.ValidateOutput(&resolved.tool, result.Structured); err != nil {
			return RunResult{}, WrapError(toolID, &backend, "validate_output", fmt.Errorf("%w: %v", ErrOutputValidation, err))
		}
	}

	return result, nil
}

// RunWithProgress executes a tool and emits coarse progress updates.
func (r *DefaultRunner) RunWithProgress(ctx context.Context, toolID string, args map[string]any, onProgress ProgressCallback) (RunResult, error) {
	if onProgress != nil {
		onProgress(ProgressEvent{Progress: 0, Total: 1, Message: "started"})
	}

	result, err := r.Run(ctx, toolID, args)

	if onProgress != nil {
		msg := "completed"
		if err != nil {
			msg = "error"
		}
		onProgress(ProgressEvent{Progress: 1, Total: 1, Message: msg})
	}

	return result, err
}

// RunStream executes a tool with streaming support.
func (r *DefaultRunner) RunStream(ctx context.Context, toolID string, args map[string]any) (<-chan StreamEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if toolID == "" {
		return nil, WrapError(toolID, nil, "validate_tool_id", ErrInvalidToolID)
	}
	// 1. Resolve tool + backends
	resolved, err := r.resolveTool(ctx, toolID)
	if err != nil {
		return nil, WrapError(toolID, nil, "resolve", err)
	}

	// 2. Select backend
	backend, err := r.selectBackend(resolved.backends)
	if err != nil {
		return nil, WrapError(toolID, nil, "select_backend", err)
	}

	// 3. Validate input
	if r.cfg.ValidateInput {
		if err := r.cfg.Validator.ValidateInput(&resolved.tool, args); err != nil {
			return nil, WrapError(toolID, &backend, "validate_input", fmt.Errorf("%w: %v", ErrValidation, err))
		}
	}

	// 4. Dispatch stream
	rawChan, err := r.dispatchStream(ctx, resolved.tool, backend, args)
	if err != nil {
		return nil, WrapError(toolID, &backend, "stream", err)
	}
	if rawChan == nil {
		// Guard against executors returning (nil, nil), which would hang callers.
		return nil, WrapError(toolID, &backend, "stream", ErrStreamNotSupported)
	}

	// 5. Wrap channel to stamp ToolID on events when missing
	out := make(chan StreamEvent)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-rawChan:
				if !ok {
					return
				}
				if ev.ToolID == "" {
					ev.ToolID = toolID
				}
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// RunChain executes a sequence of tool steps.
func (r *DefaultRunner) RunChain(ctx context.Context, steps []ChainStep) (RunResult, []StepResult, error) {
	return r.runChainWithProgress(ctx, steps, nil)
}

// RunChainWithProgress executes a chain and emits progress updates.
func (r *DefaultRunner) RunChainWithProgress(ctx context.Context, steps []ChainStep, onProgress ProgressCallback) (RunResult, []StepResult, error) {
	return r.runChainWithProgress(ctx, steps, onProgress)
}

func (r *DefaultRunner) runChainWithProgress(ctx context.Context, steps []ChainStep, onProgress ProgressCallback) (RunResult, []StepResult, error) {
	if len(steps) == 0 {
		return RunResult{}, nil, nil
	}

	if onProgress != nil {
		onProgress(ProgressEvent{Progress: 0, Total: float64(len(steps)), Message: "started"})
	}

	var results []StepResult
	var previous any

	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			return RunResult{}, results, err
		}
		// Build args with previous injection
		args := r.buildChainArgs(step, previous)

		// Execute the step
		result, err := r.Run(ctx, step.ToolID, args)

		// Resolve backend for StepResult (we need to resolve again to get it)
		var backend model.ToolBackend
		if err == nil {
			backend = result.Backend
		} else {
			var toolErr *ToolError
			if errors.As(err, &toolErr) && toolErr.Backend != nil {
				backend = *toolErr.Backend
			}
		}

		stepResult := StepResult{
			ToolID:  step.ToolID,
			Backend: backend,
			Result:  result,
			Err:     err,
		}
		results = append(results, stepResult)

		if onProgress != nil {
			msg := "step_completed"
			if err != nil {
				msg = "step_error"
			}
			onProgress(ProgressEvent{
				Progress: float64(i + 1),
				Total:    float64(len(steps)),
				Message:  msg,
			})
		}

		// Stop on first error (v1 policy)
		if err != nil {
			return RunResult{}, results, err
		}

		// Update previous for next step
		previous = result.Structured
	}

	// Return the last successful result
	lastResult := results[len(results)-1].Result
	return lastResult, results, nil
}

// buildChainArgs builds the args map for a chain step.
// If UsePrevious is true, injects previous result at args["previous"].
func (r *DefaultRunner) buildChainArgs(step ChainStep, previous any) map[string]any {
	args := make(map[string]any)
	for k, v := range step.Args {
		args[k] = v
	}
	if step.UsePrevious {
		args["previous"] = previous
	}
	return args
}

// Ensure DefaultRunner implements Runner.
var _ Runner = (*DefaultRunner)(nil)
