package run

import (
	"context"
	"testing"
)

func TestRunWithProgress_EmitsStartAndEnd(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testLocalBackend("myhandler")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	localReg := newMockLocalRegistry()
	localReg.Register("myhandler", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "ok"}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	var events []ProgressEvent
	_, err := runner.RunWithProgress(context.Background(), "mytool", nil, func(ev ProgressEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("RunWithProgress() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 progress events, got %d", len(events))
	}
	if events[0].Progress != 0 {
		t.Errorf("start progress = %v, want 0", events[0].Progress)
	}
	if events[1].Progress != 1 {
		t.Errorf("end progress = %v, want 1", events[1].Progress)
	}
}

func TestRunChainWithProgress_EmitsStepProgress(t *testing.T) {
	idx := newMockIndex()
	tool1 := testTool("tool1")
	tool2 := testTool("tool2")
	backend1 := testLocalBackend("h1")
	backend2 := testLocalBackend("h2")
	mustRegisterTool(t, idx, tool1, backend1)
	mustRegisterTool(t, idx, tool2, backend2)
	idx.DefaultBackends["tool1"] = backend1
	idx.DefaultBackends["tool2"] = backend2

	localReg := newMockLocalRegistry()
	localReg.Register("h1", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "step1"}, nil
	})
	localReg.Register("h2", func(_ context.Context, _ map[string]any) (any, error) {
		return map[string]any{"result": "step2"}, nil
	})

	runner := NewRunner(
		WithIndex(idx),
		WithLocalRegistry(localReg),
		WithValidation(false, false),
	)

	steps := []ChainStep{
		{ToolID: "tool1"},
		{ToolID: "tool2"},
	}

	var events []ProgressEvent
	_, _, err := runner.RunChainWithProgress(context.Background(), steps, func(ev ProgressEvent) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("RunChainWithProgress() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 progress events, got %d", len(events))
	}
	if events[0].Progress != 0 {
		t.Errorf("start progress = %v, want 0", events[0].Progress)
	}
	if events[1].Progress != 1 || events[2].Progress != 2 {
		t.Errorf("step progress = %v, want [1 2]", []float64{events[1].Progress, events[2].Progress})
	}
	if events[2].Total != 2 {
		t.Errorf("total = %v, want 2", events[2].Total)
	}
}
