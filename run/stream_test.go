package run

import (
	"context"
	"errors"
	"testing"
)

func TestRunStream_ValidatesInput(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	mockVal := newMockValidator()
	mockVal.ValidateInputErr = errors.New("input invalid")

	runner := NewRunner(
		WithIndex(idx),
		WithValidator(mockVal),
		WithValidation(true, false),
	)

	_, err := runner.RunStream(context.Background(), "mytool", nil)
	if err == nil {
		t.Error("RunStream() should fail when input validation fails")
	}

	if !errors.Is(err, ErrValidation) {
		t.Errorf("RunStream() error = %v, want ErrValidation", err)
	}
}

func TestRunStream_EmptyToolID(t *testing.T) {
	runner := NewRunner(WithIndex(newMockIndex()))

	_, err := runner.RunStream(context.Background(), "", nil)
	if err == nil {
		t.Fatal("RunStream() should fail when toolID is empty")
	}
	if !errors.Is(err, ErrInvalidToolID) {
		t.Errorf("RunStream() error = %v, want ErrInvalidToolID", err)
	}
}

func TestRunStream_ContextCancelled_BeforeExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	mcpExec := newMockMCPExecutor()

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	_, err := runner.RunStream(ctx, "mytool", nil)
	if err == nil {
		t.Fatal("RunStream() should return error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunStream() error = %v, want context.Canceled", err)
	}
	if mcpExec.CallCount != 0 {
		t.Errorf("RunStream() should not invoke executor when context is cancelled")
	}
}

func TestRunStream_MCP(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	eventChan := make(chan StreamEvent, 1)
	eventChan <- StreamEvent{Kind: StreamEventDone, ToolID: "mytool"}
	close(eventChan)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolStreamChan = eventChan

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	ch, err := runner.RunStream(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	// Read event from channel
	event, ok := <-ch
	if !ok {
		t.Fatal("channel closed unexpectedly")
	}
	if event.Kind != StreamEventDone {
		t.Errorf("event.Kind = %q, want %q", event.Kind, StreamEventDone)
	}
}

func TestRunStream_Provider(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testProviderBackend("myprovider", "tool1")
	mustRegisterTool(t, idx, tool, backend)
	idx.DefaultBackends["mytool"] = backend

	eventChan := make(chan StreamEvent, 1)
	eventChan <- StreamEvent{Kind: StreamEventChunk, Data: "partial"}
	close(eventChan)

	provExec := newMockProviderExecutor()
	provExec.CallToolStreamChan = eventChan

	runner := NewRunner(
		WithIndex(idx),
		WithProviderExecutor(provExec),
		WithValidation(false, false),
	)

	ch, err := runner.RunStream(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	event := <-ch
	if event.Kind != StreamEventChunk {
		t.Errorf("event.Kind = %q, want %q", event.Kind, StreamEventChunk)
	}
}

func TestRunStream_Local_NotSupported(t *testing.T) {
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

	_, err := runner.RunStream(context.Background(), "mytool", nil)
	if err == nil {
		t.Error("RunStream() should return error for local backend")
	}

	if !errors.Is(err, ErrStreamNotSupported) {
		t.Errorf("RunStream() error = %v, want ErrStreamNotSupported", err)
	}
}

func TestRunStream_StampsToolID_WhenMissing(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	// Executor returns events WITHOUT ToolID set
	eventChan := make(chan StreamEvent, 2)
	eventChan <- StreamEvent{Kind: StreamEventChunk, Data: "partial"}
	eventChan <- StreamEvent{Kind: StreamEventDone}
	close(eventChan)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolStreamChan = eventChan

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	ch, err := runner.RunStream(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	// Read all events and verify ToolID is stamped
	for event := range ch {
		if event.ToolID != "mytool" {
			t.Errorf("event.ToolID = %q, want %q (should be stamped when missing)", event.ToolID, "mytool")
		}
	}
}

func TestRunStream_PreservesExistingToolID(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	// Executor returns events WITH ToolID already set
	eventChan := make(chan StreamEvent, 1)
	eventChan <- StreamEvent{Kind: StreamEventDone, ToolID: "original-id"}
	close(eventChan)

	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolStreamChan = eventChan

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	ch, err := runner.RunStream(context.Background(), "mytool", nil)
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	event := <-ch
	// Should preserve the existing ToolID
	if event.ToolID != "original-id" {
		t.Errorf("event.ToolID = %q, want %q (should preserve existing)", event.ToolID, "original-id")
	}
}

func TestRunStream_ExecutorNotConfigured(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	runner := NewRunner(
		WithIndex(idx),
		// No MCP executor configured
		WithValidation(false, false),
	)

	_, err := runner.RunStream(context.Background(), "mytool", nil)
	if err == nil {
		t.Error("RunStream() should return error when executor not configured")
	}
}

func TestRunStream_NilChannel_IsError(t *testing.T) {
	idx := newMockIndex()
	tool := testTool("mytool")
	backend := testMCPBackend("server1")
	mustRegisterTool(t, idx, tool, backend)

	// Executor returns (nil, nil) for streaming, which should be treated as an error.
	mcpExec := newMockMCPExecutor()
	mcpExec.CallToolStreamChan = nil
	mcpExec.CallToolStreamErr = nil

	runner := NewRunner(
		WithIndex(idx),
		WithMCPExecutor(mcpExec),
		WithValidation(false, false),
	)

	_, err := runner.RunStream(context.Background(), "mytool", nil)
	if err == nil {
		t.Fatal("RunStream() should return an error when executor returns a nil channel")
	}
	if !errors.Is(err, ErrStreamNotSupported) {
		t.Errorf("RunStream() error = %v, want ErrStreamNotSupported", err)
	}
}
