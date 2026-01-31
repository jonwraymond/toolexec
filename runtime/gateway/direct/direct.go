// Package direct provides a gateway that implements ToolGateway
// by directly delegating to toolindex, tooldocs, and toolrun components.
// This gateway runs in-process with no isolation boundary.
package direct

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// Errors for limit enforcement
var (
	// ErrToolCallLimitExceeded is returned when MaxToolCalls is exceeded.
	ErrToolCallLimitExceeded = errors.New("tool call limit exceeded")

	// ErrChainStepLimitExceeded is returned when MaxChainSteps is exceeded.
	ErrChainStepLimitExceeded = errors.New("chain step limit exceeded")
)

// Config configures a direct gateway.
type Config struct {
	// Index is the tool index for search and lookup.
	Index index.Index

	// Docs is the documentation store.
	Docs tooldoc.Store

	// Runner is the tool execution runner.
	Runner run.Runner

	// MaxToolCalls limits the total number of tool invocations.
	// Zero means unlimited.
	MaxToolCalls int

	// MaxChainSteps limits the number of steps in a chain.
	// Zero means unlimited.
	MaxChainSteps int
}

// Gateway implements ToolGateway by directly delegating to
// the index, docs, and runner components.
type Gateway struct {
	index         index.Index
	docs          tooldoc.Store
	runner        run.Runner
	maxToolCalls  int
	maxChainSteps int

	mu        sync.Mutex
	callCount int
	toolCalls []runtime.ToolCallRecord
}

// New creates a new direct gateway with the given configuration.
func New(cfg Config) *Gateway {
	return &Gateway{
		index:         cfg.Index,
		docs:          cfg.Docs,
		runner:        cfg.Runner,
		maxToolCalls:  cfg.MaxToolCalls,
		maxChainSteps: cfg.MaxChainSteps,
	}
}

// SearchTools delegates to the index.
func (g *Gateway) SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return g.index.Search(query, limit)
}

// ListNamespaces delegates to the index.
func (g *Gateway) ListNamespaces(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return g.index.ListNamespaces()
}

// DescribeTool delegates to the docs store.
func (g *Gateway) DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	if ctx.Err() != nil {
		return tooldoc.ToolDoc{}, ctx.Err()
	}
	return g.docs.DescribeTool(id, level)
}

// ListToolExamples delegates to the docs store.
func (g *Gateway) ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return g.docs.ListExamples(id, maxExamples)
}

// RunTool delegates to the runner and records the call.
func (g *Gateway) RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error) {
	if ctx.Err() != nil {
		return run.RunResult{}, ctx.Err()
	}

	// Check tool call limit
	g.mu.Lock()
	if g.maxToolCalls > 0 && g.callCount >= g.maxToolCalls {
		g.mu.Unlock()
		return run.RunResult{}, fmt.Errorf("%w: max %d calls exceeded", ErrToolCallLimitExceeded, g.maxToolCalls)
	}
	g.callCount++
	g.mu.Unlock()

	// Execute
	start := time.Now()
	result, err := g.runner.Run(ctx, id, args)
	duration := time.Since(start)

	// Record the call
	record := runtime.ToolCallRecord{
		ToolID:   id,
		Duration: duration,
	}
	if err != nil {
		record.ErrorOp = "run"
	}
	if result.Backend.Kind != "" {
		record.BackendKind = string(result.Backend.Kind)
	}

	g.mu.Lock()
	g.toolCalls = append(g.toolCalls, record)
	g.mu.Unlock()

	return result, err
}

// RunChain delegates to the runner and records the calls.
func (g *Gateway) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	if ctx.Err() != nil {
		return run.RunResult{}, nil, ctx.Err()
	}

	// Handle empty/nil steps
	if len(steps) == 0 {
		return run.RunResult{}, nil, nil
	}

	// Check chain step limit
	if g.maxChainSteps > 0 && len(steps) > g.maxChainSteps {
		return run.RunResult{}, nil, fmt.Errorf("%w: max %d steps exceeded (got %d)",
			ErrChainStepLimitExceeded, g.maxChainSteps, len(steps))
	}

	// Check if we have enough room for all steps
	reserved := len(steps)
	g.mu.Lock()
	if g.maxToolCalls > 0 && g.callCount+reserved > g.maxToolCalls {
		g.mu.Unlock()
		return run.RunResult{}, nil, fmt.Errorf("%w: would exceed max %d calls",
			ErrToolCallLimitExceeded, g.maxToolCalls)
	}
	g.callCount += reserved
	g.mu.Unlock()

	// Execute
	start := time.Now()
	result, stepResults, err := g.runner.RunChain(ctx, steps)
	duration := time.Since(start)

	executed := len(stepResults)
	if executed == 0 && err == nil {
		executed = reserved
	}
	if executed > reserved {
		executed = reserved
	}

	// Adjust reserved count if fewer steps actually executed.
	if executed < reserved {
		g.mu.Lock()
		g.callCount -= reserved - executed
		if g.callCount < 0 {
			g.callCount = 0
		}
		g.mu.Unlock()
	}

	// Record the calls (approximate duration per step)
	if executed == 0 {
		return result, stepResults, err
	}
	stepDuration := duration / time.Duration(executed)

	g.mu.Lock()
	for i, step := range steps[:executed] {
		record := runtime.ToolCallRecord{
			ToolID:   step.ToolID,
			Duration: stepDuration,
		}
		if i < len(stepResults) && stepResults[i].Err != nil {
			record.ErrorOp = "chain"
		}
		if i < len(stepResults) && stepResults[i].Backend.Kind != "" {
			record.BackendKind = string(stepResults[i].Backend.Kind)
		}
		g.toolCalls = append(g.toolCalls, record)
	}
	g.mu.Unlock()

	return result, stepResults, err
}

// GetToolCalls returns a copy of all recorded tool calls.
func (g *Gateway) GetToolCalls() []runtime.ToolCallRecord {
	g.mu.Lock()
	defer g.mu.Unlock()
	result := make([]runtime.ToolCallRecord, len(g.toolCalls))
	copy(result, g.toolCalls)
	return result
}

// Reset clears recorded tool calls and resets the call counter.
func (g *Gateway) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.callCount = 0
	g.toolCalls = nil
}
