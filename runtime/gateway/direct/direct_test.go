package direct

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
	"github.com/jonwraymond/toolfoundation/model"
)

// mockIndex implements index.Index for testing
type mockIndex struct {
	summaries  []index.Summary
	namespaces []string
	searchErr  error
}

func (m *mockIndex) RegisterTool(_ model.Tool, _ model.ToolBackend) error {
	return nil
}

func (m *mockIndex) RegisterTools(_ []index.ToolRegistration) error {
	return nil
}

func (m *mockIndex) RegisterToolsFromMCP(_ string, _ []model.Tool) error {
	return nil
}

func (m *mockIndex) UnregisterBackend(_ string, _ model.BackendKind, _ string) error {
	return nil
}

func (m *mockIndex) GetTool(_ string) (model.Tool, model.ToolBackend, error) {
	return model.Tool{}, model.ToolBackend{}, nil
}

func (m *mockIndex) GetAllBackends(_ string) ([]model.ToolBackend, error) {
	return nil, nil
}

func (m *mockIndex) Search(_ string, limit int) ([]index.Summary, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if limit > len(m.summaries) {
		return m.summaries, nil
	}
	return m.summaries[:limit], nil
}

func (m *mockIndex) SearchPage(query string, limit int, _ string) ([]index.Summary, string, error) {
	results, err := m.Search(query, limit)
	return results, "", err
}

func (m *mockIndex) ListNamespaces() ([]string, error) {
	return m.namespaces, nil
}

func (m *mockIndex) ListNamespacesPage(_ int, _ string) ([]string, string, error) {
	results, err := m.ListNamespaces()
	return results, "", err
}

// mockDocs implements tooldoc.Store for testing
type mockDocs struct {
	docs        map[string]tooldoc.ToolDoc
	examples    map[string][]tooldoc.ToolExample
	descErr     error
	examplesErr error
}

func (m *mockDocs) DescribeTool(id string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	if m.descErr != nil {
		return tooldoc.ToolDoc{}, m.descErr
	}
	doc, ok := m.docs[id]
	if !ok {
		return tooldoc.ToolDoc{}, errors.New("tool not found")
	}
	return doc, nil
}

func (m *mockDocs) ListExamples(id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	if m.examplesErr != nil {
		return nil, m.examplesErr
	}
	examples := m.examples[id]
	if maxExamples > 0 && maxExamples < len(examples) {
		return examples[:maxExamples], nil
	}
	return examples, nil
}

// mockRunner implements run.Runner for testing
type mockRunner struct {
	runResult   run.RunResult
	runErr      error
	chainResult run.RunResult
	stepResults []run.StepResult
	chainErr    error
	callCount   int
	mu          sync.Mutex
}

func (m *mockRunner) Run(ctx context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if ctx.Err() != nil {
		return run.RunResult{}, ctx.Err()
	}
	if m.runErr != nil {
		return run.RunResult{}, m.runErr
	}
	return m.runResult, nil
}

func (m *mockRunner) RunStream(_ context.Context, _ string, _ map[string]any) (<-chan run.StreamEvent, error) {
	return nil, errors.New("streaming not supported")
}

func (m *mockRunner) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	m.mu.Lock()
	m.callCount += len(steps)
	m.mu.Unlock()

	if ctx.Err() != nil {
		return run.RunResult{}, nil, ctx.Err()
	}
	if m.chainErr != nil {
		return run.RunResult{}, m.stepResults, m.chainErr
	}
	return m.chainResult, m.stepResults, nil
}

// TestGatewayImplementsInterface verifies Gateway satisfies ToolGateway
func TestGatewayImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.ToolGateway = (*Gateway)(nil)
}

