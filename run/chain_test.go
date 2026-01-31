package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
)

func TestRunChain_SingleStep(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("step1")
	backend := testLocalBackend("handler1")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["step1"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("handler1", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"step": 1}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "step1", Args: map[string]any{"input": "data"}},
	}

	final, results, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1", len(results))
	}

	if results[0].ToolID != "step1" {
		t.Errorf("results[0].ToolID = %q, want %q", results[0].ToolID, "step1")
	}

	m, ok := final.Structured.(map[string]any)
	if !ok {
		t.Fatalf("final.Structured type = %T, want map[string]any", final.Structured)
	}
	if m["step"] != float64(1) && m["step"] != 1 {
		t.Errorf("final.Structured[step] = %v, want 1", m["step"])
	}
}

func TestRunChain_MultiStep(t *testing.T) {
	idx := newMockIndex()

	// Register two tools
	for _, name := range []string{"step1", "step2", "step3"} {
		tool := testTool(name)
		backend := testLocalBackend("handler-" + name)
		mustRegisterTool(t, idx, tool, backend)
		idx.DefaultBackends[name] = backend
	}

	localReg := newMockLocalRegistry()
	callOrder := []string{}

	for _, name := range []string{"step1", "step2", "step3"} {
		n := name // capture
		localReg.Register("handler-"+n, func(_ context.Context, _ map[string]any) (any, error) {
			callOrder = append(callOrder, n)
			return map[string]any{"step": n}, nil
		})
	}

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "step1"},
		{ToolID: "step2"},
		{ToolID: "step3"},
	}

	final, results, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}

	// Verify execution order
	if len(callOrder) != 3 || callOrder[0] != "step1" || callOrder[1] != "step2" || callOrder[2] != "step3" {
		t.Errorf("call order = %v, want [step1 step2 step3]", callOrder)
	}

	// Final result should be from last step
	m, ok := final.Structured.(map[string]any)
	if !ok {
		t.Fatalf("final.Structured type = %T", final.Structured)
	}
	if m["step"] != "step3" {
		t.Errorf("final.Structured[step] = %v, want step3", m["step"])
	}
}

func TestRunChain_UsePrevious(t *testing.T) {
	idx := newMockIndex()

	tool1 := testTool("producer")
	backend1 := testLocalBackend("producer-handler")
	mustRegisterTool(t, idx, tool1, backend1)
	idx.DefaultBackends["producer"] = backend1

	tool2 := testTool("consumer")
	backend2 := testLocalBackend("consumer-handler")
	mustRegisterTool(t, idx, tool2, backend2)
	idx.DefaultBackends["consumer"] = backend2

	localReg := newMockLocalRegistry()
	localReg.Register("producer-handler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"produced": "data"}, nil
	})

	var receivedArgs map[string]any
	localReg.Register("consumer-handler", func(_ context.Context, args map[string]any) (any, error) {
		receivedArgs = args
		return map[string]any{"consumed": true}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "producer"},
		{ToolID: "consumer", UsePrevious: true},
	}

	_, _, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// Check that previous result was injected
	prev, ok := receivedArgs["previous"]
	if !ok {
		t.Fatal("consumer should have received 'previous' arg")
	}

	prevMap, ok := prev.(map[string]any)
	if !ok {
		t.Fatalf("previous type = %T, want map[string]any", prev)
	}
	if prevMap["produced"] != "data" {
		t.Errorf("previous[produced] = %v, want 'data'", prevMap["produced"])
	}
}

func TestRunChain_UsePrevious_Overwrites(t *testing.T) {
	idx := newMockIndex()

	for _, name := range []string{"step1", "step2"} {
		tool := testTool(name)
		backend := testLocalBackend("handler-" + name)
		mustRegisterTool(t, idx, tool, backend)
		idx.DefaultBackends[name] = backend
	}

	localReg := newMockLocalRegistry()
	localReg.Register("handler-step1", func(_ context.Context, _ map[string]any) (any, error) {
		return "first-result", nil
	})

	var receivedArgs map[string]any
	localReg.Register("handler-step2", func(_ context.Context, args map[string]any) (any, error) {
		receivedArgs = args
		return "second-result", nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "step1"},
		{
			ToolID:      "step2",
			Args:        map[string]any{"previous": "should-be-overwritten"},
			UsePrevious: true,
		},
	}

	_, _, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// UsePrevious should overwrite existing "previous" arg
	if receivedArgs["previous"] != "first-result" {
		t.Errorf("previous = %v, want 'first-result' (should overwrite)", receivedArgs["previous"])
	}
}

func TestRunChain_StopsOnError(t *testing.T) {
	idx := newMockIndex()

	for _, name := range []string{"step1", "step2", "step3"} {
		tool := testTool(name)
		backend := testLocalBackend("handler-" + name)
		mustRegisterTool(t, idx, tool, backend)
		idx.DefaultBackends[name] = backend
	}

	localReg := newMockLocalRegistry()
	callCount := 0

	localReg.Register("handler-step1", func(_ context.Context, _ map[string]any) (any, error) {
		callCount++
		return "ok", nil
	})
	localReg.Register("handler-step2", func(_ context.Context, _ map[string]any) (any, error) {
		callCount++
		return nil, errors.New("step2 failed")
	})
	localReg.Register("handler-step3", func(_ context.Context, _ map[string]any) (any, error) {
		callCount++
		return "should not reach", nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "step1"},
		{ToolID: "step2"},
		{ToolID: "step3"},
	}

	_, results, err := runner.RunChain(context.Background(), steps)

	// Should return error
	if err == nil {
		t.Error("RunChain() should return error when step fails")
	}

	// Should have 2 results (step1 success, step2 failure)
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	// Step 3 should not have been called
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (step3 should not be called)", callCount)
	}

	// Check step2 has error in results
	if results[1].Err == nil {
		t.Error("results[1].Err should not be nil")
	}
}

