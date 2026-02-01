package run

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/toolfoundation/model"
)

// dispatchResult holds the result of dispatching to a backend.
type dispatchResult struct {
	// structured is the structured result from the executor.
	// For MCP, this is either StructuredContent or best-effort parsed content.
	// For provider/local, this is the executor return value.
	structured any

	// mcpResult is the raw MCP result when the backend was MCP.
	mcpResult *mcp.CallToolResult
}

// dispatch executes a tool via the appropriate backend.
func (r *DefaultRunner) dispatch(ctx context.Context, tool model.Tool, backend model.ToolBackend, args map[string]any) (*dispatchResult, error) {
	switch backend.Kind {
	case model.BackendKindMCP:
		return r.dispatchMCP(ctx, tool, backend, args)
	case model.BackendKindProvider:
		return r.dispatchProvider(ctx, tool, backend, args)
	case model.BackendKindLocal:
		return r.dispatchLocal(ctx, tool, backend, args)
	default:
		return nil, fmt.Errorf("unknown backend kind: %s", backend.Kind)
	}
}

// dispatchMCP executes a tool via an MCP server.
func (r *DefaultRunner) dispatchMCP(ctx context.Context, tool model.Tool, backend model.ToolBackend, args map[string]any) (*dispatchResult, error) {
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

	result, err := r.cfg.MCP.CallTool(ctx, backend.MCP.ServerName, params)
	if err != nil {
		return nil, err
	}

	return &dispatchResult{
		mcpResult: result,
	}, nil
}

// dispatchProvider executes a tool via a provider.
func (r *DefaultRunner) dispatchProvider(ctx context.Context, _ model.Tool, backend model.ToolBackend, args map[string]any) (*dispatchResult, error) {
	if r.cfg.Provider == nil {
		return nil, fmt.Errorf("provider executor not configured")
	}

	if backend.Provider == nil {
		return nil, fmt.Errorf("provider backend missing details")
	}

	result, err := r.cfg.Provider.CallTool(ctx, backend.Provider.ProviderID, backend.Provider.ToolID, args)
	if err != nil {
		return nil, err
	}

	return &dispatchResult{
		structured: result,
	}, nil
}

// dispatchLocal executes a tool via a local handler.
func (r *DefaultRunner) dispatchLocal(ctx context.Context, _ model.Tool, backend model.ToolBackend, args map[string]any) (*dispatchResult, error) {
	if r.cfg.Local == nil {
		return nil, fmt.Errorf("local registry not configured")
	}

	if backend.Local == nil {
		return nil, fmt.Errorf("local backend missing name")
	}

	handler, ok := r.cfg.Local.Get(backend.Local.Name)
	if !ok || handler == nil {
		return nil, fmt.Errorf("local handler %q not found", backend.Local.Name)
	}

	result, err := handler(ctx, args)
	if err != nil {
		return nil, err
	}

	return &dispatchResult{
		structured: result,
	}, nil
}
