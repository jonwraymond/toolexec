package code

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/jonwraymond/toolexec/run"
)

// Integration tests verify end-to-end behavior of the toolcode package.

func TestIntegration_SimpleExecution(t *testing.T) {
	// Engine that returns a simple value without using tools
	engine := &simpleEngine{
		value: map[string]any{"answer": 42},
	}

	cfg := Config{
		Index:           &mockIndex{},
		Docs:            &mockStore{},
		Run:             &mockRunner{},
		Engine:          engine,
		DefaultLanguage: "go",
		DefaultTimeout:  5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code: `__out := map[string]any{"answer": 42}`,
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have the value from the engine
	if result.Value == nil {
		t.Fatal("expected non-nil Value")
	}
	valueMap, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result.Value)
	}
	if valueMap["answer"] != 42 {
		t.Errorf("expected answer=42, got %v", valueMap["answer"])
	}

	// No tool calls
	if len(result.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(result.ToolCalls))
	}

	// Duration should be positive
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative duration, got %d", result.DurationMs)
	}
}

func TestIntegration_RunSingleTool(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: "tool output",
			Backend: model.ToolBackend{
				Kind: model.BackendKindMCP,
			},
		},
	}

	// Engine that runs a single tool
	engine := &toolUsingEngine{
		toolID: "ns:mytool",
		args:   map[string]any{"input": "test"},
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            runner,
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "result := RunTool(ctx, 'ns:mytool', args)",
		Language: "go",
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly one tool call
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}

	record := result.ToolCalls[0]
	if record.ToolID != "ns:mytool" {
		t.Errorf("expected ToolID 'ns:mytool', got %q", record.ToolID)
	}
	if record.Args["input"] != "test" {
		t.Errorf("expected Args[input]='test', got %v", record.Args)
	}
	if record.Structured != "tool output" {
		t.Errorf("expected Structured 'tool output', got %v", record.Structured)
	}
	if record.BackendKind != "mcp" {
		t.Errorf("expected BackendKind 'mcp', got %q", record.BackendKind)
	}
	if record.DurationMs < 0 {
		t.Errorf("expected non-negative DurationMs, got %d", record.DurationMs)
	}
}

func TestIntegration_RunChain(t *testing.T) {
	runner := &mockRunner{
		chainResult: run.RunResult{
			Structured: "final result",
		},
		chainSteps: []run.StepResult{
			{
				ToolID: "tool1",
				Result: run.RunResult{
					Structured: "step1 result",
					Backend:    model.ToolBackend{Kind: model.BackendKindLocal},
				},
			},
			{
				ToolID: "tool2",
				Result: run.RunResult{
					Structured: "step2 result",
					Backend:    model.ToolBackend{Kind: model.BackendKindMCP},
				},
			},
		},
	}

	// Engine that runs a chain
	engine := &chainUsingEngine{
		steps: []run.ChainStep{
			{ToolID: "tool1", Args: map[string]any{"a": 1}},
			{ToolID: "tool2", Args: map[string]any{"b": 2}, UsePrevious: true},
		},
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            runner,
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "chain code",
		Language: "go",
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 tool call records (one per step)
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}

	if result.ToolCalls[0].ToolID != "tool1" {
		t.Errorf("expected first record ToolID 'tool1', got %q", result.ToolCalls[0].ToolID)
	}
	if result.ToolCalls[0].Structured != "step1 result" {
		t.Errorf("expected first record Structured 'step1 result', got %v", result.ToolCalls[0].Structured)
	}
	if result.ToolCalls[1].ToolID != "tool2" {
		t.Errorf("expected second record ToolID 'tool2', got %q", result.ToolCalls[1].ToolID)
	}
	if result.ToolCalls[1].BackendKind != "mcp" {
		t.Errorf("expected second record BackendKind 'mcp', got %q", result.ToolCalls[1].BackendKind)
	}
}

