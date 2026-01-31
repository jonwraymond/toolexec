package local

import (
	"context"
	"sync"

	"github.com/jonwraymond/toolexec/backend"
	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandlerFunc is the function signature for tool handlers.
type HandlerFunc func(ctx context.Context, args map[string]any) (any, error)

// ToolDef defines a local tool with its handler.
type ToolDef struct {
	Name         string
	Title        string
	Description  string
	InputSchema  map[string]any
	OutputSchema map[string]any
	Annotations  *mcp.ToolAnnotations
	Tags         []string
	Handler      HandlerFunc
}

// Backend implements the backend.Backend interface for local tool handlers.
type Backend struct {
	name     string
	enabled  bool
	handlers map[string]ToolDef
	mu       sync.RWMutex
}

// New creates a new local backend.
func New(name string) *Backend {
	return &Backend{
		name:     name,
		enabled:  true,
		handlers: make(map[string]ToolDef),
	}
}

// Kind returns the backend kind.
func (b *Backend) Kind() string {
	return "local"
}

// Name returns the backend instance name.
func (b *Backend) Name() string {
	return b.name
}

// Enabled returns whether the backend is enabled.
func (b *Backend) Enabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.enabled
}

// SetEnabled enables or disables the backend.
func (b *Backend) SetEnabled(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = enabled
}

// RegisterHandler registers a tool handler.
func (b *Backend) RegisterHandler(name string, def ToolDef) {
	if def.Name == "" {
		def.Name = name
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = def
}

// UnregisterHandler removes a tool handler.
func (b *Backend) UnregisterHandler(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, name)
}

// ListTools returns tools available from this backend.
func (b *Backend) ListTools(_ context.Context) ([]model.Tool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]model.Tool, 0, len(b.handlers))
	for _, def := range b.handlers {
		tool := model.Tool{
			Tool: mcp.Tool{
				Name:         def.Name,
				Title:        def.Title,
				Description:  def.Description,
				InputSchema:  def.InputSchema,
				OutputSchema: def.OutputSchema,
				Annotations:  def.Annotations,
			},
			Namespace: b.name,
			Tags:      model.NormalizeTags(def.Tags),
		}
		out = append(out, tool)
	}
	return out, nil
}

// Execute invokes a tool handler.
func (b *Backend) Execute(ctx context.Context, tool string, args map[string]any) (any, error) {
	b.mu.RLock()
	enabled := b.enabled
	def, ok := b.handlers[tool]
	b.mu.RUnlock()

	if !enabled {
		return nil, backend.ErrBackendDisabled
	}
	if !ok || def.Handler == nil {
		return nil, backend.ErrToolNotFound
	}
	return def.Handler(ctx, args)
}

// Start initializes the backend (no-op for local backend).
func (b *Backend) Start(_ context.Context) error {
	return nil
}

// Stop stops the backend (no-op for local backend).
func (b *Backend) Stop() error {
	return nil
}
