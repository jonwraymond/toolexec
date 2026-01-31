package toolcodeengine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/code"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

func newEngine(t *testing.T, runtime runtime.Runtime, profile runtime.SecurityProfile) *Engine {
	t.Helper()
	engine, err := New(Config{
		Runtime: runtime,
		Profile: profile,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return engine
}

// mockRuntime implements runtime.Runtime for testing
type mockRuntime struct {
	result      runtime.ExecuteResult
	err         error
	capturedReq runtime.ExecuteRequest
}

func (m *mockRuntime) Execute(_ context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	m.capturedReq = req
	if m.err != nil {
		return runtime.ExecuteResult{}, m.err
	}
	return m.result, nil
}

// mockTools implements code.Tools for testing
type mockTools struct {
	searchResults []index.Summary
	namespaces    []string
	toolDoc       tooldoc.ToolDoc
	examples      []tooldoc.ToolExample
	runResult     run.RunResult
	chainResult   run.RunResult
	stepResults   []run.StepResult
}

func (m *mockTools) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return m.searchResults, nil
}

func (m *mockTools) ListNamespaces(_ context.Context) ([]string, error) {
	return m.namespaces, nil
}

func (m *mockTools) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return m.toolDoc, nil
}

func (m *mockTools) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return m.examples, nil
}

func (m *mockTools) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return m.runResult, nil
}

func (m *mockTools) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return m.chainResult, m.stepResults, nil
}

func (m *mockTools) Println(_ ...any) {
	// Mock implementation
}

// TestEngineImplementsInterface verifies Engine satisfies code.Engine
func TestEngineImplementsInterface(t *testing.T) {
	t.Helper()
	var _ code.Engine = (*Engine)(nil)
}

func TestEngineExecute(t *testing.T) {
	rt := &mockRuntime{
		result: runtime.ExecuteResult{
			Value:    "result",
			Stdout:   "output",
			Stderr:   "",
			Duration: 100 * time.Millisecond,
		},
	}

	engine := newEngine(t, rt, runtime.ProfileDev)

	ctx := context.Background()
	params := code.ExecuteParams{
		Language:     "go",
		Code:         `__out = "hello"`,
		Timeout:      30 * time.Second,
		MaxToolCalls: 10,
	}
	tools := &mockTools{}

	result, err := engine.Execute(ctx, params, tools)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Value != "result" {
		t.Errorf("Execute().Value = %v, want %v", result.Value, "result")
	}
	if result.Stdout != "output" {
		t.Errorf("Execute().Stdout = %q, want %q", result.Stdout, "output")
	}
}

func TestEngineExecuteMapsParams(t *testing.T) {
	rt := &mockRuntime{
		result: runtime.ExecuteResult{},
	}

	engine := newEngine(t, rt, runtime.ProfileStandard)

	ctx := context.Background()
	params := code.ExecuteParams{
		Language:     "go",
		Code:         "test code",
		Timeout:      5 * time.Second,
		MaxToolCalls: 20,
	}
	tools := &mockTools{}

	_, _ = engine.Execute(ctx, params, tools)

	if rt.capturedReq.Language != "go" {
		t.Errorf("ExecuteRequest.Language = %q, want %q", rt.capturedReq.Language, "go")
	}
	if rt.capturedReq.Code != "test code" {
		t.Errorf("ExecuteRequest.Code = %q, want %q", rt.capturedReq.Code, "test code")
	}
	if rt.capturedReq.Timeout != 5*time.Second {
		t.Errorf("ExecuteRequest.Timeout = %v, want %v", rt.capturedReq.Timeout, 5*time.Second)
	}
	if rt.capturedReq.Limits.MaxToolCalls != 20 {
		t.Errorf("ExecuteRequest.Limits.MaxToolCalls = %d, want %d", rt.capturedReq.Limits.MaxToolCalls, 20)
	}
	if rt.capturedReq.Profile != runtime.ProfileStandard {
		t.Errorf("ExecuteRequest.Profile = %v, want %v", rt.capturedReq.Profile, runtime.ProfileStandard)
	}
}

func TestEngineExecuteTimeoutError(t *testing.T) {
	rt := &mockRuntime{
		err: runtime.ErrTimeout,
	}

	engine := newEngine(t, rt, runtime.ProfileDev)

	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}
	tools := &mockTools{}

	_, err := engine.Execute(ctx, params, tools)
	if err == nil {
		t.Error("Execute() should return error for timeout")
	}
	// Should map to code.ErrLimitExceeded
	if !errors.Is(err, code.ErrLimitExceeded) {
		t.Errorf("Execute() error = %v, want wrapped %v", err, code.ErrLimitExceeded)
	}
}

func TestEngineExecuteResourceLimitError(t *testing.T) {
	rt := &mockRuntime{
		err: runtime.ErrResourceLimit,
	}

	engine := newEngine(t, rt, runtime.ProfileDev)

	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}
	tools := &mockTools{}

	_, err := engine.Execute(ctx, params, tools)
	if err == nil {
		t.Error("Execute() should return error for resource limit")
	}
	if !errors.Is(err, code.ErrLimitExceeded) {
		t.Errorf("Execute() error = %v, want wrapped %v", err, code.ErrLimitExceeded)
	}
}

func TestEngineExecuteSandboxViolationError(t *testing.T) {
	rt := &mockRuntime{
		err: runtime.ErrSandboxViolation,
	}

	engine := newEngine(t, rt, runtime.ProfileDev)

	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}
	tools := &mockTools{}

	_, err := engine.Execute(ctx, params, tools)
	if err == nil {
		t.Error("Execute() should return error for sandbox violation")
	}
	if !errors.Is(err, code.ErrCodeExecution) {
		t.Errorf("Execute() error = %v, want wrapped %v", err, code.ErrCodeExecution)
	}
}
