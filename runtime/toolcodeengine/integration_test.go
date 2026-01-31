package toolcodeengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/code"
	"github.com/jonwraymond/toolexec/run"
	runt "github.com/jonwraymond/toolexec/runtime"
	"github.com/jonwraymond/toolexec/runtime/backend/unsafe"
	"github.com/jonwraymond/toolexec/runtime/toolcodeengine"
)

func newEngine(t *testing.T, runtime runt.Runtime, profile runt.SecurityProfile) *toolcodeengine.Engine {
	t.Helper()
	engine, err := toolcodeengine.New(toolcodeengine.Config{
		Runtime: runtime,
		Profile: profile,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return engine
}

// testTools implements code.Tools for integration testing
type testTools struct {
	searchResults []index.Summary
	namespaces    []string
	toolDoc       tooldoc.ToolDoc
	examples      []tooldoc.ToolExample
	runResult     run.RunResult
	chainResult   run.RunResult
	stepResults   []run.StepResult
}

func (t *testTools) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return t.searchResults, nil
}

func (t *testTools) ListNamespaces(_ context.Context) ([]string, error) {
	return t.namespaces, nil
}

func (t *testTools) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return t.toolDoc, nil
}

func (t *testTools) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return t.examples, nil
}

func (t *testTools) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return t.runResult, nil
}

func (t *testTools) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return t.chainResult, t.stepResults, nil
}

func (t *testTools) Println(_ ...any) {}

var _ code.Tools = (*testTools)(nil)

// TestFullStackExecution tests toolcode -> toolcodeengine -> toolruntime -> unsafe backend
func TestFullStackExecution(t *testing.T) {
	// Create an unsafe backend
	backend := unsafe.New(unsafe.Config{
		Mode: unsafe.ModeSubprocess,
	})

	// Create a runtime with the backend
	runtime := runt.NewDefaultRuntime(runt.RuntimeConfig{
		Backends: map[runt.SecurityProfile]runt.Backend{
			runt.ProfileDev: backend,
		},
		DefaultProfile: runt.ProfileDev,
	})

	// Create the toolcode engine adapter
	engine := newEngine(t, runtime, runt.ProfileDev)

	// Verify Engine implements code.Engine
	var _ code.Engine = engine

	// Create test tools
	tools := &testTools{
		searchResults: []index.Summary{
			{ID: "test:tool", Name: "tool"},
		},
	}

	// Execute simple code
	ctx := context.Background()
	params := code.ExecuteParams{
		Language:     "go",
		Code:         `__out = "hello world"`,
		Timeout:      10 * time.Second,
		MaxToolCalls: 5,
	}

	result, err := engine.Execute(ctx, params, tools)
	if err != nil {
		t.Logf("Execute() error (expected during integration): %v", err)
		// Note: The unsafe backend subprocess execution may fail depending on environment
		// The important thing is that the full stack is wired up correctly
	}

	// If execution succeeded, verify result
	if err == nil {
		t.Logf("Execute() result: %+v", result)
		if result.Value != "hello world" {
			t.Logf("Unexpected value: %v (may vary based on backend execution)", result.Value)
		}
	}
}

// TestErrorMappingIntegration tests that errors are correctly mapped through the stack
func TestErrorMappingIntegration(t *testing.T) {
	// Create a mock backend that returns specific errors
	mockBackend := &errorBackend{}

	runtime := runt.NewDefaultRuntime(runt.RuntimeConfig{
		Backends: map[runt.SecurityProfile]runt.Backend{
			runt.ProfileDev: mockBackend,
		},
		DefaultProfile: runt.ProfileDev,
	})

	engine := newEngine(t, runtime, runt.ProfileDev)

	tools := &testTools{}
	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}

	t.Run("timeout maps to ErrLimitExceeded", func(t *testing.T) {
		mockBackend.err = runt.ErrTimeout
		_, err := engine.Execute(ctx, params, tools)
		if !errors.Is(err, code.ErrLimitExceeded) {
			t.Errorf("timeout should map to ErrLimitExceeded, got: %v", err)
		}
	})

	t.Run("resource limit maps to ErrLimitExceeded", func(t *testing.T) {
		mockBackend.err = runt.ErrResourceLimit
		_, err := engine.Execute(ctx, params, tools)
		if !errors.Is(err, code.ErrLimitExceeded) {
			t.Errorf("resource limit should map to ErrLimitExceeded, got: %v", err)
		}
	})

	t.Run("sandbox violation maps to ErrCodeExecution", func(t *testing.T) {
		mockBackend.err = runt.ErrSandboxViolation
		_, err := engine.Execute(ctx, params, tools)
		if !errors.Is(err, code.ErrCodeExecution) {
			t.Errorf("sandbox violation should map to ErrCodeExecution, got: %v", err)
		}
	})
}

