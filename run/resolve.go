package run

import (
	"context"
	"errors"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

// resolveResult holds the resolved tool and its available backends.
type resolveResult struct {
	tool     model.Tool
	backends []model.ToolBackend
}

// resolveTool resolves a tool ID to its definition and available backends.
// Resolution order:
//  1. If Index is configured, try Index.GetTool(id) and Index.GetAllBackends(id)
//  2. If not found (or Index not configured), try ToolResolver and BackendsResolver
//  3. Return ErrToolNotFound if all sources fail
func (r *DefaultRunner) resolveTool(ctx context.Context, toolID string) (*resolveResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var tool model.Tool
	var backends []model.ToolBackend
	var toolFound, backendsFound bool

	// 1. Try Index if configured
	if r.cfg.Index != nil {
		t, defaultBackend, err := r.cfg.Index.GetTool(toolID)
		if err == nil {
			tool = t
			toolFound = true

			// Get all backends for full selection
			allBackends, err := r.cfg.Index.GetAllBackends(toolID)
			if err == nil && len(allBackends) > 0 {
				backends = allBackends
				backendsFound = true
			} else if err != nil && !errors.Is(err, index.ErrNotFound) {
				// Unexpected error from Index
				return nil, err
			}
			if !backendsFound {
				// Fallback to just the default backend
				backends = []model.ToolBackend{defaultBackend}
				backendsFound = true
			}
		} else if !errors.Is(err, index.ErrNotFound) {
			// Unexpected error from Index
			return nil, err
		}
		// If ErrNotFound, fall through to resolvers
	}

	// 2. Try ToolResolver if tool not found
	if !toolFound && r.cfg.ToolResolver != nil {
		t, err := r.cfg.ToolResolver(toolID)
		if err == nil && t != nil {
			tool = *t
			toolFound = true
		}
	}

	// 3. Try BackendsResolver if backends not found
	if !backendsFound && r.cfg.BackendsResolver != nil {
		b, err := r.cfg.BackendsResolver(toolID)
		if err == nil && len(b) > 0 {
			backends = b
			backendsFound = true
		}
	}

	// 4. Check if we have what we need
	if !toolFound {
		return nil, ErrToolNotFound
	}
	if !backendsFound || len(backends) == 0 {
		return nil, ErrNoBackends
	}

	return &resolveResult{
		tool:     tool,
		backends: backends,
	}, nil
}

// selectBackend chooses the best backend from the available options.
// Uses the configured BackendSelector (defaults to local > provider > mcp).
func (r *DefaultRunner) selectBackend(backends []model.ToolBackend) (model.ToolBackend, error) {
	if len(backends) == 0 {
		return model.ToolBackend{}, ErrNoBackends
	}
	return r.cfg.BackendSelector(backends), nil
}
