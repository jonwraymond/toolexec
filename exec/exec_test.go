package exec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolfoundation/model"
)

// testSetup creates a configured index, docs store, and sample tool for testing.
func testSetup(t *testing.T) (index.Index, tooldoc.Store, model.Tool) {
	t.Helper()

	idx := index.NewInMemoryIndex(index.IndexOptions{
		Searcher: search.NewBM25Searcher(search.BM25Config{}),
	})
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "greet",
			Description: "Greets a user by name",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []any{"name"},
			},
		},
		Namespace: "test",
	}

	return idx, docs, tool
}

func TestNew_ValidOptions(t *testing.T) {
	idx, docs, _ := testSetup(t)

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
	})

	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if exec == nil {
		t.Fatal("New() returned nil Exec")
	}
	if exec.Index() != idx {
		t.Error("Index() did not return configured index")
	}
	if exec.DocStore() != docs {
		t.Error("DocStore() did not return configured docs")
	}
}

func TestNew_MissingIndex(t *testing.T) {
	_, docs, _ := testSetup(t)

	_, err := New(Options{
		Docs: docs,
	})

	if !errors.Is(err, ErrIndexRequired) {
		t.Errorf("New() error = %v, want %v", err, ErrIndexRequired)
	}
}

func TestNew_MissingDocs(t *testing.T) {
	idx, _, _ := testSetup(t)

	_, err := New(Options{
		Index: idx,
	})

	if !errors.Is(err, ErrDocsRequired) {
		t.Errorf("New() error = %v, want %v", err, ErrDocsRequired)
	}
}

func TestNew_DefaultsApplied(t *testing.T) {
	idx, docs, _ := testSetup(t)

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Check defaults were applied
	if exec.opts.MaxToolCalls != DefaultMaxToolCalls {
		t.Errorf("MaxToolCalls = %d, want %d", exec.opts.MaxToolCalls, DefaultMaxToolCalls)
	}
	if exec.opts.DefaultLanguage != DefaultLanguage {
		t.Errorf("DefaultLanguage = %q, want %q", exec.opts.DefaultLanguage, DefaultLanguage)
	}
	if exec.opts.DefaultTimeout != DefaultTimeout {
		t.Errorf("DefaultTimeout = %v, want %v", exec.opts.DefaultTimeout, DefaultTimeout)
	}
}

func TestExec_RunTool_LocalHandler(t *testing.T) {
	idx, docs, tool := testSetup(t)

	// Register tool with local backend
	err := idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]Handler{
			"greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
				name, _ := args["name"].(string)
				return "Hello, " + name + "!", nil
			},
		},
		ValidateInput:  false, // Disable for simpler test
		ValidateOutput: false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	result, err := exec.RunTool(ctx, "test:greet", map[string]any{"name": "World"})

	if err != nil {
		t.Fatalf("RunTool() error = %v", err)
	}
	if !result.OK() {
		t.Errorf("Result.OK() = false, want true")
	}
	if result.Value != "Hello, World!" {
		t.Errorf("Result.Value = %v, want %q", result.Value, "Hello, World!")
	}
	if result.ToolID != "test:greet" {
		t.Errorf("Result.ToolID = %q, want %q", result.ToolID, "test:greet")
	}
	if result.Duration == 0 {
		t.Error("Result.Duration = 0, want > 0")
	}
}

func TestExec_RunTool_NotFound(t *testing.T) {
	idx, docs, _ := testSetup(t)

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	result, err := exec.RunTool(ctx, "nonexistent:tool", nil)

	if err == nil {
		t.Fatal("RunTool() error = nil, want error")
	}
	if result.OK() {
		t.Error("Result.OK() = true, want false")
	}
	if result.Error == nil {
		t.Error("Result.Error = nil, want error")
	}
}

