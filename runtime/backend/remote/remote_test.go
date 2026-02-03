package remote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

type mockGateway struct{}

func (m *mockGateway) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, nil
}
func (m *mockGateway) ListNamespaces(_ context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockGateway) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, nil
}
func (m *mockGateway) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return nil, nil
}
func (m *mockGateway) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return run.RunResult{}, nil
}
func (m *mockGateway) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, nil
}

type stubClient struct {
	response RemoteResponse
	err      error
	seen     RemoteRequest
}

func (s *stubClient) Execute(_ context.Context, req RemoteRequest) (RemoteResponse, error) {
	s.seen = req
	if s.err != nil {
		return RemoteResponse{}, s.err
	}
	return s.response, nil
}

func (s *stubClient) Endpoint() string {
	return "http://stub"
}

func TestBackendRequiresClient(t *testing.T) {
	b := New(Config{})
	_, err := b.Execute(context.Background(), runtime.ExecuteRequest{
		Code:    "return 1",
		Gateway: &mockGateway{},
	})
	if !errors.Is(err, ErrClientNotConfigured) {
		t.Fatalf("expected ErrClientNotConfigured, got %v", err)
	}
}

func TestBackendExecuteSuccess(t *testing.T) {
	client := &stubClient{
		response: RemoteResponse{
			Result: &ExecuteResultPayload{
				Value:          map[string]any{"answer": 42},
				Stdout:         "ok",
				DurationMillis: 12,
			},
		},
	}

	b := New(Config{
		Client:          client,
		GatewayEndpoint: "http://gateway",
		GatewayToken:    "token",
		EnableStreaming: true,
	})

	result, err := b.Execute(context.Background(), runtime.ExecuteRequest{
		Code:    "return 42",
		Gateway: &mockGateway{},
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Stdout != "ok" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "ok")
	}
	if val, ok := result.Value.(map[string]any); !ok || val["answer"] != 42 {
		t.Errorf("Value = %#v, want answer=42", result.Value)
	}
	if client.seen.Gateway == nil || client.seen.Gateway.URL != "http://gateway" {
		t.Fatalf("gateway descriptor not set")
	}
}

func TestBackendExecuteErrorResponse(t *testing.T) {
	client := &stubClient{
		response: RemoteResponse{Error: &RemoteError{Code: "unauthorized", Message: "nope"}},
	}
	b := New(Config{Client: client})
	_, err := b.Execute(context.Background(), runtime.ExecuteRequest{
		Code:    "return 0",
		Gateway: &mockGateway{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRemoteExecutionFailed) {
		t.Fatalf("expected ErrRemoteExecutionFailed, got %v", err)
	}
}
