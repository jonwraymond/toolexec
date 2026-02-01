package backend

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	b := &mockBackend{kind: "local", name: "test", enabled: true}

	if err := registry.Register(b); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := registry.Register(b); err == nil {
		t.Error("Register() should fail on duplicate")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	b := &mockBackend{kind: "local", name: "test", enabled: true}
	_ = registry.Register(b)

	got, ok := registry.Get("test")
	if !ok {
		t.Fatal("Get() returned false")
	}
	if got.Name() != "test" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "test")
	}

	if _, ok := registry.Get("nonexistent"); ok {
		t.Error("Get() should return false for nonexistent backend")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{kind: "local", name: "a", enabled: true})
	_ = registry.Register(&mockBackend{kind: "mcp", name: "b", enabled: true})
	_ = registry.Register(&mockBackend{kind: "http", name: "c", enabled: false})

	all := registry.List()
	if len(all) != 3 {
		t.Errorf("List() returned %d backends, want 3", len(all))
	}

	enabled := registry.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("ListEnabled() returned %d backends, want 2", len(enabled))
	}
}

func TestRegistry_ListByKind(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{kind: "local", name: "local1", enabled: true})
	_ = registry.Register(&mockBackend{kind: "local", name: "local2", enabled: true})
	_ = registry.Register(&mockBackend{kind: "mcp", name: "mcp1", enabled: true})

	locals := registry.ListByKind("local")
	if len(locals) != 2 {
		t.Errorf("ListByKind(local) returned %d backends, want 2", len(locals))
	}

	mcps := registry.ListByKind("mcp")
	if len(mcps) != 1 {
		t.Errorf("ListByKind(mcp) returned %d backends, want 1", len(mcps))
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	b := &mockBackend{kind: "local", name: "test", enabled: true}
	_ = registry.Register(b)

	registry.Unregister("test")

	if _, ok := registry.Get("test"); ok {
		t.Error("Get() should return false after Unregister()")
	}
}

func TestRegistry_Names(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{kind: "local", name: "zz", enabled: true})
	_ = registry.Register(&mockBackend{kind: "local", name: "aa", enabled: true})
	_ = registry.Register(&mockBackend{kind: "local", name: "mm", enabled: true})

	names := registry.Names()

	if len(names) != 3 {
		t.Fatalf("Names() returned %d names, want 3", len(names))
	}

	// Should be sorted
	if names[0] != "aa" || names[1] != "mm" || names[2] != "zz" {
		t.Errorf("Names() = %v, want [aa mm zz]", names)
	}
}

func TestRegistry_StartAll(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{kind: "local", name: "ok1", enabled: true})
	_ = registry.Register(&mockBackend{kind: "local", name: "ok2", enabled: true})

	err := registry.StartAll(context.Background())
	if err != nil {
		t.Errorf("StartAll() error = %v", err)
	}
}

func TestRegistry_StartAll_Error(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{
		kind:     "local",
		name:     "failing",
		enabled:  true,
		startErr: errors.New("start failed"),
	})

	err := registry.StartAll(context.Background())
	if err == nil {
		t.Error("StartAll() should propagate error")
	}
}

func TestRegistry_StopAll(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{kind: "local", name: "ok1", enabled: true})
	_ = registry.Register(&mockBackend{kind: "local", name: "ok2", enabled: true})

	err := registry.StopAll()
	if err != nil {
		t.Errorf("StopAll() error = %v", err)
	}
}

func TestRegistry_StopAll_Error(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{
		kind:    "local",
		name:    "failing",
		enabled: true,
		stopErr: errors.New("stop failed"),
	})

	err := registry.StopAll()
	if err == nil {
		t.Error("StopAll() should propagate error")
	}
}

func TestRegistry_RegisterFactory(t *testing.T) {
	registry := NewRegistry()

	factory := func(name string) (Backend, error) {
		return &mockBackend{kind: "custom", name: name, enabled: true}, nil
	}

	// Should not panic on empty kind or nil factory
	registry.RegisterFactory("", factory)
	registry.RegisterFactory("custom", nil)

	// Valid registration
	registry.RegisterFactory("custom", factory)
}

func TestRegistry_Register_Nil(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(nil)
	if err == nil {
		t.Error("Register(nil) should return error")
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(&mockBackend{kind: "local", name: "", enabled: true})
	if err == nil {
		t.Error("Register() with empty name should return error")
	}
}
