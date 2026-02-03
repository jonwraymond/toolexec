package containerd

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// mockGateway implements runtime.ToolGateway for testing.
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

func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})
	if b.Kind() != runtime.BackendContainerd {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendContainerd)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.imageRef != "toolruntime-sandbox:latest" {
		t.Errorf("imageRef = %q, want %q", b.imageRef, "toolruntime-sandbox:latest")
	}
	if b.namespace != "default" {
		t.Errorf("namespace = %q, want %q", b.namespace, "default")
	}
	if b.socketPath != "/run/containerd/containerd.sock" {
		t.Errorf("socketPath = %q, want %q", b.socketPath, "/run/containerd/containerd.sock")
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{})
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

func TestBackendRequiresClient(t *testing.T) {
	b := New(Config{})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrClientNotConfigured) {
		t.Errorf("Execute() without client error = %v, want %v", err, ErrClientNotConfigured)
	}
}

func TestBackendWithMockClient(t *testing.T) {
	mockRunner := &mockContainerRunner{
		runFunc: func(_ context.Context, spec ContainerSpec) (ContainerResult, error) {
			if spec.Image != "toolruntime-sandbox:latest" {
				t.Errorf("spec.Image = %q, want %q", spec.Image, "toolruntime-sandbox:latest")
			}
			return ContainerResult{
				ExitCode: 0,
				Stdout:   "__OUT__:{\"value\":42}",
			}, nil
		},
	}

	b := New(Config{Client: mockRunner})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}

	result, err := b.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Value == nil {
		t.Fatalf("Value is nil, want parsed __out")
	}
}

type mockContainerRunner struct {
	runFunc func(ctx context.Context, spec ContainerSpec) (ContainerResult, error)
}

func (m *mockContainerRunner) Run(ctx context.Context, spec ContainerSpec) (ContainerResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, spec)
	}
	return ContainerResult{}, nil
}
