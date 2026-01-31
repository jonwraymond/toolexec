package code

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolfoundation/model"
)

type customStruct struct {
	Name   string
	Count  int
	Nested *nestedStruct
}

type nestedStruct struct {
	Flag bool
}

func TestTools_SearchTools_DelegatesToIndex(t *testing.T) {
	index := &mockIndex{
		searchResult: []index.Summary{
			{ID: "tool1", Name: "tool1", ShortDescription: "A tool"},
		},
	}
	tools := newTools(&Config{
		Index:  index,
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	result, err := tools.SearchTools(context.Background(), "query", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(index.searchCalls) != 1 {
		t.Fatalf("expected 1 search call, got %d", len(index.searchCalls))
	}
	if index.searchCalls[0].query != "query" {
		t.Errorf("expected query 'query', got %q", index.searchCalls[0].query)
	}
	if index.searchCalls[0].limit != 10 {
		t.Errorf("expected limit 10, got %d", index.searchCalls[0].limit)
	}
	if len(result) != 1 || result[0].ID != "tool1" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTools_SearchTools_Error(t *testing.T) {
	expectedErr := errors.New("search failed")
	index := &mockIndex{
		searchErr: expectedErr,
	}
	tools := newTools(&Config{
		Index:  index,
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	_, err := tools.SearchTools(context.Background(), "query", 10)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestTools_SearchTools_ContextCanceled(t *testing.T) {
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tools.SearchTools(ctx, "query", 10)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestTools_ListNamespaces_DelegatesToIndex(t *testing.T) {
	index := &mockIndex{
		namespacesResult: []string{"ns1", "ns2"},
	}
	tools := newTools(&Config{
		Index:  index,
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	result, err := tools.ListNamespaces(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if index.namespacesCalls != 1 {
		t.Fatalf("expected 1 namespaces call, got %d", index.namespacesCalls)
	}
	if len(result) != 2 || result[0] != "ns1" || result[1] != "ns2" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTools_DescribeTool_DelegatesToDocs(t *testing.T) {
	store := &mockStore{
		describeResult: tooldoc.ToolDoc{
			Summary: "A tool",
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   store,
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	result, err := tools.DescribeTool(context.Background(), "tool1", tooldoc.DetailFull)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.describeCalls) != 1 {
		t.Fatalf("expected 1 describe call, got %d", len(store.describeCalls))
	}
	if store.describeCalls[0].id != "tool1" {
		t.Errorf("expected id 'tool1', got %q", store.describeCalls[0].id)
	}
	if store.describeCalls[0].level != tooldoc.DetailFull {
		t.Errorf("expected level DetailFull, got %v", store.describeCalls[0].level)
	}
	if result.Summary != "A tool" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTools_ListToolExamples_DelegatesToDocs(t *testing.T) {
	store := &mockStore{
		examplesResult: []tooldoc.ToolExample{
			{Title: "Example 1"},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   store,
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	result, err := tools.ListToolExamples(context.Background(), "tool1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.examplesCalls) != 1 {
		t.Fatalf("expected 1 examples call, got %d", len(store.examplesCalls))
	}
	if store.examplesCalls[0].id != "tool1" {
		t.Errorf("expected id 'tool1', got %q", store.examplesCalls[0].id)
	}
	if store.examplesCalls[0].maxExamples != 5 {
		t.Errorf("expected max 5, got %d", store.examplesCalls[0].maxExamples)
	}
	if len(result) != 1 || result[0].Title != "Example 1" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTools_RunTool_DelegatesToRunner(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: map[string]any{"key": "value"},
			Backend: model.ToolBackend{
				Kind: model.BackendKindMCP,
			},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	args := map[string]any{"arg1": "value1"}
	result, err := tools.RunTool(ctx, "tool1", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}
	if runner.runCalls[0].toolID != "tool1" {
		t.Errorf("expected toolID 'tool1', got %q", runner.runCalls[0].toolID)
	}
	if runner.runCalls[0].args["arg1"] != "value1" {
		t.Errorf("expected arg1 'value1', got %v", runner.runCalls[0].args)
	}
	if result.Structured.(map[string]any)["key"] != "value" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTools_RunTool_RecordsToolCall(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: map[string]any{"result": 42},
			Backend: model.ToolBackend{
				Kind: model.BackendKindLocal,
			},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	args := map[string]any{"x": 1}
	_, err := tools.RunTool(ctx, "ns:tool", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := tools.GetToolCalls()
	if len(records) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(records))
	}

	record := records[0]
	if record.ToolID != "ns:tool" {
		t.Errorf("expected ToolID 'ns:tool', got %q", record.ToolID)
	}
	if record.Args["x"] != 1 {
		t.Errorf("expected Args['x'] = 1, got %v", record.Args)
	}
	if record.Structured.(map[string]any)["result"] != 42 {
		t.Errorf("expected Structured['result'] = 42, got %v", record.Structured)
	}
	if record.DurationMs < 0 {
		t.Errorf("expected non-negative DurationMs, got %d", record.DurationMs)
	}
}

func TestDeepCopyArgs_CustomStructPointer(t *testing.T) {
	input := map[string]any{
		"custom": &customStruct{
			Name:  "alpha",
			Count: 7,
			Nested: &nestedStruct{
				Flag: true,
			},
		},
	}

	copied := deepCopyArgs(input)
	customVal, ok := copied["custom"].(map[string]any)
	if !ok {
		t.Fatalf("expected custom to be map[string]any, got %T", copied["custom"])
	}
	if customVal["Name"] != "alpha" {
		t.Errorf("expected Name 'alpha', got %v", customVal["Name"])
	}
	if customVal["Count"] != float64(7) {
		t.Errorf("expected Count 7, got %v", customVal["Count"])
	}
	nestedVal, ok := customVal["Nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected Nested to be map[string]any, got %T", customVal["Nested"])
	}
	if nestedVal["Flag"] != true {
		t.Errorf("expected Flag true, got %v", nestedVal["Flag"])
	}
}

func TestTools_RunTool_RecordsError(t *testing.T) {
	expectedErr := errors.New("tool execution failed")
	runner := &mockRunner{
		runErr: expectedErr,
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	_, err := tools.RunTool(ctx, "tool1", nil)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	records := tools.GetToolCalls()
	if len(records) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(records))
	}

	record := records[0]
	if record.Error != "tool execution failed" {
		t.Errorf("expected Error 'tool execution failed', got %q", record.Error)
	}
	if record.ErrorOp != "run" {
		t.Errorf("expected ErrorOp 'run', got %q", record.ErrorOp)
	}
}

func TestTools_RunTool_RecordsBackendKindOnToolError(t *testing.T) {
	backend := model.ToolBackend{Kind: model.BackendKindMCP}
	toolErr := run.WrapError("tool1", &backend, "execute", errors.New("boom"))
	runner := &mockRunner{
		runErr: toolErr,
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	_, err := tools.RunTool(context.Background(), "tool1", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	records := tools.GetToolCalls()
	if len(records) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(records))
	}
	if records[0].BackendKind != "mcp" {
		t.Errorf("expected BackendKind mcp, got %q", records[0].BackendKind)
	}
}

func TestTools_RunTool_RecordsBackendKind(t *testing.T) {
	testCases := []struct {
		name     string
		kind     model.BackendKind
		expected string
	}{
		{"mcp", model.BackendKindMCP, "mcp"},
		{"provider", model.BackendKindProvider, "provider"},
		{"local", model.BackendKindLocal, "local"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &mockRunner{
				runResult: run.RunResult{
					Backend: model.ToolBackend{Kind: tc.kind},
				},
			}
			tools := newTools(&Config{
				Index:  &mockIndex{},
				Docs:   &mockStore{},
				Run:    runner,
				Engine: &mockEngine{},
			}, 0, 0)

			ctx := context.Background()
			_, _ = tools.RunTool(ctx, "tool", nil)

			records := tools.GetToolCalls()
			if records[0].BackendKind != tc.expected {
				t.Errorf("expected BackendKind %q, got %q", tc.expected, records[0].BackendKind)
			}
		})
	}
}

func TestTools_RunChain_DelegatesToRunner(t *testing.T) {
	runner := &mockRunner{
		chainResult: run.RunResult{
			Structured: "final",
		},
		chainSteps: []run.StepResult{
			{Result: run.RunResult{Structured: "step1"}},
			{Result: run.RunResult{Structured: "step2"}},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	steps := []run.ChainStep{
		{ToolID: "tool1", Args: map[string]any{"a": 1}},
		{ToolID: "tool2", Args: map[string]any{"b": 2}, UsePrevious: true},
	}
	result, stepResults, err := tools.RunChain(ctx, steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.chainCalls) != 1 {
		t.Fatalf("expected 1 chain call, got %d", len(runner.chainCalls))
	}
	if len(runner.chainCalls[0]) != 2 {
		t.Errorf("expected 2 steps, got %d", len(runner.chainCalls[0]))
	}
	if result.Structured != "final" {
		t.Errorf("unexpected final result: %v", result)
	}
	if len(stepResults) != 2 {
		t.Errorf("expected 2 step results, got %d", len(stepResults))
	}
}

func TestTools_RunChain_RecordsAllSteps(t *testing.T) {
	runner := &mockRunner{
		chainResult: run.RunResult{},
		chainSteps: []run.StepResult{
			{
				Result: run.RunResult{
					Structured: "result1",
					Backend:    model.ToolBackend{Kind: model.BackendKindMCP},
				},
			},
			{
				Result: run.RunResult{
					Structured: "result2",
					Backend:    model.ToolBackend{Kind: model.BackendKindLocal},
				},
			},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	steps := []run.ChainStep{
		{ToolID: "tool1", Args: map[string]any{"a": 1}},
		{ToolID: "tool2", Args: map[string]any{"b": 2}},
	}
	_, _, err := tools.RunChain(ctx, steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := tools.GetToolCalls()
	if len(records) != 2 {
		t.Fatalf("expected 2 tool call records, got %d", len(records))
	}

	if records[0].ToolID != "tool1" {
		t.Errorf("expected first record ToolID 'tool1', got %q", records[0].ToolID)
	}
	if records[0].Structured != "result1" {
		t.Errorf("expected first record Structured 'result1', got %v", records[0].Structured)
	}
	if records[1].ToolID != "tool2" {
		t.Errorf("expected second record ToolID 'tool2', got %q", records[1].ToolID)
	}
	if records[1].BackendKind != "local" {
		t.Errorf("expected second record BackendKind 'local', got %q", records[1].BackendKind)
	}
}

func TestTools_RunChain_ReconstructsEffectiveArgsAndCopies(t *testing.T) {
	step1Structured := map[string]any{"value": "one"}
	runner := &mockRunner{
		chainResult: run.RunResult{},
		chainSteps: []run.StepResult{
			{
				Result: run.RunResult{
					Structured: step1Structured,
					Backend:    model.ToolBackend{Kind: model.BackendKindLocal},
				},
				Backend: model.ToolBackend{Kind: model.BackendKindLocal},
			},
			{
				Result: run.RunResult{
					Structured: "two",
					Backend:    model.ToolBackend{Kind: model.BackendKindMCP},
				},
				Backend: model.ToolBackend{Kind: model.BackendKindMCP},
			},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	steps := []run.ChainStep{
		{ToolID: "tool1", Args: map[string]any{"a": 1}},
		{
			ToolID:      "tool2",
			Args:        map[string]any{"b": 2, "previous": "bad"},
			UsePrevious: true,
		},
	}
	_, _, err := tools.RunChain(ctx, steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := tools.GetToolCalls()
	if len(records) != 2 {
		t.Fatalf("expected 2 tool call records, got %d", len(records))
	}

	// Step 2 should have previous injected and overwriting the provided value.
	prev, ok := records[1].Args["previous"].(map[string]any)
	if !ok {
		t.Fatalf("previous arg type = %T, want map[string]any", records[1].Args["previous"])
	}
	if prev["value"] != "one" {
		t.Errorf("previous[value] = %v, want one", prev["value"])
	}

	// Deep copy: mutating source values should not affect recorded args.
	step1Structured["value"] = "changed"
	if prev["value"] != "one" {
		t.Errorf("previous[value] changed to %v, want one", prev["value"])
	}
	steps[0].Args["a"] = 99
	if records[0].Args["a"] != 1 {
		t.Errorf("records[0].Args[a] = %v, want 1", records[0].Args["a"])
	}
}

func TestTools_RunChain_BackendKindOnError(t *testing.T) {
	chainErr := errors.New("chain failed")
	runner := &mockRunner{
		chainSteps: []run.StepResult{
			{
				ToolID:  "tool1",
				Backend: model.ToolBackend{Kind: model.BackendKindLocal},
				Err:     chainErr,
			},
		},
		chainErr: chainErr,
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	_, _, err := tools.RunChain(context.Background(), []run.ChainStep{{ToolID: "tool1"}})
	if err == nil {
		t.Fatal("expected error")
	}

	records := tools.GetToolCalls()
	if len(records) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(records))
	}
	if records[0].BackendKind != "local" {
		t.Errorf("expected BackendKind local, got %q", records[0].BackendKind)
	}
	if records[0].Error == "" {
		t.Error("expected error to be recorded")
	}
}

func TestTools_RunChain_CountsExecutedStepsOnly(t *testing.T) {
	chainErr := errors.New("step1 failed")
	runner := &mockRunner{
		runResult: run.RunResult{
			Backend: model.ToolBackend{Kind: model.BackendKindLocal},
		},
		chainSteps: []run.StepResult{
			{
				ToolID:  "step1",
				Backend: model.ToolBackend{Kind: model.BackendKindLocal},
				Err:     chainErr,
			},
		},
		chainErr: chainErr,
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 4, 0)

	steps := []run.ChainStep{
		{ToolID: "step1"},
		{ToolID: "step2"},
		{ToolID: "step3"},
	}
	_, _, err := tools.RunChain(context.Background(), steps)
	if err == nil {
		t.Fatal("expected chain error")
	}

	// Only one step executed, so we should still have room for 3 more calls.
	for i := 0; i < 3; i++ {
		if _, err := tools.RunTool(context.Background(), "extra", nil); err != nil {
			t.Fatalf("RunTool %d unexpected error: %v", i+1, err)
		}
	}
}

func TestTools_Println_CapturesToStdout(t *testing.T) {
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	tools.Println("hello")

	stdout := tools.GetStdout()
	if stdout != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", stdout)
	}
}

func TestTools_Println_MultipleArgs(t *testing.T) {
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	tools.Println("hello", "world", 42)

	stdout := tools.GetStdout()
	if stdout != "hello world 42\n" {
		t.Errorf("expected stdout 'hello world 42\\n', got %q", stdout)
	}
}

func TestTools_Println_MultipleCalls(t *testing.T) {
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}, 0, 0)

	tools.Println("line1")
	tools.Println("line2")

	stdout := tools.GetStdout()
	if stdout != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", stdout)
	}
}

func TestTools_MaxToolCalls_Enforced(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 2, 0) // Max 2 calls

	ctx := context.Background()

	// First call should succeed
	_, err := tools.RunTool(ctx, "tool1", nil)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	// Second call should succeed
	_, err = tools.RunTool(ctx, "tool2", nil)
	if err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}

	// Third call should fail with ErrLimitExceeded
	_, err = tools.RunTool(ctx, "tool3", nil)
	if err == nil {
		t.Fatal("third call should fail")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
}

func TestTools_MaxChainSteps_Enforced(t *testing.T) {
	runner := &mockRunner{
		chainResult: run.RunResult{},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 1) // Max 1 step per chain

	steps := []run.ChainStep{
		{ToolID: "tool1"},
		{ToolID: "tool2"},
	}
	_, _, err := tools.RunChain(context.Background(), steps)
	if err == nil {
		t.Fatal("expected error due to max chain steps exceeded")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
	if len(runner.chainCalls) != 0 {
		t.Errorf("expected runner not to be called, got %d calls", len(runner.chainCalls))
	}
}

func TestTools_MaxToolCalls_ZeroIsUnlimited(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0) // Unlimited

	ctx := context.Background()

	// Should be able to make many calls
	for i := 0; i < 100; i++ {
		_, err := tools.RunTool(ctx, "tool", nil)
		if err != nil {
			t.Fatalf("call %d should succeed: %v", i, err)
		}
	}
}

func TestTools_RunTool_NilArgs(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: "result",
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 0, 0)

	ctx := context.Background()
	result, err := tools.RunTool(ctx, "tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Structured != "result" {
		t.Errorf("expected result 'result', got %v", result.Structured)
	}

	// Verify nil args was passed through
	if len(runner.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(runner.runCalls))
	}
	if runner.runCalls[0].args != nil {
		t.Errorf("expected nil args, got %v", runner.runCalls[0].args)
	}
}

func TestTools_RunChain_CountsAgainstMaxToolCalls(t *testing.T) {
	runner := &mockRunner{
		runResult: run.RunResult{},
		chainSteps: []run.StepResult{
			{Result: run.RunResult{}},
			{Result: run.RunResult{}},
		},
	}
	tools := newTools(&Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}, 3, 0) // Max 3 calls

	ctx := context.Background()

	// Chain with 2 steps should use 2 of the 3 calls
	_, _, err := tools.RunChain(ctx, []run.ChainStep{
		{ToolID: "tool1"},
		{ToolID: "tool2"},
	})
	if err != nil {
		t.Fatalf("chain should succeed: %v", err)
	}

	// Only 1 call left, so second chain with 2 steps should fail
	_, _, err = tools.RunChain(ctx, []run.ChainStep{
		{ToolID: "tool3"},
		{ToolID: "tool4"},
	})
	if err == nil {
		t.Fatal("second chain should fail")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
}
