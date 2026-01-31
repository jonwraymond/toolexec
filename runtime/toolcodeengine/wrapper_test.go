package toolcodeengine

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/code"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// TestWrapToolsImplementsGateway verifies the wrapper satisfies ToolGateway
func TestWrapToolsImplementsGateway(t *testing.T) {
	t.Helper()
	tools := &mockTools{}
	gw := WrapTools(tools)

	_ = gw
	var _ runtime.ToolGateway = (*toolsGateway)(nil)
}

func TestWrapToolsSearchTools(t *testing.T) {
	tools := &mockTools{
		searchResults: []index.Summary{
			{ID: "test:tool", Name: "tool"},
		},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	results, err := gw.SearchTools(ctx, "test", 10)
	if err != nil {
		t.Fatalf("SearchTools() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchTools() returned %d results, want 1", len(results))
	}
	if results[0].ID != "test:tool" {
		t.Errorf("SearchTools()[0].ID = %q, want %q", results[0].ID, "test:tool")
	}
}

func TestWrapToolsListNamespaces(t *testing.T) {
	tools := &mockTools{
		namespaces: []string{"ns1", "ns2"},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	results, err := gw.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("ListNamespaces() returned %d results, want 2", len(results))
	}
}

func TestWrapToolsDescribeTool(t *testing.T) {
	tools := &mockTools{
		toolDoc: tooldoc.ToolDoc{Summary: "Test tool"},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	doc, err := gw.DescribeTool(ctx, "test:tool", tooldoc.DetailSummary)
	if err != nil {
		t.Fatalf("DescribeTool() error = %v", err)
	}

	if doc.Summary != "Test tool" {
		t.Errorf("DescribeTool().Summary = %q, want %q", doc.Summary, "Test tool")
	}
}

func TestWrapToolsListToolExamples(t *testing.T) {
	tools := &mockTools{
		examples: []tooldoc.ToolExample{
			{Title: "Example 1"},
		},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	results, err := gw.ListToolExamples(ctx, "test:tool", 10)
	if err != nil {
		t.Fatalf("ListToolExamples() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("ListToolExamples() returned %d results, want 1", len(results))
	}
}

func TestWrapToolsRunTool(t *testing.T) {
	tools := &mockTools{
		runResult: run.RunResult{
			Structured: "result",
		},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	result, err := gw.RunTool(ctx, "test:tool", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("RunTool() error = %v", err)
	}

	if result.Structured != "result" {
		t.Errorf("RunTool().Structured = %v, want %v", result.Structured, "result")
	}
}

func TestWrapToolsRunChain(t *testing.T) {
	tools := &mockTools{
		chainResult: run.RunResult{
			Structured: "chain_result",
		},
		stepResults: []run.StepResult{
			{ToolID: "step1"},
		},
	}
	gw := WrapTools(tools)

	ctx := context.Background()
	steps := []run.ChainStep{
		{ToolID: "step1"},
	}
	result, stepResults, err := gw.RunChain(ctx, steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	if result.Structured != "chain_result" {
		t.Errorf("RunChain().Structured = %v, want %v", result.Structured, "chain_result")
	}
	if len(stepResults) != 1 {
		t.Errorf("RunChain() returned %d step results, want 1", len(stepResults))
	}
}

func TestWrapToolsContextPropagation(t *testing.T) {
	t.Helper()
	tools := &ctxTools{}
	gw := WrapTools(tools)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := gw.SearchTools(ctx, "test", 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

type ctxTools struct{}

func (c *ctxTools) SearchTools(ctx context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, ctx.Err()
}

func (c *ctxTools) ListNamespaces(ctx context.Context) ([]string, error) {
	return nil, ctx.Err()
}

func (c *ctxTools) DescribeTool(ctx context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, ctx.Err()
}

func (c *ctxTools) ListToolExamples(ctx context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return nil, ctx.Err()
}

func (c *ctxTools) RunTool(ctx context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return run.RunResult{}, ctx.Err()
}

func (c *ctxTools) RunChain(ctx context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, ctx.Err()
}

func (c *ctxTools) Println(_ ...any) {}

// errTools returns errors for testing error handling
type errTools struct {
	err error
}

func (e *errTools) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, e.err
}

func (e *errTools) ListNamespaces(_ context.Context) ([]string, error) {
	return nil, e.err
}

func (e *errTools) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, e.err
}

func (e *errTools) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return nil, e.err
}

func (e *errTools) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return run.RunResult{}, e.err
}

func (e *errTools) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, e.err
}

func (e *errTools) Println(_ ...any) {}

var _ code.Tools = (*errTools)(nil)

func TestWrapToolsErrorPropagation(t *testing.T) {
	expectedErr := errors.New("test error")
	tools := &errTools{err: expectedErr}
	gw := WrapTools(tools)

	ctx := context.Background()

	t.Run("SearchTools propagates error", func(t *testing.T) {
		_, err := gw.SearchTools(ctx, "test", 10)
		if err != expectedErr {
			t.Errorf("SearchTools() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("ListNamespaces propagates error", func(t *testing.T) {
		_, err := gw.ListNamespaces(ctx)
		if err != expectedErr {
			t.Errorf("ListNamespaces() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("DescribeTool propagates error", func(t *testing.T) {
		_, err := gw.DescribeTool(ctx, "test", tooldoc.DetailSummary)
		if err != expectedErr {
			t.Errorf("DescribeTool() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("RunTool propagates error", func(t *testing.T) {
		_, err := gw.RunTool(ctx, "test", nil)
		if err != expectedErr {
			t.Errorf("RunTool() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("RunChain propagates error", func(t *testing.T) {
		_, _, err := gw.RunChain(ctx, []run.ChainStep{{ToolID: "test"}})
		if err != expectedErr {
			t.Errorf("RunChain() error = %v, want %v", err, expectedErr)
		}
	})
}
