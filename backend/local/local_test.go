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

func TestLocalBackend_Enabled(t *testing.T) {
	b := New("test")

	if !b.Enabled() {
		t.Error("Enabled() should be true by default")
	}
}

func TestLocalBackend_SetEnabled(t *testing.T) {
	b := New("test")

	b.SetEnabled(false)
	if b.Enabled() {
		t.Error("Enabled() should be false after SetEnabled(false)")
	}

	b.SetEnabled(true)
	if !b.Enabled() {
		t.Error("Enabled() should be true after SetEnabled(true)")
	}
}

func TestLocalBackend_UnregisterHandler(t *testing.T) {
	b := New("test")

	b.RegisterHandler("tool", ToolDef{
		Name:    "tool",
		Handler: func(_ context.Context, _ map[string]any) (any, error) { return nil, nil },
	})

	tools, _ := b.ListTools(context.Background())
	if len(tools) != 1 {
		t.Fatalf("ListTools() returned %d tools, want 1", len(tools))
	}

	b.UnregisterHandler("tool")

	tools, _ = b.ListTools(context.Background())
	if len(tools) != 0 {
		t.Errorf("ListTools() returned %d tools after unregister, want 0", len(tools))
	}
}

func TestLocalBackend_ExecuteDisabled(t *testing.T) {
	b := New("test")

	b.RegisterHandler("tool", ToolDef{
		Name:    "tool",
		Handler: func(_ context.Context, _ map[string]any) (any, error) { return "ok", nil },
	})

	b.SetEnabled(false)

	_, err := b.Execute(context.Background(), "tool", nil)
	if err != backend.ErrBackendDisabled {
		t.Errorf("Execute() error = %v, want ErrBackendDisabled", err)
	}
}

func TestLocalBackend_ExecuteNilHandler(t *testing.T) {
	b := New("test")

	b.RegisterHandler("tool", ToolDef{
		Name:    "tool",
		Handler: nil, // No handler
	})

	_, err := b.Execute(context.Background(), "tool", nil)
	if err != backend.ErrToolNotFound {
		t.Errorf("Execute() error = %v, want ErrToolNotFound", err)
	}
}

func TestLocalBackend_RegisterHandler_DefaultsName(t *testing.T) {
	b := New("test")

	// Register with empty Name in ToolDef - should default to key
	b.RegisterHandler("my_tool", ToolDef{
		Description: "A test tool",
		Handler:     func(_ context.Context, _ map[string]any) (any, error) { return nil, nil },
	})

	tools, _ := b.ListTools(context.Background())
	if len(tools) != 1 {
		t.Fatalf("ListTools() returned %d tools, want 1", len(tools))
	}

	if tools[0].Name != "my_tool" {
		t.Errorf("Tool.Name = %q, want %q (should default to key)", tools[0].Name, "my_tool")
	}
}

func TestLocalBackend_Start(t *testing.T) {
	b := New("test")

	err := b.Start(context.Background())
	if err != nil {
		t.Errorf("Start() error = %v, want nil", err)
	}
}

func TestLocalBackend_Stop(t *testing.T) {
	b := New("test")

	err := b.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}
