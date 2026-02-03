package remote

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestBackendExecuteSuccess(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer token")
		}
		if sig := r.Header.Get("X-Toolruntime-Signature"); sig == "" {
			t.Error("expected signature header")
		}

		var req remoteRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Request.Code != "return 42" {
			t.Errorf("request code = %q", req.Request.Code)
		}

		resp := remoteResponse{
			Result: &executeResultPayload{
				Value:          map[string]any{"answer": 42},
				Stdout:         "ok",
				DurationMillis: 12,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	b := New(Config{
		Endpoint:  srv.URL,
		AuthToken: "token",
	})

	ctx := context.Background()
	result, err := b.Execute(ctx, runtime.ExecuteRequest{
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
	if val, ok := result.Value.(map[string]any); !ok || val["answer"] != float64(42) {
		t.Errorf("Value = %#v, want answer=42", result.Value)
	}
}

func TestBackendExecuteErrorResponse(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := remoteResponse{
			Error: &remoteError{Code: "unauthorized", Message: "nope"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	b := New(Config{Endpoint: srv.URL})
	_, err := b.Execute(context.Background(), runtime.ExecuteRequest{
		Code:    "return 0",
		Gateway: &mockGateway{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBackendExecuteStreaming(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: stdout\n"))
		_, _ = w.Write([]byte("data: hello\n\n"))
		result := executeResultPayload{
			Value:          "ok",
			Stdout:         "hello",
			DurationMillis: 5,
		}
		data, _ := json.Marshal(result)
		_, _ = w.Write([]byte("event: result\n"))
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
	}))
	defer srv.Close()

	b := New(Config{
		Endpoint:        srv.URL,
		EnableStreaming: true,
	})

	res, err := b.Execute(context.Background(), runtime.ExecuteRequest{
		Code:    "return",
		Gateway: &mockGateway{},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if res.Stdout != "hello" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello")
	}
	if res.Value != "ok" {
		t.Errorf("Value = %#v, want %q", res.Value, "ok")
	}
}
