package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func TestResolveTool_ViaIndex(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	runner := NewRunner(WithIndex(idx))

	resolved, err := runner.resolveTool(context.Background(), "mytool")
	if err != nil {
		t.Fatalf("resolveTool() error = %v", err)
	}

	if resolved.tool.Name != "mytool" {
		t.Errorf("tool.Name = %q, want %q", resolved.tool.Name, "mytool")
	}
	if len(resolved.backends) == 0 {
		t.Error("backends should not be empty")
	}
}

func TestResolveTool_ViaIndex_WithAllBackends(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	mcpBackend := testMCPBackend("server1")
	localBackend := testLocalBackend("handler1")

	// Register same tool with multiple backends
	mustRegisterTool(t, idx, tool, mcpBackend)
	idx.Backends["mytool"] = append(idx.Backends["mytool"], localBackend)

	runner := NewRunner(WithIndex(idx))

	resolved, err := runner.resolveTool(context.Background(), "mytool")
	if err != nil {
		t.Fatalf("resolveTool() error = %v", err)
	}

	if len(resolved.backends) != 2 {
		t.Errorf("len(backends) = %d, want 2", len(resolved.backends))
	}
}

func TestResolveTool_GetAllBackendsUnexpectedError(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	idx.GetAllBackendsErr = errors.New("boom")

	runner := NewRunner(WithIndex(idx))

	_, err := runner.resolveTool(context.Background(), "mytool")
	if err == nil {
		t.Fatal("resolveTool() should return error when GetAllBackends fails unexpectedly")
	}
	if !errors.Is(err, idx.GetAllBackendsErr) {
		t.Errorf("resolveTool() error = %v, want %v", err, idx.GetAllBackendsErr)
	}
}

func TestResolveTool_ViaFallbackResolvers(t *testing.T) {
	tool := testTool("resolved-tool")
	backends := []model.ToolBackend{testLocalBackend("handler1")}

	toolResolver := func(id string) (*model.Tool, error) {
		if id == "resolved-tool" {
			return &tool, nil
		}
		return nil, errors.New("not found")
	}

	backendsResolver := func(id string) ([]model.ToolBackend, error) {
		if id == "resolved-tool" {
			return backends, nil
		}
		return nil, errors.New("not found")
	}

	runner := NewRunner(
		WithToolResolver(toolResolver),
		WithBackendsResolver(backendsResolver),
	)

	resolved, err := runner.resolveTool(context.Background(), "resolved-tool")
	if err != nil {
		t.Fatalf("resolveTool() error = %v", err)
	}

	if resolved.tool.Name != "resolved-tool" {
		t.Errorf("tool.Name = %q, want %q", resolved.tool.Name, "resolved-tool")
	}
	if len(resolved.backends) != 1 {
		t.Errorf("len(backends) = %d, want 1", len(resolved.backends))
	}
}

func TestResolveTool_IndexNotFound_FallsBack(t *testing.T) {
	// Index returns ErrNotFound, should fall back to resolvers
	idx := newMockIndex()
	idx.GetToolErr = index.ErrNotFound

	tool := testTool("fallback-tool")
	backends := []model.ToolBackend{testMCPBackend("server1")}

	runner := NewRunner(
		WithIndex(idx),
		WithToolResolver(func(_ string) (*model.Tool, error) {
			return &tool, nil
		}),
		WithBackendsResolver(func(_ string) ([]model.ToolBackend, error) {
			return backends, nil
		}),
	)

	resolved, err := runner.resolveTool(context.Background(), "fallback-tool")
	if err != nil {
		t.Fatalf("resolveTool() error = %v", err)
	}

	if resolved.tool.Name != "fallback-tool" {
		t.Errorf("tool.Name = %q, want %q", resolved.tool.Name, "fallback-tool")
	}
}

func TestResolveTool_NotFound(t *testing.T) {
	runner := NewRunner()

	_, err := runner.resolveTool(context.Background(), "nonexistent")

	if !errors.Is(err, ErrToolNotFound) {
		t.Errorf("resolveTool() error = %v, want ErrToolNotFound", err)
	}
}

func TestResolveTool_NoBackends(t *testing.T) {
	tool := testTool("no-backends")

	runner := NewRunner(
		WithToolResolver(func(_ string) (*model.Tool, error) {
			return &tool, nil
		}),
		// No backends resolver
	)

	_, err := runner.resolveTool(context.Background(), "no-backends")

	if !errors.Is(err, ErrNoBackends) {
		t.Errorf("resolveTool() error = %v, want ErrNoBackends", err)
	}
}

func TestSelectBackend_LocalPriority(t *testing.T) {
	runner := NewRunner()

	backends := []model.ToolBackend{
		testMCPBackend("server1"),
		testProviderBackend("provider1", "tool1"),
		testLocalBackend("handler1"),
	}

	selected, err := runner.selectBackend(backends)
	if err != nil {
		t.Fatalf("selectBackend() error = %v", err)
	}

	if selected.Kind != model.BackendKindLocal {
		t.Errorf("selectBackend() = %s, want local", selected.Kind)
	}
}

func TestSelectBackend_ProviderPriority(t *testing.T) {
	runner := NewRunner()

	backends := []model.ToolBackend{
		testMCPBackend("server1"),
		testProviderBackend("provider1", "tool1"),
	}

	selected, err := runner.selectBackend(backends)
	if err != nil {
		t.Fatalf("selectBackend() error = %v", err)
	}

	if selected.Kind != model.BackendKindProvider {
		t.Errorf("selectBackend() = %s, want provider", selected.Kind)
	}
}

func TestSelectBackend_MCPFallback(t *testing.T) {
	runner := NewRunner()

	backends := []model.ToolBackend{
		testMCPBackend("server1"),
	}

	selected, err := runner.selectBackend(backends)
	if err != nil {
		t.Fatalf("selectBackend() error = %v", err)
	}

	if selected.Kind != model.BackendKindMCP {
		t.Errorf("selectBackend() = %s, want mcp", selected.Kind)
	}
}

func TestSelectBackend_NoBackends(t *testing.T) {
	runner := NewRunner()

	_, err := runner.selectBackend(nil)

	if !errors.Is(err, ErrNoBackends) {
		t.Errorf("selectBackend() error = %v, want ErrNoBackends", err)
	}
}

func TestSelectBackend_CustomSelector(t *testing.T) {
	// Custom selector that always picks MCP
	customSelector := func(backends []model.ToolBackend) model.ToolBackend {
		for _, b := range backends {
			if b.Kind == model.BackendKindMCP {
				return b
			}
		}
		return backends[0]
	}

	runner := NewRunner(WithBackendSelector(customSelector))

	backends := []model.ToolBackend{
		testLocalBackend("handler1"),
		testMCPBackend("server1"),
	}

	selected, err := runner.selectBackend(backends)
	if err != nil {
		t.Fatalf("selectBackend() error = %v", err)
	}

	if selected.Kind != model.BackendKindMCP {
		t.Errorf("selectBackend() = %s, want mcp", selected.Kind)
	}
}