func TestIntegration_ToolError_Captured(t *testing.T) {
	toolErr := errors.New("tool execution failed: permission denied")
	runner := &mockRunner{
		runErr: toolErr,
	}

	engine := &toolUsingEngine{
		toolID: "failing-tool",
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            runner,
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
	}
	result, err := exec.ExecuteCode(ctx, params)
	// The executor should complete (engine succeeded), but tool call has error
	if err != nil {
		t.Fatalf("unexpected executor error: %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}

	record := result.ToolCalls[0]
	if record.Error != "tool execution failed: permission denied" {
		t.Errorf("expected Error 'tool execution failed: permission denied', got %q", record.Error)
	}
	if record.ErrorOp != "run" {
		t.Errorf("expected ErrorOp 'run', got %q", record.ErrorOp)
	}
}

func TestIntegration_MaxToolCalls_Exceeded(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{},
	}

	// Engine that tries to make 3 tool calls
	engine := &multiToolEngine{
		toolIDs: []string{"tool1", "tool2", "tool3"},
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            runner,
		Engine:         engine,
		MaxToolCalls:   2, // Only allow 2
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
	}
	_, err = exec.ExecuteCode(ctx, params)
	// Engine should receive ErrLimitExceeded on 3rd call
	if err == nil {
		t.Fatal("expected error due to max tool calls exceeded")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
}

func TestIntegration_Timeout_Exceeded(t *testing.T) {
	engine := &slowEngine{
		delay: 5 * time.Second,
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  50 * time.Millisecond, // Very short timeout
	}

	start := time.Now()
	_, err = exec.ExecuteCode(ctx, params)
	elapsed := time.Since(start)

	// Should timeout quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected quick timeout, took %v", elapsed)
	}

	if err == nil {
		t.Fatal("expected error due to timeout")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
}

func TestIntegration_Println_Captured(t *testing.T) {
	engine := &printingEngine{
		messages: []string{"line 1", "line 2", "line 3"},
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            &mockRunner{},
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "Println calls",
		Language: "go",
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedStdout := "line 1\nline 2\nline 3\n"
	if result.Stdout != expectedStdout {
		t.Errorf("expected Stdout %q, got %q", expectedStdout, result.Stdout)
	}
}

func TestIntegration_OutVariable(t *testing.T) {
	// Engine that sets __out via result
	engine := &simpleEngine{
		value: map[string]any{
			"status": "success",
			"count":  42,
			"items":  []string{"a", "b", "c"},
		},
	}

	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            &mockRunner{},
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "__out := result",
		Language: "go",
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the Value matches what engine returned
	valueMap, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result.Value)
	}
	if valueMap["status"] != "success" {
		t.Errorf("expected status='success', got %v", valueMap["status"])
	}
	if valueMap["count"] != 42 {
		t.Errorf("expected count=42, got %v", valueMap["count"])
	}
}

// Helper engines for integration tests

// simpleEngine returns a preset value
type simpleEngine struct {
	value any
}

func (e *simpleEngine) Execute(_ context.Context, _ ExecuteParams, _ Tools) (ExecuteResult, error) {
	return ExecuteResult{Value: e.value}, nil
}

// chainUsingEngine runs a chain of tools
type chainUsingEngine struct {
	steps []run.ChainStep
}

func (e *chainUsingEngine) Execute(ctx context.Context, _ ExecuteParams, tools Tools) (ExecuteResult, error) {
	result, _, err := tools.RunChain(ctx, e.steps)
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{Value: result.Structured}, nil
}

// multiToolEngine runs multiple individual tools
type multiToolEngine struct {
	toolIDs []string
}

func (e *multiToolEngine) Execute(ctx context.Context, _ ExecuteParams, tools Tools) (ExecuteResult, error) {
	for _, id := range e.toolIDs {
		_, err := tools.RunTool(ctx, id, nil)
		if err != nil {
			return ExecuteResult{}, err
		}
	}
	return ExecuteResult{Value: "done"}, nil
}