func TestGatewaySearchTools(t *testing.T) {
	summaries := []index.Summary{
		{ID: "test:tool1", Name: "tool1"},
		{ID: "test:tool2", Name: "tool2"},
		{ID: "test:tool3", Name: "tool3"},
	}

	index := &mockIndex{summaries: summaries}
	gw := New(Config{
		Index:  index,
		Docs:   &mockDocs{},
		Runner: &mockRunner{},
	})

	ctx := context.Background()

	t.Run("delegates to index", func(t *testing.T) {
		results, err := gw.SearchTools(ctx, "test", 10)
		if err != nil {
			t.Fatalf("SearchTools() error = %v", err)
		}
		if len(results) != 3 {
			t.Errorf("SearchTools() returned %d results, want 3", len(results))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		results, err := gw.SearchTools(ctx, "test", 2)
		if err != nil {
			t.Fatalf("SearchTools() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("SearchTools() returned %d results, want 2", len(results))
		}
	})

	t.Run("propagates errors", func(t *testing.T) {
		index := &mockIndex{searchErr: errors.New("search failed")}
		gw := New(Config{
			Index:  index,
			Docs:   &mockDocs{},
			Runner: &mockRunner{},
		})

		_, err := gw.SearchTools(ctx, "test", 10)
		if err == nil {
			t.Error("SearchTools() should propagate error")
		}
	})
}

func TestGatewayListNamespaces(t *testing.T) {
	namespaces := []string{"ns1", "ns2"}
	index := &mockIndex{namespaces: namespaces}
	gw := New(Config{
		Index:  index,
		Docs:   &mockDocs{},
		Runner: &mockRunner{},
	})

	ctx := context.Background()

	results, err := gw.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("ListNamespaces() returned %d results, want 2", len(results))
	}
}

func TestGatewayDescribeTool(t *testing.T) {
	docs := &mockDocs{
		docs: map[string]tooldoc.ToolDoc{
			"test:tool": {Summary: "Test tool"},
		},
	}
	gw := New(Config{
		Index:  &mockIndex{},
		Docs:   docs,
		Runner: &mockRunner{},
	})

	ctx := context.Background()

	t.Run("delegates to docs store", func(t *testing.T) {
		doc, err := gw.DescribeTool(ctx, "test:tool", tooldoc.DetailSummary)
		if err != nil {
			t.Fatalf("DescribeTool() error = %v", err)
		}
		if doc.Summary != "Test tool" {
			t.Errorf("DescribeTool().Summary = %q, want %q", doc.Summary, "Test tool")
		}
	})

	t.Run("returns error for non-existent tool", func(t *testing.T) {
		_, err := gw.DescribeTool(ctx, "nonexistent:tool", tooldoc.DetailSummary)
		if err == nil {
			t.Error("DescribeTool() should return error for non-existent tool")
		}
	})
}

func TestGatewayListToolExamples(t *testing.T) {
	examples := []tooldoc.ToolExample{
		{Title: "Example 1"},
		{Title: "Example 2"},
	}
	docs := &mockDocs{
		examples: map[string][]tooldoc.ToolExample{
			"test:tool": examples,
		},
	}
	gw := New(Config{
		Index:  &mockIndex{},
		Docs:   docs,
		Runner: &mockRunner{},
	})

	ctx := context.Background()

	results, err := gw.ListToolExamples(ctx, "test:tool", 10)
	if err != nil {
		t.Fatalf("ListToolExamples() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("ListToolExamples() returned %d results, want 2", len(results))
	}
}

func TestGatewayRunTool(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: "result",
		},
	}
	gw := New(Config{
		Index:  &mockIndex{},
		Docs:   &mockDocs{},
		Runner: runner,
	})

	ctx := context.Background()

	t.Run("delegates to runner", func(t *testing.T) {
		result, err := gw.RunTool(ctx, "test:tool", map[string]any{"key": "value"})
		if err != nil {
			t.Fatalf("RunTool() error = %v", err)
		}
		if result.Structured != "result" {
			t.Errorf("RunTool().Structured = %v, want %v", result.Structured, "result")
		}
	})

	t.Run("propagates context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := gw.RunTool(ctx, "test:tool", nil)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("RunTool() should return context.Canceled, got %v", err)
		}
	})

	t.Run("records tool calls", func(t *testing.T) {
		runner := &mockRunner{}
		gw := New(Config{
			Index:  &mockIndex{},
			Docs:   &mockDocs{},
			Runner: runner,
		})

		_, _ = gw.RunTool(context.Background(), "tool1", nil)
		_, _ = gw.RunTool(context.Background(), "tool2", nil)

		records := gw.GetToolCalls()
		if len(records) != 2 {
			t.Errorf("GetToolCalls() returned %d records, want 2", len(records))
		}
	})
}

