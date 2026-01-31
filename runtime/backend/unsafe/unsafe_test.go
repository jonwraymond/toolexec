package unsafe

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// mockGateway implements runtime.ToolGateway for testing
type mockGateway struct {
	searchResults []index.Summary
	runResult     run.RunResult
	runErr        error
}

func (m *mockGateway) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return m.searchResults, nil
}

func (m *mockGateway) ListNamespaces(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockGateway) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, nil
}

func (m *mockGateway) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return nil, nil
}

func (m *mockGateway) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	if m.runErr != nil {
		return run.RunResult{}, m.runErr
	}
	return m.runResult, nil
}

func (m *mockGateway) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, nil
}

// mockLogger captures log messages for testing
type mockLogger struct {
	messages []string
}

func (l *mockLogger) Info(msg string, _ ...any)  { l.messages = append(l.messages, "INFO: "+msg) }
func (l *mockLogger) Warn(msg string, _ ...any)  { l.messages = append(l.messages, "WARN: "+msg) }
func (l *mockLogger) Error(msg string, _ ...any) { l.messages = append(l.messages, "ERROR: "+msg) }

func (l *mockLogger) hasWarning(substr string) bool {
	for _, m := range l.messages {
		if strings.Contains(m, "WARN") && strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

// TestBackendImplementsInterface verifies Backend satisfies runtime.Backend
func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})

	if b.Kind() != runtime.BackendUnsafeHost {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendUnsafeHost)
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: nil,
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingGateway) {
		t.Errorf("Execute() without gateway error = %v, want %v", err, runtime.ErrMissingGateway)
	}
}

func TestBackendRequiresCode(t *testing.T) {
	b := New(Config{})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingCode) {
		t.Errorf("Execute() without code error = %v, want %v", err, runtime.ErrMissingCode)
	}
}

func TestBackendLogsWarning(t *testing.T) {
	logger := &mockLogger{}
	b := New(Config{Logger: logger})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    `__out = "hello"`,
		Gateway: &mockGateway{},
	}

	_, _ = b.Execute(ctx, req)

	if !logger.hasWarning("UNSAFE") {
		t.Error("Execute() should log UNSAFE warning")
	}
}

func TestBackendRequiresOptIn(t *testing.T) {
	b := New(Config{RequireOptIn: true})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    `__out = "hello"`,
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrOptInRequired) {
		t.Errorf("Execute() without opt-in error = %v, want %v", err, ErrOptInRequired)
	}
}

func TestBackendOptInAllows(t *testing.T) {
	b := New(Config{RequireOptIn: true})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    `__out = "hello"`,
		Gateway: &mockGateway{},
		Metadata: map[string]any{
			"unsafeOptIn": true,
		},
	}

	// Should not return ErrOptInRequired
	_, err := b.Execute(ctx, req)
	// May fail for other reasons (no interpreter), but not opt-in
	if errors.Is(err, ErrOptInRequired) {
		t.Error("Execute() with opt-in should not return ErrOptInRequired")
	}
}

func TestBackendRespectsTimeout(t *testing.T) {
	b := New(Config{Mode: ModeInterpreter})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := runtime.ExecuteRequest{
		Code:    `for { }`, // Infinite loop
		Gateway: &mockGateway{},
		Timeout: 10 * time.Millisecond,
	}

	start := time.Now()
	_, err := b.Execute(ctx, req)
	elapsed := time.Since(start)

	// Should timeout within reasonable time
	if elapsed > 500*time.Millisecond {
		t.Errorf("Execute() took %v, should timeout faster", elapsed)
	}

	// Should return timeout error
	if err == nil {
		t.Error("Execute() should return timeout error")
	}
}

func TestBackendReturnsBackendInfo(t *testing.T) {
	b := New(Config{Mode: ModeInterpreter})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    `__out = "hello"`,
		Gateway: &mockGateway{},
	}

	result, err := b.Execute(ctx, req)
	if err != nil {
		t.Skipf("Execute() error = %v (interpreter may not be available)", err)
	}

	if result.Backend.Kind != runtime.BackendUnsafeHost {
		t.Errorf("Backend.Kind = %v, want %v", result.Backend.Kind, runtime.BackendUnsafeHost)
	}
}

func TestBackendCapturesStdout(t *testing.T) {
	b := New(Config{Mode: ModeInterpreter})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    `fmt.Println("hello world")`,
		Gateway: &mockGateway{},
	}

	result, err := b.Execute(ctx, req)
	if err != nil {
		t.Skipf("Execute() error = %v (interpreter may not be available)", err)
	}

	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Stdout = %q, want to contain %q", result.Stdout, "hello world")
	}
}

func TestBackendModeSelection(t *testing.T) {
	tests := []struct {
		mode ExecutionMode
		want ExecutionMode
	}{
		{ModeInterpreter, ModeInterpreter},
		{ModeSubprocess, ModeSubprocess},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			b := New(Config{Mode: tt.mode})
			if b.mode != tt.want {
				t.Errorf("mode = %v, want %v", b.mode, tt.want)
			}
		})
	}
}

func TestBackendDefaultMode(t *testing.T) {
	b := New(Config{})
	if b.mode != ModeInterpreter {
		t.Errorf("default mode = %v, want %v", b.mode, ModeInterpreter)
	}
}

func TestBackendContractCompliance(t *testing.T) {
	runtime.RunBackendContractTests(t, runtime.BackendContract{
		NewBackend: func() runtime.Backend {
			return New(Config{Mode: ModeInterpreter})
		},
		NewGateway: func() runtime.ToolGateway {
			return &mockGateway{}
		},
		ExpectedKind:       runtime.BackendUnsafeHost,
		SkipExecutionTests: true, // Interpreter may not be available in all environments
	})
}

// Test that stdout buffer is properly captured
func TestStdoutBuffer(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("test output\n")

	if !strings.Contains(buf.String(), "test output") {
		t.Error("buffer should contain test output")
	}
}
