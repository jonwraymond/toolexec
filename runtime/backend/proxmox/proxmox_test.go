package proxmox

import (
	"context"
	"errors"
	"testing"

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
	b := New(Config{})
	if b.Kind() != runtime.BackendProxmoxLXC {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendProxmoxLXC)
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

func TestBackendRequiresRuntimeEndpoint(t *testing.T) {
	b := New(Config{Node: "node-1", VMID: 100})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrRuntimeNotConfigured) {
		t.Errorf("Execute() missing runtime endpoint error = %v, want %v", err, ErrRuntimeNotConfigured)
	}
}

func TestBackendRequiresClient(t *testing.T) {
	b := New(Config{
		Node:            "node-1",
		VMID:            100,
		RuntimeEndpoint: "http://runtime",
	})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrClientNotConfigured) {
		t.Errorf("Execute() missing client error = %v, want %v", err, ErrClientNotConfigured)
	}
}
