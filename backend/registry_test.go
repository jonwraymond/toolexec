package backend

import "testing"

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
