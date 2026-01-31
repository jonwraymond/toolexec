package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/jonwraymond/toolfoundation/model"
)

// ErrInvalidToolID is returned for malformed tool IDs.
var ErrInvalidToolID = errors.New("invalid tool ID format")

// Aggregator combines tools from multiple backends.
type Aggregator struct {
	registry *Registry
}

// NewAggregator creates a new tool aggregator.
func NewAggregator(registry *Registry) *Aggregator {
	return &Aggregator{registry: registry}
}

// ListAllTools returns tools from all enabled backends.
func (a *Aggregator) ListAllTools(ctx context.Context) ([]model.Tool, error) {
	backends := a.registry.ListEnabled()
	all := make([]model.Tool, 0)

	for _, b := range backends {
		tools, err := b.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for i := range tools {
			if tools[i].Namespace == "" {
				tools[i].Namespace = b.Name()
			}
			all = append(all, tools[i])
		}
	}

	return all, nil
}

// Execute invokes a tool through the backend registry.
func (a *Aggregator) Execute(ctx context.Context, toolID string, args map[string]any) (any, error) {
	backendName, tool, err := ParseToolID(toolID)
	if err != nil {
		return nil, err
	}
	if backendName == "" {
		return nil, ErrInvalidToolID
	}

	b, ok := a.registry.Get(backendName)
	if !ok {
		return nil, ErrBackendNotFound
	}
	if !b.Enabled() {
		return nil, ErrBackendDisabled
	}
	return b.Execute(ctx, tool, args)
}

// ParseToolID splits a tool ID into backend and tool name.
func ParseToolID(id string) (backendName, tool string, err error) {
	backendName, tool, err = model.ParseToolID(id)
	if err != nil {
		return "", "", ErrInvalidToolID
	}
	return backendName, tool, nil
}

// FormatToolID builds a tool ID from backend and tool name.
func FormatToolID(backendName, tool string) string {
	if backendName == "" {
		return tool
	}
	return fmt.Sprintf("%s:%s", backendName, tool)
}
