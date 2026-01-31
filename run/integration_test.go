package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Integration tests verify end-to-end scenarios with real (non-mock) components
// working together.

func TestIntegration_LocalTool_WithValidation(t *testing.T) {
	// Create a real index
	idx := newMockIndex()

	// Register a tool with input/output schemas
	tool := model.Tool{
		Tool: mcp.Tool{
			Name: "echo",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{"type": "string"},
				},
				"required": []any{"message"},
			},
		},
	}

	backend := testLocalBackend("echo-handler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["echo"] = backend

	// Create a local registry with the handler
	localReg := newMockLocalRegistry()
	localReg.Register("echo-handler", func(_ context.Context, args map[string]any) (any, error) {
		msg, _ := args["message"].(string)
		return map[string]any{"echoed": msg}, nil
	})

	// Create runner with real default validator and validation enabled
	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		// Use default validator (model.NewDefaultValidator())
		WithValidation(true, false), // Input validation on
	)

	// Execute with valid input
	result, err := runner.Run(context.Background(), "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	m, ok := result.Structured.(map[string]any)
	if !ok {
		t.Fatalf("Structured type = %T", result.Structured)
	}
	if m["echoed"] != "hello" {
		t.Errorf("echoed = %v, want 'hello'", m["echoed"])
	}
}

func TestIntegration_MCPTool_WithNormalization(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("mcp-tool")
	backend := testMCPBackend("test-server")
	mustRegisterTool(t, idx, tool, backend)

	// Mock MCP executor returning structured content
	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolResult = testMCPResultStructured(map[string]any{
		"status": "success",
		"count":  42,
		"items":  []any{"a", "b", "c"},
		"nested": map[string]any{"key": "value"},
	})

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	result, err := runner.Run(context.Background(), "mcp-tool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// MCPResult should be preserved
	if result.MCPResult == nil {
		t.Error("MCPResult should not be nil")
	}

	// Structured should be the structured content
	m, ok := result.Structured.(map[string]any)
	if !ok {
		t.Fatalf("Structured type = %T", result.Structured)
	}
	if m["status"] != "success" {
		t.Errorf("status = %v, want 'success'", m["status"])
	}
	if m["count"] != 42 {
		t.Errorf("count = %v, want 42", m["count"])
	}
}

func TestIntegration_Chain_DataPassing(t *testing.T) {
	idx := newMockIndex()

	// Tool 1: Generate a list
	tool1 := testTool("generate")
	backend1 := testLocalBackend("generate-handler")
	mustRegisterTool(t, idx, tool1, backend1)
	idx.DefaultBackends["generate"] = backend1

	// Tool 2: Transform the list
	tool2 := testTool("transform")
	backend2 := testLocalBackend("transform-handler")
	mustRegisterTool(t, idx, tool2, backend2)
	idx.DefaultBackends["transform"] = backend2

	// Tool 3: Summarize
	tool3 := testTool("summarize")
	backend3 := testLocalBackend("summarize-handler")
	mustRegisterTool(t, idx, tool3, backend3)
	idx.DefaultBackends["summarize"] = backend3

	localReg := newMockLocalRegistry()

	localReg.Register("generate-handler", func(_ context.Context, args map[string]any) (any, error) {
		count, _ := args["count"].(float64)
		items := make([]string, int(count))
		for i := range items {
			items[i] = "item"
		}
		return map[string]any{"items": items}, nil
	})

	localReg.Register("transform-handler", func(_ context.Context, args map[string]any) (any, error) {
		prev, _ := args["previous"].(map[string]any)
		items, _ := prev["items"].([]string)

		// Transform: uppercase
		transformed := make([]string, len(items))
		for i, item := range items {
			transformed[i] = "TRANSFORMED-" + item
		}
		return map[string]any{"items": transformed}, nil
	})

	localReg.Register("summarize-handler", func(_ context.Context, args map[string]any) (any, error) {
		prev, _ := args["previous"].(map[string]any)
		items, _ := prev["items"].([]string)
		return map[string]any{
			"count":   len(items),
			"summary": "Processed items",
		}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "generate", Args: map[string]any{"count": float64(3)}},
		{ToolID: "transform", UsePrevious: true},
		{ToolID: "summarize", UsePrevious: true},
	}

	final, results, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// Should have 3 results
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}

	// Final should be summary
	m, ok := final.Structured.(map[string]any)
	if !ok {
		t.Fatalf("final.Structured type = %T", final.Structured)
	}
	if m["summary"] != "Processed items" {
		t.Errorf("summary = %v", m["summary"])
	}
}

func TestIntegration_FallbackResolvers(t *testing.T) {
	// Test using only fallback resolvers (no index)
	tool := testTool("dynamic-tool")
	backends := []model.ToolBackend{testLocalBackend("dynamic-handler")}

	toolResolver := func(id string) (*model.Tool, error) {
		if id == "dynamic-tool" {
			return &tool, nil
		}
		return nil, errors.New("not found")
	}

	backendsResolver := func(id string) ([]model.ToolBackend, error) {
		if id == "dynamic-tool" {
			return backends, nil
		}
		return nil, errors.New("not found")
	}

	localReg := newMockLocalRegistry()
	localReg.Register("dynamic-handler", func(_ context.Context, _ map[string]any) (any, error) {
		return "dynamically resolved", nil
	})

	runner := NewRunner(
		WithToolResolver(toolResolver),
		WithBackendsResolver(backendsResolver),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	result, err := runner.Run(context.Background(), "dynamic-tool", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Structured != "dynamically resolved" {
		t.Errorf("Structured = %v", result.Structured)
	}
}

func TestIntegration_CustomBackendSelector(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("multi-backend")
	mcpBackend := testMCPBackend("server1")
	localBackend := testLocalBackend("handler1")

	mustRegisterTool(t, idx, tool, mcpBackend)
	idx.Backends["multi-backend"] = []model.ToolBackend{mcpBackend, localBackend}

	// Custom selector that always picks MCP even when local is available
	customSelector := func(backends []model.ToolBackend) model.ToolBackend {
		for _, b := range backends {
			if b.Kind == model.BackendKindMCP {
				return b
			}
		}
		return backends[0]
	}

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolResult = testMCPResult("from mcp")

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithBackendSelector(customSelector),
		WithValidation(false, false),
	)

	result, err := runner.Run(context.Background(), "multi-backend", nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should use MCP despite local being available
	if result.Backend.Kind != model.BackendKindMCP {
		t.Errorf("Backend.Kind = %q, want mcp", result.Backend.Kind)
	}
}

func TestIntegration_ErrorPropagation(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("failing-tool")
	backend := testLocalBackend("failing-handler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["failing-tool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("failing-handler", func(_ context.Context, _ map[string]any) (any, error) {
		return nil, errors.New("intentional failure")
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	_, err := runner.Run(context.Background(), "failing-tool", nil)
	if err == nil {
		t.Fatal("Run() should return error")
	}

	// Error should be wrapped in ToolError
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("error type = %T, want *ToolError", err)
	}

	// Should be an execution error
	if !errors.Is(err, ErrExecution) {
		t.Errorf("error = %v, want ErrExecution", err)
	}

	// Should contain tool context
	if toolErr.ToolID != "failing-tool" {
		t.Errorf("ToolID = %q, want 'failing-tool'", toolErr.ToolID)
	}
	if toolErr.Backend == nil || toolErr.Backend.Kind != model.BackendKindLocal {
		t.Error("Backend should be set to local backend")
	}
}
