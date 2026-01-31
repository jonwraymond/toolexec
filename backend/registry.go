package backend

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrBackendExists is returned when registering a duplicate backend.
var ErrBackendExists = errors.New("backend already registered")

// Registry manages backend instances.
type Registry struct {
	mu        sync.RWMutex
	backends  map[string]Backend
	factories map[string]Factory
}

// NewRegistry creates a new backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends:  make(map[string]Backend),
		factories: make(map[string]Factory),
	}
}

// RegisterFactory registers a factory for a backend kind.
func (r *Registry) RegisterFactory(kind string, factory Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if kind == "" || factory == nil {
		return
	}
	r.factories[kind] = factory
}

// Register adds a backend to the registry.
func (r *Registry) Register(b Backend) error {
	if b == nil {
		return fmt.Errorf("backend is nil")
	}
	name := b.Name()
	if name == "" {
		return fmt.Errorf("backend name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.backends[name]; exists {
		return fmt.Errorf("%w: %s", ErrBackendExists, name)
	}
	r.backends[name] = b
	return nil
}

// Unregister removes a backend from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, exists := r.backends[name]; exists {
		_ = b.Stop()
		delete(r.backends, name)
	}
}

// Get retrieves a backend by name.
func (r *Registry) Get(name string) (Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[name]
	return b, ok
}

// List returns all backends.
func (r *Registry) List() []Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Backend, 0, len(r.backends))
	for _, b := range r.backends {
		out = append(out, b)
	}
	return out
}

// ListEnabled returns enabled backends only.
func (r *Registry) ListEnabled() []Backend {
	all := r.List()
	out := make([]Backend, 0, len(all))
	for _, b := range all {
		if b.Enabled() {
			out = append(out, b)
		}
	}
	return out
}

// ListByKind returns backends matching the given kind.
func (r *Registry) ListByKind(kind string) []Backend {
	all := r.List()
	out := make([]Backend, 0, len(all))
	for _, b := range all {
		if b.Kind() == kind {
			out = append(out, b)
		}
	}
	return out
}

// Names returns backend names sorted for deterministic output.
func (r *Registry) Names() []string {
	all := r.List()
	out := make([]string, 0, len(all))
	for _, b := range all {
		out = append(out, b.Name())
	}
	sort.Strings(out)
	return out
}

// StartAll starts all backends.
func (r *Registry) StartAll(ctx context.Context) error {
	for _, b := range r.ListEnabled() {
		if err := b.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all backends.
func (r *Registry) StopAll() error {
	for _, b := range r.List() {
		if err := b.Stop(); err != nil {
			return err
		}
	}
	return nil
}
