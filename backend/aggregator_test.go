package backend

import (
	"context"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAggregator_ListAllTools(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{
		kind:    "local",
		name:    "local1",
		enabled: true,
		tools: []model.Tool{
			{Tool: mcp.Tool{Name: "tool_a"}, Namespace: "local1"},
			{Tool: mcp.Tool{Name: "tool_b"}, Namespace: "local1"},
		},
	})

	_ = registry.Register(&mockBackend{
		kind:    "mcp",
		name:    "github",
		enabled: true,
		tools: []model.Tool{
			{Tool: mcp.Tool{Name: "create_issue"}, Namespace: "github"},
		},
	})

	_ = registry.Register(&mockBackend{
		kind:    "local",
		name:    "disabled",
		enabled: false,
		tools: []model.Tool{
			{Tool: mcp.Tool{Name: "should_not_appear"}, Namespace: "disabled"},
		},
	})

	agg := NewAggregator(registry)

	tools, err := agg.ListAllTools(context.Background())
	if err != nil {
		t.Fatalf("ListAllTools() error = %v", err)
	}

	if len(tools) != 3 {
		t.Errorf("ListAllTools() returned %d tools, want 3", len(tools))
	}
}

func TestAggregator_Execute(t *testing.T) {
	registry := NewRegistry()

	_ = registry.Register(&mockBackend{
		kind:    "local",
		name:    "local",
		enabled: true,
		execFn: func(_ context.Context, tool string, args map[string]any) (any, error) {
			if tool == "echo" {
				return args["msg"], nil
			}
			return nil, ErrToolNotFound
		},
	})

	agg := NewAggregator(registry)

	result, err := agg.Execute(context.Background(), "local:echo", map[string]any{
		"msg": "hello",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "hello" {
		t.Errorf("Execute() = %v, want %v", result, "hello")
	}
}

func TestAggregator_ExecuteNotFound(t *testing.T) {
	registry := NewRegistry()
	agg := NewAggregator(registry)

	_, err := agg.Execute(context.Background(), "nonexistent:tool", nil)
	if err == nil {
		t.Error("Execute() should fail for nonexistent backend")
	}
}

func TestAggregator_ParseToolID(t *testing.T) {
	tests := []struct {
		id          string
		wantBackend string
		wantTool    string
		wantErr     bool
	}{
		{"local:echo", "local", "echo", false},
		{"github:create_issue", "github", "create_issue", false},
		{"my-backend:my_tool", "my-backend", "my_tool", false},
		{"no_namespace", "", "no_namespace", false},
		{"", "", "", true},
		{"bad:format:tool", "", "", true},
	}

	for _, tt := range tests {
		backend, tool, err := ParseToolID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseToolID(%q) error = %v, wantErr = %v", tt.id, err, tt.wantErr)
			continue
		}
		if backend != tt.wantBackend {
			t.Errorf("ParseToolID(%q) backend = %q, want %q", tt.id, backend, tt.wantBackend)
		}
		if tool != tt.wantTool {
			t.Errorf("ParseToolID(%q) tool = %q, want %q", tt.id, tool, tt.wantTool)
		}
	}
}
