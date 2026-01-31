package run

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerContract_ContextCancellation(t *testing.T) {
	runner := NewRunner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runner.Run(ctx, "any:tool", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
}

func TestRunnerContract_InvalidToolID(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Run(context.Background(), "", nil)
	if !errors.Is(err, ErrInvalidToolID) {
		t.Fatalf("Run error = %v, want ErrInvalidToolID", err)
	}
}

func TestProgressRunnerContract_Callbacks(t *testing.T) {
	runner := NewRunner()
	var events []ProgressEvent
	_, _ = runner.RunWithProgress(context.Background(), "", nil, func(ev ProgressEvent) {
		events = append(events, ev)
	})
	if len(events) < 2 {
		t.Fatalf("expected at least 2 progress events, got %d", len(events))
	}
	if events[0].Progress != 0 {
		t.Fatalf("expected first progress 0, got %v", events[0].Progress)
	}
	if events[len(events)-1].Progress != 1 {
		t.Fatalf("expected last progress 1, got %v", events[len(events)-1].Progress)
	}
}

func TestMCPExecutorContract_StreamChannel(t *testing.T) {
	exec := newMockMCPExecutor()
	exec.CallToolStreamChan = make(chan StreamEvent)
	ch, err := exec.CallToolStream(context.Background(), "server", nil)
	if err != nil {
		t.Fatalf("CallToolStream error: %v", err)
	}
	if ch == nil {
		t.Fatalf("expected non-nil stream channel")
	}
	close(exec.CallToolStreamChan)
}

func TestProviderExecutorContract_StreamChannel(t *testing.T) {
	exec := newMockProviderExecutor()
	exec.CallToolStreamChan = make(chan StreamEvent)
	ch, err := exec.CallToolStream(context.Background(), "provider", "tool", nil)
	if err != nil {
		t.Fatalf("CallToolStream error: %v", err)
	}
	if ch == nil {
		t.Fatalf("expected non-nil stream channel")
	}
	close(exec.CallToolStreamChan)
}

func TestLocalRegistryContract_NotFound(t *testing.T) {
	reg := newMockLocalRegistry()
	handler, ok := reg.Get("missing")
	if ok || handler != nil {
		t.Fatalf("expected (nil, false) for missing handler")
	}
}
