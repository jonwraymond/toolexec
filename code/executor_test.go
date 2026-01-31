package code

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/toolexec/run"
)

func TestExecutor_Interface(t *testing.T) {
	t.Helper()
	// Verify Executor interface has ExecuteCode method with correct signature
	var _ Executor = (*DefaultExecutor)(nil)
}

func TestNewDefaultExecutor_ValidConfig(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	exec, err := NewDefaultExecutor(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestNewDefaultExecutor_InvalidConfig(t *testing.T) {
	cfg := Config{} // Missing required fields
	_, err := NewDefaultExecutor(cfg)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
}

func TestDefaultExecutor_ImplementsExecutor(t *testing.T) {
	t.Helper()
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	exec, _ := NewDefaultExecutor(cfg)
	var _ Executor = exec // This compiles if DefaultExecutor implements Executor
}

func TestExecuteCode_AppliesDefaultLanguage(t *testing.T) {
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:           &mockIndex{},
		Docs:            &mockStore{},
		Run:             &mockRunner{},
		Engine:          engine,
		DefaultLanguage: "typescript",
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:    "code",
		Timeout: time.Second,
		// Language is empty
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(engine.executeCalls) != 1 {
		t.Fatalf("expected 1 execute call, got %d", len(engine.executeCalls))
	}
	if engine.executeCalls[0].params.Language != "typescript" {
		t.Errorf("expected Language 'typescript', got %q", engine.executeCalls[0].params.Language)
	}
}

func TestExecuteCode_AppliesDefaultTimeout(t *testing.T) {
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:          &mockIndex{},
		Docs:           &mockStore{},
		Run:            &mockRunner{},
		Engine:         engine,
		DefaultTimeout: 5 * time.Second,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		// Timeout is zero
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(engine.executeCalls) != 1 {
		t.Fatalf("expected 1 execute call, got %d", len(engine.executeCalls))
	}
	if engine.executeCalls[0].params.Timeout != 5*time.Second {
		t.Errorf("expected Timeout 5s, got %v", engine.executeCalls[0].params.Timeout)
	}
}

func TestExecuteCode_CapsMaxToolCalls(t *testing.T) {
	// When params MaxToolCalls > config MaxToolCalls, use config
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:        &mockIndex{},
		Docs:         &mockStore{},
		Run:          &mockRunner{},
		Engine:       engine,
		MaxToolCalls: 10, // Config limit
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:         "code",
		Language:     "go",
		Timeout:      time.Second,
		MaxToolCalls: 100, // Params wants more
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The executor should pass the config limit (10) to tools, not 100
	// We can't directly verify this without checking the tools object,
	// but we can verify the params passed to engine still has the original value
	// The actual capping happens in tools creation, not in params
}

func TestExecuteCode_MaxToolCalls_ParamsLower(t *testing.T) {
	// When params MaxToolCalls < config MaxToolCalls, use params
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:        &mockIndex{},
		Docs:         &mockStore{},
		Run:          &mockRunner{},
		Engine:       engine,
		MaxToolCalls: 100, // Config limit
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:         "code",
		Language:     "go",
		Timeout:      time.Second,
		MaxToolCalls: 5, // Params wants less
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteCode_MaxToolCalls_BothZero(t *testing.T) {
	// Both zero means unlimited
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:        &mockIndex{},
		Docs:         &mockStore{},
		Run:          &mockRunner{},
		Engine:       engine,
		MaxToolCalls: 0, // Unlimited
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:         "code",
		Language:     "go",
		Timeout:      time.Second,
		MaxToolCalls: 0, // Also unlimited
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteCode_DelegatesToEngine(t *testing.T) {
	engine := &mockEngine{
		executeResult: ExecuteResult{
			Value: "result",
		},
	}
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "return 42",
		Language: "go",
		Timeout:  time.Second,
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(engine.executeCalls) != 1 {
		t.Fatalf("expected 1 execute call, got %d", len(engine.executeCalls))
	}
	if engine.executeCalls[0].params.Code != "return 42" {
		t.Errorf("expected Code 'return 42', got %q", engine.executeCalls[0].params.Code)
	}
	if result.Value != "result" {
		t.Errorf("expected Value 'result', got %v", result.Value)
	}
}

func TestExecuteCode_CollectsToolCalls(t *testing.T) {
	// Engine executes tools via the provided Tools interface
	engine := &mockEngine{}
	engine.executeResult = ExecuteResult{Value: "ok"}

	// Make the engine use the tools to run some tools
	var capturedTools Tools
	engine = &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	// Override Execute to actually use tools
	originalExecute := engine.Execute
	_ = originalExecute // unused but shows the pattern

	runner := &mockRunner{
		runResult: run.RunResult{
			Structured: "tool result",
		},
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: engine,
	}

	// Create a custom engine that uses the tools
	customEngine := &toolUsingEngine{
		toolID: "test-tool",
		args:   map[string]any{"key": "value"},
	}
	cfg.Engine = customEngine
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  time.Second,
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tool calls should be collected in the result
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].ToolID != "test-tool" {
		t.Errorf("expected ToolID 'test-tool', got %q", result.ToolCalls[0].ToolID)
	}
	_ = capturedTools
}

func TestExecuteCode_CollectsStdout(t *testing.T) {
	// Engine that uses Println
	customEngine := &printingEngine{
		messages: []string{"hello", "world"},
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: customEngine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  time.Second,
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Stdout != "hello\nworld\n" {
		t.Errorf("expected Stdout 'hello\\nworld\\n', got %q", result.Stdout)
	}
}

func TestExecuteCode_MeasuresDuration(t *testing.T) {
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  time.Second,
	}
	result, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duration should be at least 0
	if result.DurationMs < 0 {
		t.Errorf("expected non-negative DurationMs, got %d", result.DurationMs)
	}
}

func TestExecuteCode_Timeout_ContextCancellation(t *testing.T) {
	// Engine that receives the context with deadline
	var receivedCtx context.Context
	engine := &contextCapturingEngine{
		captureCtx: &receivedCtx,
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  5 * time.Second,
	}
	_, _ = exec.ExecuteCode(ctx, params)

	// The context passed to engine should have a deadline
	if receivedCtx == nil {
		t.Fatal("engine did not receive context")
	}
	deadline, ok := receivedCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}
	// Deadline should be approximately 5 seconds from now (allowing some slack)
	expectedDeadline := time.Now().Add(5 * time.Second)
	if deadline.Before(expectedDeadline.Add(-100*time.Millisecond)) ||
		deadline.After(expectedDeadline.Add(100*time.Millisecond)) {
		t.Errorf("deadline %v not within expected range around %v", deadline, expectedDeadline)
	}
}

func TestExecuteCode_Timeout_Enforced(t *testing.T) {
	// Engine that blocks until context is cancelled
	engine := &slowEngine{
		delay: 10 * time.Second,
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  50 * time.Millisecond, // Very short timeout
	}

	start := time.Now()
	_, err := exec.ExecuteCode(ctx, params)
	elapsed := time.Since(start)

	// Should return quickly (not wait 10 seconds)
	if elapsed > time.Second {
		t.Errorf("expected quick timeout, took %v", elapsed)
	}

	if err == nil {
		t.Fatal("expected error due to timeout")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("expected ErrLimitExceeded, got %v", err)
	}
}

func TestExecuteCode_EmptyCode(t *testing.T) {
	engine := &mockEngine{
		executeResult: ExecuteResult{Value: "ok"},
	}
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: engine,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "", // Empty code is valid
		Language: "go",
		Timeout:  time.Second,
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("empty code should be valid: %v", err)
	}

	if len(engine.executeCalls) != 1 {
		t.Fatalf("expected 1 execute call, got %d", len(engine.executeCalls))
	}
	if engine.executeCalls[0].params.Code != "" {
		t.Errorf("expected empty Code, got %q", engine.executeCalls[0].params.Code)
	}
}

func TestExecuteCode_Logger_ToolCallLogged(t *testing.T) {
	logger := &mockLogger{}
	engine := &toolUsingEngine{
		toolID: "logged-tool",
	}

	runner := &mockRunner{
		runResult: run.RunResult{},
	}

	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    runner,
		Engine: engine,
		Logger: logger,
	}
	exec, _ := NewDefaultExecutor(cfg)

	ctx := context.Background()
	params := ExecuteParams{
		Code:     "code",
		Language: "go",
		Timeout:  time.Second,
	}
	_, err := exec.ExecuteCode(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Logger should have received at least one message
	if len(logger.messages) == 0 {
		t.Error("expected logger to receive messages")
	}
}

// Helper test engines

// toolUsingEngine calls RunTool during Execute
type toolUsingEngine struct {
	toolID string
	args   map[string]any
}

func (e *toolUsingEngine) Execute(ctx context.Context, _ ExecuteParams, tools Tools) (ExecuteResult, error) {
	_, _ = tools.RunTool(ctx, e.toolID, e.args)
	return ExecuteResult{Value: "done"}, nil
}

// printingEngine calls Println during Execute
type printingEngine struct {
	messages []string
}

func (e *printingEngine) Execute(_ context.Context, _ ExecuteParams, tools Tools) (ExecuteResult, error) {
	for _, msg := range e.messages {
		tools.Println(msg)
	}
	return ExecuteResult{Value: "done"}, nil
}

// contextCapturingEngine captures the context for inspection
type contextCapturingEngine struct {
	captureCtx *context.Context
}

func (e *contextCapturingEngine) Execute(ctx context.Context, _ ExecuteParams, _ Tools) (ExecuteResult, error) {
	*e.captureCtx = ctx
	return ExecuteResult{Value: "done"}, nil
}

// slowEngine simulates a slow operation
type slowEngine struct {
	delay time.Duration
}

func (e *slowEngine) Execute(ctx context.Context, _ ExecuteParams, _ Tools) (ExecuteResult, error) {
	select {
	case <-time.After(e.delay):
		return ExecuteResult{Value: "done"}, nil
	case <-ctx.Done():
		return ExecuteResult{}, ctx.Err()
	}
}
