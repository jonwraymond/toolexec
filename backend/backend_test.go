package backend

import (
	"context"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockBackend implements Backend for testing.
//
//nolint:revive // test helper
type mockBackend struct {
	kind    string
	name    string
	enabled bool
	tools   []model.Tool
	execFn  func(ctx context.Context, tool string, args map[string]any) (any, error)
}

func (m *mockBackend) Kind() string  { return m.kind }
func (m *mockBackend) Name() string  { return m.name }
func (m *mockBackend) Enabled() bool { return m.enabled }

func (m *mockBackend) ListTools(_ context.Context) ([]model.Tool, error) {
	return m.tools, nil
}

func (m *mockBackend) Execute(ctx context.Context, tool string, args map[string]any) (any, error) {
	if m.execFn != nil {
		return m.execFn(ctx, tool, args)
	}
	return nil, nil
}

func (m *mockBackend) Start(_ context.Context) error { return nil }
func (m *mockBackend) Stop() error                   { return nil }

func TestBackend_Interface(t *testing.T) {
	t.Helper()
	var _ Backend = (*mockBackend)(nil)
}

func TestBackend_Methods(t *testing.T) {
	backend := &mockBackend{
		kind:    "local",
		name:    "test-backend",
		enabled: true,
		tools: []model.Tool{
			{Tool: mcp.Tool{Name: "test_tool", Description: "A test tool"}},
		},
		execFn: func(_ context.Context, _ string, _ map[string]any) (any, error) {
			return "executed", nil
		},
	}

	if backend.Kind() != "local" {
		t.Errorf("Kind() = %q, want %q", backend.Kind(), "local")
	}
	if backend.Name() != "test-backend" {
		t.Errorf("Name() = %q, want %q", backend.Name(), "test-backend")
	}
	if !backend.Enabled() {
		t.Error("Enabled() = false, want true")
	}

	tools, err := backend.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("ListTools() returned %d tools, want 1", len(tools))
	}

	result, err := backend.Execute(context.Background(), "test_tool", nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "executed" {
		t.Errorf("Execute() = %v, want %v", result, "executed")
	}
}
