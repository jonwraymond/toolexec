package code

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// Tools is the metatool environment exposed to code snippets during execution.
// It provides functions for discovering, documenting, and executing tools.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods must honor cancellation/deadlines and return ctx.Err() when canceled.
// - Errors: execution failures propagate underlying errors (e.g., ErrLimitExceeded).
// - Ownership: args are read-only; returned slices/results are caller-owned snapshots.
// - Nil/zero: empty IDs return ErrNotFound/ErrInvalidToolID downstream; nil args treated as empty.
type Tools interface {
	// SearchTools searches for tools matching the query, returning up to limit results.
	SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error)

	// ListNamespaces returns all available tool namespaces.
	ListNamespaces(ctx context.Context) ([]string, error)

	// DescribeTool returns documentation for a tool at the specified detail level.
	DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error)

	// ListToolExamples returns up to maxExamples usage examples for a tool.
	ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error)

	// RunTool executes a single tool and returns the result.
	// Each call is recorded in the tool call trace.
	RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error)

	// RunChain executes a sequence of tool calls, where each step can
	// optionally use the previous step's result via UsePrevious.
	// Each step is recorded in the tool call trace.
	RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error)

	// Println writes output to the captured stdout buffer.
	Println(args ...any)
}

// toolsImpl is the internal implementation of Tools that tracks tool calls
// and enforces limits.
type toolsImpl struct {
	index         index.Index
	docs          tooldoc.Store
	runner        run.Runner
	logger        Logger
	toolCalls     []ToolCallRecord
	stdout        strings.Builder
	maxToolCalls  int
	maxChainSteps int
	callCount     int
}

// newTools creates a new Tools implementation with the given configuration
// and limits. If a limit is 0, it is treated as unlimited.
func newTools(cfg *Config, maxToolCalls int, maxChainSteps int) *toolsImpl {
	return &toolsImpl{
		index:         cfg.Index,
		docs:          cfg.Docs,
		runner:        cfg.Run,
		logger:        cfg.Logger,
		maxToolCalls:  maxToolCalls,
		maxChainSteps: maxChainSteps,
	}
}

func (t *toolsImpl) SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return t.index.Search(query, limit)
}

func (t *toolsImpl) ListNamespaces(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return t.index.ListNamespaces()
}

func (t *toolsImpl) DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	if err := ctx.Err(); err != nil {
		return tooldoc.ToolDoc{}, err
	}
	return t.docs.DescribeTool(id, level)
}

func (t *toolsImpl) ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return t.docs.ListExamples(id, maxExamples)
}

func (t *toolsImpl) RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error) {
	if t.maxToolCalls > 0 && t.callCount >= t.maxToolCalls {
		return run.RunResult{}, fmt.Errorf("%w: max tool calls (%d) exceeded",
			ErrLimitExceeded, t.maxToolCalls)
	}
	t.callCount++

	start := time.Now()
	result, err := t.runner.Run(ctx, id, args)
	duration := time.Since(start).Milliseconds()

	record := ToolCallRecord{
		ToolID:     id,
		Args:       deepCopyArgs(args),
		DurationMs: duration,
	}
	if err != nil {
		record.Error = err.Error()
		record.ErrorOp = "run"
		var toolErr *run.ToolError
		if errors.As(err, &toolErr) && toolErr.Backend != nil && toolErr.Backend.Kind != "" {
			record.BackendKind = string(toolErr.Backend.Kind)
		}
	} else {
		record.Structured = result.Structured
		record.BackendKind = string(result.Backend.Kind)
	}
	t.toolCalls = append(t.toolCalls, record)

	return result, err
}

func (t *toolsImpl) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	if t.maxChainSteps > 0 && len(steps) > t.maxChainSteps {
		return run.RunResult{}, nil, fmt.Errorf("%w: max chain steps (%d) exceeded (got %d)",
			ErrLimitExceeded, t.maxChainSteps, len(steps))
	}

	// Check if we have enough room for all steps
	if t.maxToolCalls > 0 {
		stepsNeeded := len(steps)
		if t.callCount+stepsNeeded > t.maxToolCalls {
			return run.RunResult{}, nil, fmt.Errorf("%w: max tool calls (%d) exceeded (need %d, have %d remaining)",
				ErrLimitExceeded, t.maxToolCalls, stepsNeeded, t.maxToolCalls-t.callCount)
		}
	}

	start := time.Now()
	result, stepResults, err := t.runner.RunChain(ctx, steps)
	totalDuration := time.Since(start).Milliseconds()

	executed := len(stepResults)
	if executed == 0 && err == nil {
		executed = len(steps)
	}
	if executed > len(steps) {
		executed = len(steps)
	}
	denom := int64(executed)
	if denom == 0 {
		denom = 1
	}

	// Record each executed step, reconstructing the effective args
	// (including previous injection) and normalizing to MCP-native shapes.
	var previous any
	for i := 0; i < executed; i++ {
		step := steps[i]
		t.callCount++

		effectiveArgs := make(map[string]any, len(step.Args)+1)
		for k, v := range step.Args {
			effectiveArgs[k] = v
		}
		if step.UsePrevious {
			effectiveArgs["previous"] = previous
		}

		record := ToolCallRecord{
			ToolID:     step.ToolID,
			Args:       deepCopyArgs(effectiveArgs),
			DurationMs: totalDuration / denom,
		}

		if i < len(stepResults) {
			sr := stepResults[i]
			if sr.Backend.Kind != "" {
				record.BackendKind = string(sr.Backend.Kind)
			} else if sr.Result.Backend.Kind != "" {
				record.BackendKind = string(sr.Result.Backend.Kind)
			}
			if sr.Err != nil {
				record.Error = sr.Err.Error()
				record.ErrorOp = "chain"
			} else {
				record.Structured = sr.Result.Structured
				previous = sr.Result.Structured
			}
		}

		t.toolCalls = append(t.toolCalls, record)
	}

	return result, stepResults, err
}

func (t *toolsImpl) Println(args ...any) {
	fmt.Fprintln(&t.stdout, args...)
}

// GetToolCalls returns a copy of all recorded tool calls.
func (t *toolsImpl) GetToolCalls() []ToolCallRecord {
	return append([]ToolCallRecord(nil), t.toolCalls...)
}

// GetStdout returns the captured stdout output.
func (t *toolsImpl) GetStdout() string {
	return t.stdout.String()
}

// deepCopyArgs performs a deep copy of an args map.
// It normalizes typed maps/slices into MCP-native shapes (map[string]any, []any).
func deepCopyArgs(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	result := make(map[string]any, len(args))
	for k, v := range args {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue recursively copies a value into MCP-native shapes.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return deepCopyArgs(val)
	case []any:
		return deepCopySlice(val)
	case map[string]string:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = v
		}
		return out
	case map[string]int:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = v
		}
		return out
	case map[string]float64:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = v
		}
		return out
	case map[string]bool:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = v
		}
		return out
	case []string:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = v
		}
		return out
	case []int:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = v
		}
		return out
	case []float64:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = v
		}
		return out
	case []bool:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = v
		}
		return out
	case string, bool, float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return val
	case json.Number:
		return val
	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return nil
			}
			return deepCopyValue(rv.Elem().Interface())
		}
		if out, ok := deepCopyViaJSON(val); ok {
			return out
		}
		return val
	}
}

func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = deepCopyValue(v)
	}
	return out
}

func deepCopyViaJSON(v any) (any, bool) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	return out, true
}
