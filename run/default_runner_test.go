package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
)

// -----------------------------------------------------------------------------
// Run Tests
// -----------------------------------------------------------------------------

func TestRun_Success_MCP(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolResult = testMCPResultStructured(map[string]any{"result": "success"})

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false), // Disable validation for this test
	)

	result, err := runner.Run(context.Background(), "mytool", map[string]any{"input": "value"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Tool.Name != "mytool" {
		t.Errorf("Tool.Name = %q, want %q", result.Tool.Name, "mytool")
	}
	if result.Backend.Kind != model.BackendKindMCP {
		t.Errorf("Backend.Kind = %q, want %q", result.Backend.Kind, model.BackendKindMCP)
	}
	if result.MCPResult == nil {
		t.Error("MCPResult should not be nil")
	}
}

func TestRun_EmptyToolID(t *testing.T) {
	runner := NewRunner(WithIndex(newMockIndex()))

	_, err := runner.Run(context.Background(), "", map[string]any{"input": "value"})
	if err == nil {
		t.Fatal("Run() should fail when toolID is empty")
	}
	if !errors.Is(err, ErrInvalidToolID) {
		t.Errorf("Run() error = %v, want ErrInvalidToolID", err)
	}
}

func TestRun_Success_Provider(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testProviderBackend("myprovider", "tool1")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	provExec := newMockProviderExecutor()
	provExec.CallToolResult = map[string]any{"result": "from provider"}

	runner := NewRunner(
		WithIndex(idx),
		WithProviderExecutor(provExec),
		WithValidation(false, false),
	)

	result, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Backend.Kind != model.BackendKindProvider {
		t.Errorf("Backend.Kind = %q, want %q", result.Backend.Kind, model.BackendKindProvider)
	}
}

func TestRun_Success_Local(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "from local"}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	result, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Backend.Kind != model.BackendKindLocal {
		t.Errorf("Backend.Kind = %q, want %q", result.Backend.Kind, model.BackendKindLocal)
	}
}

func TestRun_InputValidation_Pass(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return "ok", nil
	})

	mockVal := newMockValidator()
	// No error configured - validation passes

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(true, false),
	)

	_, err := runner.Run(context.Background(), "mytool", map[string]any{"input": "value"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if mockVal.ValidateInputCalls != 1 {
		t.Errorf("ValidateInput was called %d times, want 1", mockVal.ValidateInputCalls)
	}
}

func TestRun_InputValidation_Fail(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	mockVal := newMockValidator()
	mockVal.ValidateInputErr = errors.New("input invalid")

	runner := NewRunner(
		WithIndex(idx),
		WithValidator(mockVal),
		WithValidation(true, false),
	)

	_, err := runner.Run(context.Background(), "mytool", map[string]any{})
	if err == nil {
		t.Error("Run() should fail when input validation fails")
	}

	if !errors.Is(err, ErrValidation) {
		t.Errorf("Run() error = %v, want ErrValidation", err)
	}
}

func TestRun_InputValidation_Disabled(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return "ok", nil
	})

	mockVal := newMockValidator()
	mockVal.ValidateInputErr = errors.New("would fail if called")

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(false, false), // Validation disabled
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if mockVal.ValidateInputCalls != 0 {
		t.Errorf("ValidateInput was called %d times, want 0", mockVal.ValidateInputCalls)
	}
}

func TestRun_ContextCancelled_BeforeExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	called := false
	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		called = true
		return map[string]any{"result": "ok"}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	_, err := runner.Run(ctx, "mytool", nil)
	if err == nil {
		t.Fatal("Run() should return error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run() error = %v, want context.Canceled", err)
	}
	if called {
		t.Error("Run() should not invoke handler when context is cancelled")
	}
}

func TestRun_OutputValidation_Pass(t *testing.T) {
	idx := newMockIndex()
	tool := testToolWithOutputSchema("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "valid"}, nil
	})

	mockVal := newMockValidator()
	// No error - validation passes

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(false, true), // Only output validation
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if mockVal.OutputCalls != 1 {
		t.Errorf("ValidateOutput was called %d times, want 1", mockVal.OutputCalls)
	}
}

func TestRun_OutputValidation_Fail(t *testing.T) {
	idx := newMockIndex()
	tool := testToolWithOutputSchema("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "invalid"}, nil
	})

	mockVal := newMockValidator()
	mockVal.ValidateOutputErr = errors.New("output invalid")

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(false, true),
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err == nil {
		t.Error("Run() should fail when output validation fails")
	}

	if !errors.Is(err, ErrOutputValidation) {
		t.Errorf("Run() error = %v, want ErrOutputValidation", err)
	}
}

func TestRun_OutputValidation_Disabled(t *testing.T) {
	idx := newMockIndex()
	tool := testToolWithOutputSchema("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return "result", nil
	})

	mockVal := newMockValidator()
	mockVal.ValidateOutputErr = errors.New("would fail if called")

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(false, false), // Disabled
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if mockVal.OutputCalls != 0 {
		t.Errorf("ValidateOutput was called %d times, want 0", mockVal.OutputCalls)
	}
}

func TestRun_OutputValidation_NoSchema(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool") // No output schema
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return "result", nil
	})

	mockVal := newMockValidator()
	// ValidateOutput returns nil when no OutputSchema (per model behavior)

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidator(mockVal),
		WithValidation(false, true),
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRun_ResolutionError(t *testing.T) {
	runner := NewRunner() // No index or resolvers

	_, err := runner.Run(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("Run() should fail when tool not found")
	}

	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("Run() error = %v, want ErrToolNotFound", err)
	}
}

func TestRun_ExecutionError(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolErr = errors.New("execution failed")

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err == nil {
		t.Error("Run() should fail when execution fails")
	}

	if !errors.Is(err, ErrExecution) {
		t.Errorf("Run() error = %v, want ErrExecution", err)
	}
}

func TestRun_ToolErrorContext(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolErr = errors.New("mcp failed")

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	_, err := runner.Run(context.Background(), "mytool", nil)
	if err == nil {
		t.Fatal("Run() should return error")
	}

	// Check that error contains tool context
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("error should be *ToolError, got %T", err)
	}

	if toolErr.ToolID != "mytool" {
		t.Errorf("ToolError.ToolID = %q, want %q", toolErr.ToolID, "mytool")
	}
	if toolErr.Backend == nil {
		t.Error("ToolError.Backend should not be nil")
	}
}
