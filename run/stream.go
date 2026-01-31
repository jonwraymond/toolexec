package run

import (
	"context"
	"fmt"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// dispatchStream executes a tool via the appropriate backend with streaming.
func (r *DefaultRunner) dispatchStream(ctx context.Context, tool model.Tool, backend model.ToolBackend, args map[string]any) (<-chan StreamEvent, error) {
	switch backend.Kind {
	case model.BackendKindMCP:
		return r.dispatchStreamMCP(ctx, tool, backend, args)
	case model.BackendKindProvider:
		return r.dispatchStreamProvider(ctx, tool, backend, args)
	case model.BackendKindLocal:
		return nil, ErrStreamNotSupported
	default:
		return nil, fmt.Errorf("unknown backend kind: %s", backend.Kind)
	}
}

// dispatchStreamMCP executes a tool via MCP with streaming.
func (r *DefaultRunner) dispatchStreamMCP(ctx context.Context, tool model.Tool, backend model.ToolBackend, args map[string]any) (<-chan StreamEvent, error) {
	if r.cfg.MCP == nil {
		return nil, fmt.Errorf("MCP executor not configured")
	}

	if backend.MCP == nil {
		return nil, fmt.Errorf("MCP backend missing server name")
	}

	// Build MCP params - use tool.Name, not ToolID()
	params := &mcp.CallToolParams{
		Name:      tool.Name,
		Arguments: args,
	}

	return r.cfg.MCP.CallToolStream(ctx, backend.MCP.ServerName, params)
}

// dispatchStreamProvider executes a tool via a provider with streaming.
func (r *DefaultRunner) dispatchStreamProvider(ctx context.Context, _ model.Tool, backend model.ToolBackend, args map[string]any) (<-chan StreamEvent, error) {
	if r.cfg.Provider == nil {
		return nil, fmt.Errorf("provider executor not configured")
	}

	if backend.Provider == nil {
		return nil, fmt.Errorf("provider backend missing details")
	}

	return r.cfg.Provider.CallToolStream(ctx, backend.Provider.ProviderID, backend.Provider.ToolID, args)
}