func TestRunChain_ContextCancelled_BetweenSteps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idx := newMockIndex()
	tool1 := testTool("step1")
	tool2 := testTool("step2")
	backend := testLocalBackend("handler")
	mustRegisterTool(t, idx, tool1, backend)
	mustRegisterTool(t, idx, tool2, backend)
	idx.DefaultBackends["step1"] = backend
	idx.DefaultBackends["step2"] = backend

	step2Called := make(chan struct{}, 1)

	localReg := newMockLocalRegistry()
	localReg.Register("handler", func(_ context.Context, args map[string]any) (any, error) {
		if args["step"] == "one" {
			cancel()
			return map[string]any{"step": "one"}, nil
		}
		step2Called <- struct{}{}
		return map[string]any{"step": "two"}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	errCh := make(chan error, 1)
	go func() {
		_, _, err := runner.RunChain(ctx, []ChainStep{
			{ToolID: "step1", Args: map[string]any{"step": "one"}},
			{ToolID: "step2", Args: map[string]any{"step": "two"}},
		})
		errCh <- err
	}()

	err := <-errCh
	if err == nil {
		t.Fatal("RunChain() should return error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunChain() error = %v, want context.Canceled", err)
	}
	select {
	case <-step2Called:
		t.Error("RunChain() should not execute steps after cancellation")
	default:
	}
}

func TestRunChain_Empty(t *testing.T) {
	runner := NewRunner()

	final, results, err := runner.RunChain(context.Background(), nil)

	if err != nil {
		t.Errorf("RunChain(nil) error = %v, want nil", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
	if final.Tool.Name != "" {
		t.Errorf("final.Tool.Name = %q, want empty", final.Tool.Name)
	}
}

func TestRunChain_FinalResult(t *testing.T) {
	idx := newMockIndex()

	for _, name := range []string{"step1", "step2"} {
		tool := testTool(name)
		backend := testLocalBackend("handler-" + name)
		mustRegisterTool(t, idx, tool, backend)
		idx.DefaultBackends[name] = backend
	}

	localReg := newMockLocalRegistry()
	localReg.Register("handler-step1", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"step": 1}, nil
	})
	localReg.Register("handler-step2", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"step": 2, "final": true}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "step1"},
		{ToolID: "step2"},
	}

	final, results, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// Final result should be the last step's result
	if final.Tool.Name != "step2" {
		t.Errorf("final.Tool.Name = %q, want %q", final.Tool.Name, "step2")
	}

	// Structured should be the last step's structured result
	m, ok := final.Structured.(map[string]any)
	if !ok {
		t.Fatalf("final.Structured type = %T", final.Structured)
	}
	if m["final"] != true {
		t.Error("final.Structured should be from last step")
	}

	// Final result should match the last item in results
	if results[1].Result.Tool.Name != final.Tool.Name {
		t.Error("final should match results[last].Result")
	}
}

func TestRunChain_UsePrevious_FirstStep_InjectsNil(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("mytool")
	backend := testLocalBackend("handler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	var receivedArgs map[string]any
	var previousKeyExists bool
	localReg.Register("handler", func(_ context.Context, args map[string]any) (any, error) {
		receivedArgs = args
		_, previousKeyExists = args["previous"]
		return "ok", nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	// First step with UsePrevious: true - should inject nil
	steps := []ChainStep{
		{ToolID: "mytool", UsePrevious: true},
	}

	_, _, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// Key must exist (per PRD: inject args["previous"] even when previous is nil)
	if !previousKeyExists {
		t.Error("'previous' key should exist in args when UsePrevious is true, even on first step")
	}

	// Value should be nil
	if receivedArgs["previous"] != nil {
		t.Errorf("previous = %v, want nil (first step has no previous result)", receivedArgs["previous"])
	}
}

func TestRunChain_BackendInStepResult(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return "ok", nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "mytool"},
	}

	_, results, err := runner.RunChain(context.Background(), steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	// StepResult should include backend info
	if results[0].Backend.Kind != model.BackendKindLocal {
		t.Errorf("results[0].Backend.Kind = %q, want local", results[0].Backend.Kind)
	}
}

func TestRunChain_BackendInStepResult_OnError(t *testing.T) {
	idx := newMockIndex()

	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return nil, errors.New("boom")
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	_, results, err := runner.RunChain(context.Background(), []ChainStep{{ToolID: "mytool"}})
	if err == nil {
		t.Fatal("RunChain() should return error when handler fails")
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Backend.Kind != model.BackendKindLocal {
		t.Errorf("results[0].Backend.Kind = %q, want local", results[0].Backend.Kind)
	}
	if results[0].Backend.Local == nil || results[0].Backend.Local.Name != "myhandler" {
		t.Errorf("results[0].Backend.Local = %#v, want myhandler", results[0].Backend.Local)
	}
}
