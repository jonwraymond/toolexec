package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
)

func TestDispatch_MCP(t *testing.T) {
	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolResult = testMCPResult("success")

	runner := NewRunner(WithMCPExecutor(mcpExec))

	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	args := map[string]any{"input": "value"}

	result, err := runner.dispatch(context.Background(), tool, backend, args)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	if result.mcpResult == nil {
		t.Error("mcpResult should not be nil")
	}

	if mcpExec.LastServerName != "server1" {
		t.Errorf("LastServerName = %q, want %q", mcpExec.LastServerName, "server1")
	}
	if mcpExec.LastParams.Name != "mytool" {
		t.Errorf("LastParams.Name = %q, want %q", mcpExec.LastParams.Name, "mytool")
	}
}

func TestDispatch_MCP_NotConfigured(t *testing.T) {
	runner := NewRunner() // No MCP executor

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should return error when MCP not configured")
	}
}

func TestDispatch_MCP_ExecutorError(t *testing.T) {
	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolErr = errors.New("connection failed")

	runner := NewRunner(WithMCPExecutor(mcpExec))

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should propagate executor error")
	}
}

func TestDispatch_MCP_UsesToolName(t *testing.T) {
	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolResult = testMCPResult("ok")

	runner := NewRunner(WithMCPExecutor(mcpExec))

	// Tool with namespace - MCP should use Name, not ToolID()
	tool := testToolWithNamespace("myns", "mytool")
	backend := testMCPBackend("server1")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	// Should use Name (not "myns:mytool")
	if mcpExec.LastParams.Name != "mytool" {
		t.Errorf("LastParams.Name = %q, want %q (not ToolID)", mcpExec.LastParams.Name, "mytool")
	}
}

func TestDispatch_Provider(t *testing.T) {
	provExec := newMockProviderExecutor()
	provExec.CallToolResult = map[string]any{"result": "success"}

	runner := NewRunner(WithProviderExecutor(provExec))

	tool := testTool("mytool")
	backend := testProviderBackend("myprovider", "provider-tool-id")
	args := map[string]any{"input": "value"}

	result, err := runner.dispatch(context.Background(), tool, backend, args)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	if result.structured == nil {
		t.Error("structured should not be nil")
	}

	if provExec.LastProviderID != "myprovider" {
		t.Errorf("LastProviderID = %q, want %q", provExec.LastProviderID, "myprovider")
	}
	if provExec.LastToolID != "provider-tool-id" {
		t.Errorf("LastToolID = %q, want %q", provExec.LastToolID, "provider-tool-id")
	}
}

func TestDispatch_Provider_NotConfigured(t *testing.T) {
	runner := NewRunner() // No provider executor

	tool := testTool("mytool")
	backend := testProviderBackend("myprovider", "tool1")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should return error when provider not configured")
	}
}

func TestDispatch_Provider_UsesBackendIDs(t *testing.T) {
	provExec := newMockProviderExecutor()
	provExec.CallToolResult = "ok"

	runner := NewRunner(WithProviderExecutor(provExec))

	tool := testTool("mytool")
	backend := testProviderBackend("specific-provider", "specific-tool")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	// Should use backend's ProviderID and ToolID
	if provExec.LastProviderID != "specific-provider" {
		t.Errorf("LastProviderID = %q, want %q", provExec.LastProviderID, "specific-provider")
	}
	if provExec.LastToolID != "specific-tool" {
		t.Errorf("LastToolID = %q, want %q", provExec.LastToolID, "specific-tool")
	}
}

func TestDispatch_Local(t *testing.T) {
	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"handled": true}, nil
	})

	runner := NewRunner(WithLocalRegistry(localReg))

	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")

	result, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	if result.structured == nil {
		t.Error("structured should not be nil")
	}

	m, ok := result.structured.(map[string]any)
	if !ok {
		t.Fatalf("structured type = %T, want map[string]any", result.structured)
	}
	if m["handled"] != true {
		t.Errorf("structured[handled] = %v, want true", m["handled"])
	}
}

func TestDispatch_Local_HandlerNotFound(t *testing.T) {
	localReg := newMockLocalRegistry() // No handlers

	runner := NewRunner(WithLocalRegistry(localReg))

	tool := testTool("mytool")
	backend := testLocalBackend("nonexistent")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should return error when handler not found")
	}
}

func TestDispatch_Local_HandlerNil(t *testing.T) {
	localReg := newMockLocalRegistry()
	localReg.handlers["nil-handler"] = nil

	runner := NewRunner(WithLocalRegistry(localReg))

	tool := testTool("mytool")
	backend := testLocalBackend("nil-handler")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should return error when handler is nil")
	}
}

func TestDispatch_Local_HandlerError(t *testing.T) {
	localReg := newMockLocalRegistry()
	localReg.Register("failing", func(_ context.Context, _ map[string]any) (any, error) {
		return nil, errors.New("handler failed")
	})

	runner := NewRunner(WithLocalRegistry(localReg))

	tool := testTool("mytool")
	backend := testLocalBackend("failing")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should propagate handler error")
	}
}

func TestDispatch_Local_UsesBackendName(t *testing.T) {
	called := false
	localReg := newMockLocalRegistry()
	localReg.Register("specific-handler", func(_ context.Context, _ map[string]any) (any, error) {
		called = true
		return "ok", nil
	})

	runner := NewRunner(WithLocalRegistry(localReg))

	tool := testTool("mytool")
	backend := testLocalBackend("specific-handler")

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}

	if !called {
		t.Error("specific-handler should have been called")
	}
}

func TestDispatch_UnknownBackend(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := model.ToolBackend{Kind: "unknown"}

	_, err := runner.dispatch(context.Background(), tool, backend, nil)
	if err == nil {
		t.Error("dispatch() should return error for unknown backend kind")
	}
}
