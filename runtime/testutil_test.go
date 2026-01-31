package runtime

import (
	"context"
	"errors"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// errToolNotFound is used by mock gateway
var errToolNotFound = errors.New("tool not found")

// mockToolGateway is a minimal mock for testing
type mockToolGateway struct{}

func (m *mockToolGateway) SearchTools(ctx context.Context, _ string, _ int) ([]index.Summary, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, nil
}

func (m *mockToolGateway) ListNamespaces(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, nil
}

func (m *mockToolGateway) DescribeTool(ctx context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	if ctx.Err() != nil {
		return tooldoc.ToolDoc{}, ctx.Err()
	}
	// Return error for non-existent tools (anything we don't know about)
	return tooldoc.ToolDoc{}, errToolNotFound
}

func (m *mockToolGateway) ListToolExamples(ctx context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, nil
}

func (m *mockToolGateway) RunTool(ctx context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	if ctx.Err() != nil {
		return run.RunResult{}, ctx.Err()
	}
	return run.RunResult{}, nil
}

func (m *mockToolGateway) RunChain(ctx context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	if ctx.Err() != nil {
		return run.RunResult{}, nil, ctx.Err()
	}
	return run.RunResult{}, nil, nil
}
