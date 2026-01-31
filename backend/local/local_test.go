package local

import (
	"context"
	"testing"

	"github.com/jonwraymond/toolexec/backend"
)

func TestLocalBackend_Interface(t *testing.T) {
	t.Helper()
	var _ backend.Backend = (*Backend)(nil)
}

func TestLocalBackend_Kind(t *testing.T) {
	b := New("test")
	if b.Kind() != "local" {
		t.Errorf("Kind() = %q, want %q", b.Kind(), "local")
	}
}

func TestLocalBackend_Name(t *testing.T) {
	b := New("my-local")
	if b.Name() != "my-local" {
		t.Errorf("Name() = %q, want %q", b.Name(), "my-local")
	}
}

func TestLocalBackend_RegisterHandler(t *testing.T) {
	b := New("test")

	handler := func(_ context.Context, _ map[string]any) (any, error) {
		return "handled", nil
	}

	b.RegisterHandler("my_tool", ToolDef{
		Name:        "my_tool",
		Description: "A test tool",
		Handler:     handler,
	})

	tools, err := b.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("ListTools() returned %d tools, want 1", len(tools))
	}

	if tools[0].Name != "my_tool" {
		t.Errorf("Tool.Name = %q, want %q", tools[0].Name, "my_tool")
	}
}

func TestLocalBackend_Execute(t *testing.T) {
	b := New("test")

	b.RegisterHandler("echo", ToolDef{
		Name:        "echo",
		Description: "Echo input",
		Handler: func(_ context.Context, args map[string]any) (any, error) {
			return args["message"], nil
		},
	})

	result, err := b.Execute(context.Background(), "echo", map[string]any{
		"message": "hello",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result != "hello" {
		t.Errorf("Execute() = %v, want %v", result, "hello")
	}
}

func TestLocalBackend_ExecuteNotFound(t *testing.T) {
	b := New("test")

	_, err := b.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("Execute() should fail for nonexistent tool")
	}
}