func TestGatewayRunChain(t *testing.T) {
	runner := &mockRunner{
		chainResult: run.RunResult{
			Structured: "chain_result",
		},
		stepResults: []run.StepResult{
			{ToolID: "step1"},
			{ToolID: "step2"},
		},
	}
	gw := New(Config{
		Index:  &mockIndex{},
		Docs:   &mockDocs{},
		Runner: runner,
	})

	ctx := context.Background()

	t.Run("delegates to runner", func(t *testing.T) {
		steps := []run.ChainStep{
			{ToolID: "step1"},
			{ToolID: "step2"},
		}
		result, stepResults, err := gw.RunChain(ctx, steps)
		if err != nil {
			t.Fatalf("RunChain() error = %v", err)
		}
		if result.Structured != "chain_result" {
			t.Errorf("RunChain().Structured = %v, want %v", result.Structured, "chain_result")
		}
		if len(stepResults) != 2 {
			t.Errorf("RunChain() returned %d step results, want 2", len(stepResults))
		}
	})

	t.Run("records executed steps only", func(t *testing.T) {
		steps := []run.ChainStep{
			{ToolID: "step1"},
			{ToolID: "step2"},
		}
		partialRunner := &mockRunner{
			stepResults: []run.StepResult{
				{ToolID: "step1", Err: errors.New("failed")},
			},
			chainErr: errors.New("failed"),
		}
		partialGw := New(Config{
			Index:  &mockIndex{},
			Docs:   &mockDocs{},
			Runner: partialRunner,
		})

		_, _, _ = partialGw.RunChain(ctx, steps)

		records := partialGw.GetToolCalls()
		if len(records) != 1 {
			t.Errorf("GetToolCalls() returned %d records, want 1", len(records))
		}
		if len(records) > 0 && records[0].ToolID != "step1" {
			t.Errorf("GetToolCalls()[0].ToolID = %q, want %q", records[0].ToolID, "step1")
		}
	})

	t.Run("handles empty steps", func(t *testing.T) {
		_, stepResults, err := gw.RunChain(ctx, []run.ChainStep{})
		if err != nil {
			t.Fatalf("RunChain() error = %v", err)
		}
		if len(stepResults) != 0 {
			t.Errorf("RunChain() with empty steps returned %d results", len(stepResults))
		}
	})
}

func TestGatewayToolCallLimits(t *testing.T) {
	runner := &mockRunner{}
	gw := New(Config{
		Index:        &mockIndex{},
		Docs:         &mockDocs{},
		Runner:       runner,
		MaxToolCalls: 2,
	})

	ctx := context.Background()

	// First two calls should succeed
	_, err := gw.RunTool(ctx, "tool1", nil)
	if err != nil {
		t.Fatalf("RunTool() first call error = %v", err)
	}
	_, err = gw.RunTool(ctx, "tool2", nil)
	if err != nil {
		t.Fatalf("RunTool() second call error = %v", err)
	}

	// Third call should fail
	_, err = gw.RunTool(ctx, "tool3", nil)
	if err == nil {
		t.Error("RunTool() should fail after exceeding MaxToolCalls")
	}
	if !errors.Is(err, ErrToolCallLimitExceeded) {
		t.Errorf("RunTool() error = %v, want ErrToolCallLimitExceeded", err)
	}
}

func TestGatewayChainStepLimits(t *testing.T) {
	runner := &mockRunner{
		stepResults: []run.StepResult{{}, {}},
	}
	gw := New(Config{
		Index:         &mockIndex{},
		Docs:          &mockDocs{},
		Runner:        runner,
		MaxChainSteps: 2,
	})

	ctx := context.Background()

	// Chain with 3 steps should fail
	steps := []run.ChainStep{
		{ToolID: "step1"},
		{ToolID: "step2"},
		{ToolID: "step3"},
	}
	_, _, err := gw.RunChain(ctx, steps)
	if err == nil {
		t.Error("RunChain() should fail when exceeding MaxChainSteps")
	}
	if !errors.Is(err, ErrChainStepLimitExceeded) {
		t.Errorf("RunChain() error = %v, want ErrChainStepLimitExceeded", err)
	}
}

func TestGatewayThreadSafety(t *testing.T) {
	t.Helper()
	runner := &mockRunner{}
	gw := New(Config{
		Index:  &mockIndex{summaries: []index.Summary{{ID: "test:tool"}}},
		Docs:   &mockDocs{},
		Runner: runner,
	})

	ctx := context.Background()
	var wg sync.WaitGroup

	// Run concurrent operations
	for i := 0; i < 10; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, _ = gw.SearchTools(ctx, "test", 10)
		}()
		go func() {
			defer wg.Done()
			_, _ = gw.RunTool(ctx, "test:tool", nil)
		}()
		go func() {
			defer wg.Done()
			_ = gw.GetToolCalls()
		}()
	}

	wg.Wait()
}

func TestGatewayContractCompliance(t *testing.T) {
	runtime.RunGatewayContractTests(t, runtime.GatewayContract{
		NewGateway: func() runtime.ToolGateway {
			return New(Config{
				Index:  &mockIndex{summaries: []index.Summary{{ID: "test:tool"}}},
				Docs:   &mockDocs{docs: map[string]tooldoc.ToolDoc{}},
				Runner: &mockRunner{},
			})
		},
	})
}
