package toolcodeengine

import (
	"context"

	"github.com/jonwraymond/toolexec/code"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// toolsGateway wraps code.Tools to implement runtime.ToolGateway.
type toolsGateway struct {
	tools code.Tools
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// WrapTools wraps a code.Tools implementation to satisfy runtime.ToolGateway.
// This allows the code.Tools interface to be used as a gateway in runtime.
func WrapTools(tools code.Tools) runtime.ToolGateway {
	return &toolsGateway{tools: tools}
}

// SearchTools implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return g.tools.SearchTools(ctx, query, limit)
}

// ListNamespaces implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) ListNamespaces(ctx context.Context) ([]string, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return g.tools.ListNamespaces(ctx)
}

// DescribeTool implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return tooldoc.ToolDoc{}, err
	}
	return g.tools.DescribeTool(ctx, id, level)
}

// ListToolExamples implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return g.tools.ListToolExamples(ctx, id, maxExamples)
}

// RunTool implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return run.RunResult{}, err
	}
	return g.tools.RunTool(ctx, id, args)
}

// RunChain implements runtime.ToolGateway by delegating to the wrapped Tools.
func (g *toolsGateway) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return run.RunResult{}, nil, err
	}
	return g.tools.RunChain(ctx, steps)
}