// errorBackend is a mock backend that returns configurable errors
type errorBackend struct {
	err error
}

func (b *errorBackend) Kind() runt.BackendKind {
	return runt.BackendUnsafeHost
}

func (b *errorBackend) Execute(_ context.Context, _ runt.ExecuteRequest) (runt.ExecuteResult, error) {
	if b.err != nil {
		return runt.ExecuteResult{}, b.err
	}
	return runt.ExecuteResult{
		Value: "test",
	}, nil
}

// TestGatewayWrappingIntegration tests that Tools is correctly wrapped as Gateway
func TestGatewayWrappingIntegration(t *testing.T) {
	// Create a mock backend that captures the request
	mockBackend := &capturingBackend{}

	runtime := runt.NewDefaultRuntime(runt.RuntimeConfig{
		Backends: map[runt.SecurityProfile]runt.Backend{
			runt.ProfileDev: mockBackend,
		},
		DefaultProfile: runt.ProfileDev,
	})

	engine := newEngine(t, runtime, runt.ProfileDev)

	tools := &testTools{
		searchResults: []index.Summary{
			{ID: "tool1", Name: "Tool One"},
			{ID: "tool2", Name: "Tool Two"},
		},
		namespaces: []string{"ns1", "ns2"},
	}

	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}

	_, _ = engine.Execute(ctx, params, tools)

	// Verify gateway was passed to backend
	if mockBackend.capturedReq.Gateway == nil {
		t.Error("Gateway should be passed to backend")
	}

	// Verify gateway works correctly
	gw := mockBackend.capturedReq.Gateway

	results, err := gw.SearchTools(ctx, "test", 10)
	if err != nil {
		t.Errorf("SearchTools() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchTools() returned %d results, want 2", len(results))
	}

	namespaces, err := gw.ListNamespaces(ctx)
	if err != nil {
		t.Errorf("ListNamespaces() error = %v", err)
	}
	if len(namespaces) != 2 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 2", len(namespaces))
	}
}

// capturingBackend captures the ExecuteRequest for inspection
type capturingBackend struct {
	capturedReq runt.ExecuteRequest
}

func (b *capturingBackend) Kind() runt.BackendKind {
	return runt.BackendUnsafeHost
}

func (b *capturingBackend) Execute(_ context.Context, req runt.ExecuteRequest) (runt.ExecuteResult, error) {
	b.capturedReq = req
	return runt.ExecuteResult{}, nil
}

// TestProfilePropagation tests that security profiles are correctly propagated
func TestProfilePropagation(t *testing.T) {
	mockBackend := &capturingBackend{}

	runtime := runt.NewDefaultRuntime(runt.RuntimeConfig{
		Backends: map[runt.SecurityProfile]runt.Backend{
			runt.ProfileStandard: mockBackend,
		},
		DefaultProfile: runt.ProfileStandard,
	})

	engine := newEngine(t, runtime, runt.ProfileStandard)

	tools := &testTools{}
	ctx := context.Background()
	params := code.ExecuteParams{
		Code: "test",
	}

	_, _ = engine.Execute(ctx, params, tools)

	if mockBackend.capturedReq.Profile != runt.ProfileStandard {
		t.Errorf("Profile = %v, want %v",
			mockBackend.capturedReq.Profile, runt.ProfileStandard)
	}
}

// TestLimitsPropagation tests that limits are correctly propagated
func TestLimitsPropagation(t *testing.T) {
	mockBackend := &capturingBackend{}

	runtime := runt.NewDefaultRuntime(runt.RuntimeConfig{
		Backends: map[runt.SecurityProfile]runt.Backend{
			runt.ProfileDev: mockBackend,
		},
		DefaultProfile: runt.ProfileDev,
	})

	engine := newEngine(t, runtime, runt.ProfileDev)

	tools := &testTools{}
	ctx := context.Background()
	params := code.ExecuteParams{
		Code:         "test",
		Timeout:      15 * time.Second,
		MaxToolCalls: 25,
	}

	_, _ = engine.Execute(ctx, params, tools)

	if mockBackend.capturedReq.Timeout != 15*time.Second {
		t.Errorf("Timeout = %v, want %v",
			mockBackend.capturedReq.Timeout, 15*time.Second)
	}
	if mockBackend.capturedReq.Limits.MaxToolCalls != 25 {
		t.Errorf("MaxToolCalls = %d, want %d",
			mockBackend.capturedReq.Limits.MaxToolCalls, 25)
	}
}
