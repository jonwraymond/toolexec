package exec

import (
	"context"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// Exec is the unified facade for tool execution.
// It combines discovery, execution, and result handling into a single API.
type Exec struct {
	index  index.Index
	docs   tooldoc.Store
	runner run.Runner
	opts   Options
}

// New creates a new Exec instance with the given options.
func New(opts Options) (*Exec, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	opts.applyDefaults()

	// Build local registry from handlers map
	var localReg run.LocalRegistry
	if len(opts.LocalHandlers) > 0 {
		localReg = newMapLocalRegistry(opts.LocalHandlers)
	}

	// Create runner with configuration
	runner := run.NewRunner(
		run.WithIndex(opts.Index),
		run.WithLocalRegistry(localReg),
		run.WithMCPExecutor(opts.MCPExecutor),
		run.WithProviderExecutor(opts.ProviderExecutor),
		run.WithValidation(opts.ValidateInput, opts.ValidateOutput),
	)

	return &Exec{
		index:  opts.Index,
		docs:   opts.Docs,
		runner: runner,
		opts:   opts,
	}, nil
}

// RunTool executes a single tool by ID and returns the result.
func (e *Exec) RunTool(ctx context.Context, toolID string, args map[string]any) (Result, error) {
	start := time.Now()

	runResult, err := e.runner.Run(ctx, toolID, args)
	duration := time.Since(start)

	if err != nil {
		return Result{
			ToolID:   toolID,
			Duration: duration,
			Error:    err,
		}, err
	}

	return Result{
		Value:    runResult.Structured,
		ToolID:   toolID,
		Duration: duration,
	}, nil
}

// RunChain executes a sequence of tools.
// Returns the final result, a slice of step results, and any error.
func (e *Exec) RunChain(ctx context.Context, steps []Step) (Result, []StepResult, error) {
	start := time.Now()

	// Convert exec.Step to run.ChainStep
	chainSteps := make([]run.ChainStep, len(steps))
	for i, s := range steps {
		chainSteps[i] = run.ChainStep{
			ToolID:      s.ToolID,
			Args:        s.Args,
			UsePrevious: s.UsePrevious,
		}
	}

	runResult, runSteps, err := e.runner.RunChain(ctx, chainSteps)
	duration := time.Since(start)

	// Convert run.StepResult to exec.StepResult
	stepResults := make([]StepResult, len(runSteps))
	for i, rs := range runSteps {
		stepResults[i] = StepResult{
			StepIndex: i,
			ToolID:    rs.ToolID,
			Args:      chainSteps[i].Args,
			Value:     rs.Result.Structured,
			Duration:  0, // run.StepResult doesn't track duration per step
			Skipped:   false,
		}
		if rs.Err != nil {
			stepResults[i].Error = rs.Err
		}
	}

	if err != nil {
		return Result{
			ToolID:   "",
			Duration: duration,
			Error:    err,
		}, stepResults, err
	}

	// Final result comes from the last step
	finalToolID := ""
	if len(steps) > 0 {
		finalToolID = steps[len(steps)-1].ToolID
	}

	return Result{
		Value:    runResult.Structured,
		ToolID:   finalToolID,
		Duration: duration,
	}, stepResults, nil
}

// SearchTools finds tools matching a query.
func (e *Exec) SearchTools(ctx context.Context, query string, limit int) ([]ToolSummary, error) {
	_ = ctx // reserved for future context-aware search
	return e.index.Search(query, limit)
}

// GetToolDoc retrieves tool documentation at the specified detail level.
func (e *Exec) GetToolDoc(ctx context.Context, toolID string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	_ = ctx // reserved for future context-aware doc retrieval
	return e.docs.DescribeTool(toolID, level)
}

// Index returns the underlying tool index.
// This allows advanced usage patterns like direct tool registration.
func (e *Exec) Index() index.Index {
	return e.index
}

// DocStore returns the underlying documentation store.
func (e *Exec) DocStore() tooldoc.Store {
	return e.docs
}

// mapLocalRegistry implements run.LocalRegistry using a map of handlers.
type mapLocalRegistry struct {
	handlers map[string]Handler
}

// newMapLocalRegistry creates a LocalRegistry from a map of handlers.
func newMapLocalRegistry(handlers map[string]Handler) *mapLocalRegistry {
	return &mapLocalRegistry{handlers: handlers}
}

// Get returns the handler for the given name.
func (r *mapLocalRegistry) Get(name string) (run.LocalHandler, bool) {
	h, ok := r.handlers[name]
	if !ok {
		return nil, false
	}
	// Convert exec.Handler to run.LocalHandler (same signature)
	return run.LocalHandler(h), true
}