func TestExec_RunTool_HandlerError(t *testing.T) {
	idx, docs, tool := testSetup(t)

	err := idx.RegisterTool(tool, model.NewLocalBackend("error-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	handlerErr := errors.New("handler failed")
	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]Handler{
			"error-handler": func(ctx context.Context, args map[string]any) (any, error) {
				return nil, handlerErr
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	result, err := exec.RunTool(ctx, "test:greet", nil)

	if err == nil {
		t.Fatal("RunTool() error = nil, want error")
	}
	if result.OK() {
		t.Error("Result.OK() = true, want false")
	}
}

func TestExec_RunTool_ContextCanceled(t *testing.T) {
	idx, docs, tool := testSetup(t)

	err := idx.RegisterTool(tool, model.NewLocalBackend("slow-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]Handler{
			"slow-handler": func(ctx context.Context, args map[string]any) (any, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(10 * time.Second):
					return "done", nil
				}
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = exec.RunTool(ctx, "test:greet", nil)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunTool() error = %v, want %v", err, context.Canceled)
	}
}

func TestExec_RunChain(t *testing.T) {
	idx, docs, tool := testSetup(t)

	// Register tool
	err := idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]Handler{
			"greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
				name, _ := args["name"].(string)
				return "Hello, " + name + "!", nil
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	result, steps, err := exec.RunChain(ctx, []Step{
		{ToolID: "test:greet", Args: map[string]any{"name": "Alice"}},
		{ToolID: "test:greet", Args: map[string]any{"name": "Bob"}},
	})

	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(steps))
	}
	if !steps[0].OK() {
		t.Errorf("steps[0].OK() = false, want true")
	}
	if !steps[1].OK() {
		t.Errorf("steps[1].OK() = false, want true")
	}
	if result.Value != "Hello, Bob!" {
		t.Errorf("Result.Value = %v, want %q", result.Value, "Hello, Bob!")
	}
}

func TestExec_SearchTools(t *testing.T) {
	idx, docs, tool := testSetup(t)

	err := idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	results, err := exec.SearchTools(ctx, "greet", 10)

	if err != nil {
		t.Fatalf("SearchTools() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SearchTools() returned no results")
	}
	if results[0].ID != "test:greet" {
		t.Errorf("results[0].ID = %q, want %q", results[0].ID, "test:greet")
	}
}

func TestExec_GetToolDoc(t *testing.T) {
	idx, docs, tool := testSetup(t)

	err := idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	// Register documentation using the concrete type
	concreteStore := docs.(*tooldoc.InMemoryStore)
	_ = concreteStore.RegisterDoc("test:greet", tooldoc.DocEntry{
		Summary: "A friendly greeting tool",
		Notes:   "Returns a greeting message",
	})

	exec, err := New(Options{
		Index: idx,
		Docs:  docs,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	doc, err := exec.GetToolDoc(ctx, "test:greet", tooldoc.DetailFull)

	if err != nil {
		t.Fatalf("GetToolDoc() error = %v", err)
	}
	if doc.Tool == nil || doc.Tool.Name != "greet" {
		t.Errorf("doc.Tool.Name = %v, want %q", doc.Tool, "greet")
	}
	if doc.Summary != "A friendly greeting tool" {
		t.Errorf("doc.Summary = %q, want %q", doc.Summary, "A friendly greeting tool")
	}
}

func TestResult_OK(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{"no error", Result{Value: "ok"}, true},
		{"with error", Result{Error: errors.New("fail")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.OK(); got != tt.want {
				t.Errorf("Result.OK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStepResult_OK(t *testing.T) {
	tests := []struct {
		name   string
		result StepResult
		want   bool
	}{
		{"success", StepResult{Value: "ok"}, true},
		{"with error", StepResult{Error: errors.New("fail")}, false},
		{"skipped", StepResult{Skipped: true}, false},
		{"skipped with value", StepResult{Value: "ok", Skipped: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.OK(); got != tt.want {
				t.Errorf("StepResult.OK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeResult_OK(t *testing.T) {
	tests := []struct {
		name   string
		result CodeResult
		want   bool
	}{
		{"success", CodeResult{Value: "ok"}, true},
		{"with error", CodeResult{Error: errors.New("fail")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.OK(); got != tt.want {
				t.Errorf("CodeResult.OK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapLocalRegistry_Get(t *testing.T) {
	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return "result", nil
	}

	reg := newMapLocalRegistry(map[string]Handler{
		"test-handler": handler,
	})

	t.Run("existing handler", func(t *testing.T) {
		h, ok := reg.Get("test-handler")
		if !ok {
			t.Fatal("Get() ok = false, want true")
		}
		if h == nil {
			t.Fatal("Get() handler = nil, want non-nil")
		}
	})

	t.Run("missing handler", func(t *testing.T) {
		h, ok := reg.Get("nonexistent")
		if ok {
			t.Error("Get() ok = true, want false")
		}
		if h != nil {
			t.Error("Get() handler != nil, want nil")
		}
	})
}

func TestStep_ShouldStopOnError(t *testing.T) {
	tests := []struct {
		name string
		step Step
		want bool
	}{
		{"nil StopOnError", Step{}, true},
		{"StopOnError true", Step{StopOnError: boolPtr(true)}, true},
		{"StopOnError false", Step{StopOnError: boolPtr(false)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.shouldStopOnError(); got != tt.want {
				t.Errorf("shouldStopOnError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
