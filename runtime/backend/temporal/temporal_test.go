package temporal

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// mockBackend implements runtime.Backend for testing
type mockBackend struct{}

func (m *mockBackend) Kind() runtime.BackendKind {
	return runtime.BackendDocker
}

func (m *mockBackend) Execute(_ context.Context, _ runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	return runtime.ExecuteResult{}, nil
}

func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})
	if b.Kind() != runtime.BackendTemporal {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendTemporal)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.hostPort != "localhost:7233" {
		t.Errorf("hostPort = %q, want %q", b.hostPort, "localhost:7233")
	}
	if b.namespace != "default" {
		t.Errorf("namespace = %q, want %q", b.namespace, "default")
	}
	if b.taskQueue != "toolruntime-execution" {
		t.Errorf("taskQueue = %q, want %q", b.taskQueue, "toolruntime-execution")
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{
		SandboxBackend: &mockBackend{},
	})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: nil,
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingGateway) {
		t.Errorf("Execute() without gateway error = %v, want %v", err, runtime.ErrMissingGateway)
	}
}

func TestBackendRequiresSandboxBackend(t *testing.T) {
	b := New(Config{
		SandboxBackend: nil, // No sandbox backend
	})
	ctx := context.Background()

	// Create a mock gateway
	gw := &mockGateway{}
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: gw,
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrMissingSandboxBackend) {
		t.Errorf("Execute() without sandbox backend error = %v, want %v", err, ErrMissingSandboxBackend)
	}
}

// mockGateway implements runtime.ToolGateway for testing
type mockGateway struct{}

func (m *mockGateway) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, nil
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
	return run.RunResult{}, nil
}
func (m *mockGateway) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, nil
}
