package remote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{Endpoint: "http://localhost:8080"})
	if b.Kind() != runtime.BackendRemote {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendRemote)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{Endpoint: "http://localhost:8080"})
	if b.timeoutOverhead != 5*time.Second {
		t.Errorf("timeoutOverhead = %v, want %v", b.timeoutOverhead, 5*time.Second)
	}
	if b.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want %d", b.maxRetries, 3)
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{Endpoint: "http://localhost:8080"})
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

func TestBackendRequiresEndpoint(t *testing.T) {
	b := New(Config{Endpoint: ""}) // No endpoint
	ctx := context.Background()

	// Create a mock gateway
	gw := &mockGateway{}
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: gw,
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrRemoteNotAvailable) {
		t.Errorf("Execute() without endpoint error = %v, want %v", err, ErrRemoteNotAvailable)
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
