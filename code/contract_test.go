package code

import (
	"context"
	"errors"
	"testing"
)

func TestToolsContract_ContextCancellation(t *testing.T) {
	cfg := &Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	tools := newTools(cfg, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tools.SearchTools(ctx, "query", 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SearchTools error = %v, want context.Canceled", err)
	}
}

func TestToolsContract_NilArgsRecorded(t *testing.T) {
	runner := &mockRunner{}
	cfg := &Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: &mockEngine{},
	}
	tools := newTools(cfg, 0, 0)

	_, _ = tools.RunTool(context.Background(), "tool:noop", nil)
	records := tools.GetToolCalls()
	if len(records) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(records))
	}
	if records[0].Args != nil {
		t.Fatalf("expected nil args recorded, got %v", records[0].Args)
	}
}

func TestExecutorContract_DeadlineWrapsLimit(t *testing.T) {
	engine := &mockEngine{executeErr: context.DeadlineExceeded}
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("NewDefaultExecutor failed: %v", err)
	}
	_, err = exec.ExecuteCode(context.Background(), ExecuteParams{Language: "go"})
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("ExecuteCode error = %v, want ErrLimitExceeded", err)
	}
}
