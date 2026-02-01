package run

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/toolfoundation/model"
)

func TestStreamEventKindConstants(t *testing.T) {
	tests := []struct {
		kind StreamEventKind
		want string
	}{
		{StreamEventProgress, "progress"},
		{StreamEventChunk, "chunk"},
		{StreamEventDone, "done"},
		{StreamEventError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.kind) != tt.want {
				t.Errorf("StreamEventKind = %q, want %q", tt.kind, tt.want)
			}
		})
	}
}

func TestStreamEvent_JSON(t *testing.T) {
	event := StreamEvent{
		Kind:   StreamEventProgress,
		ToolID: "myns:mytool",
		Data:   map[string]any{"percent": 50},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded StreamEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Kind != StreamEventProgress {
		t.Errorf("Kind = %q, want %q", decoded.Kind, StreamEventProgress)
	}
	if decoded.ToolID != "myns:mytool" {
		t.Errorf("ToolID = %q, want %q", decoded.ToolID, "myns:mytool")
	}
}

func TestStreamEvent_ErrNotSerialized(t *testing.T) {
	event := StreamEvent{
		Kind: StreamEventError,
		Err:  errTest,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Err should not appear in JSON (has json:"-" tag)
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := m["err"]; ok {
		t.Error("Err field should not be serialized")
	}
}

func TestChainStep_Defaults(t *testing.T) {
	step := ChainStep{
		ToolID: "test-tool",
	}

	if step.ToolID != "test-tool" {
		t.Errorf("ToolID = %q, want %q", step.ToolID, "test-tool")
	}
	if step.Args != nil {
		t.Error("Args should default to nil")
	}
	if step.UsePrevious != false {
		t.Error("UsePrevious should default to false")
	}
}

func TestChainStep_JSON(t *testing.T) {
	step := ChainStep{
		ToolID:      "myns:process",
		Args:        map[string]any{"input": "data"},
		UsePrevious: true,
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ChainStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ToolID != "myns:process" {
		t.Errorf("ToolID = %q, want %q", decoded.ToolID, "myns:process")
	}
	if decoded.Args["input"] != "data" {
		t.Errorf("Args[input] = %v, want %q", decoded.Args["input"], "data")
	}
	if decoded.UsePrevious != true {
		t.Error("UsePrevious = false, want true")
	}
}

func TestStepResult_Fields(t *testing.T) {
	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	result := StepResult{
		ToolID:  "mytool",
		Backend: backend,
		Result: RunResult{
			Tool:       tool,
			Backend:    backend,
			Structured: map[string]any{"output": "value"},
		},
		Err: nil,
	}

	if result.ToolID != "mytool" {
		t.Errorf("ToolID = %q, want %q", result.ToolID, "mytool")
	}
	if result.Backend.Kind != model.BackendKindMCP {
		t.Errorf("Backend.Kind = %q, want %q", result.Backend.Kind, model.BackendKindMCP)
	}
	if result.Err != nil {
		t.Errorf("Err = %v, want nil", result.Err)
	}
}

func TestStepResult_ErrNotSerialized(t *testing.T) {
	result := StepResult{
		ToolID: "mytool",
		Err:    errTest,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := m["err"]; ok {
		t.Error("Err field should not be serialized")
	}
}

func TestRunResult_Fields(t *testing.T) {
	tool := testToolWithNamespace("myns", "mytool")
	backend := testMCPBackend("server1")
	mcpResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "hello"},
		},
	}

	result := RunResult{
		Tool:       tool,
		Backend:    backend,
		Structured: "hello",
		MCPResult:  mcpResult,
	}

	if result.Tool.ToolID() != "myns:mytool" {
		t.Errorf("Tool.ToolID() = %q, want %q", result.Tool.ToolID(), "myns:mytool")
	}
	if result.Backend.Kind != model.BackendKindMCP {
		t.Errorf("Backend.Kind = %q, want %q", result.Backend.Kind, model.BackendKindMCP)
	}
	if result.Structured != "hello" {
		t.Errorf("Structured = %v, want %q", result.Structured, "hello")
	}
	if result.MCPResult == nil {
		t.Error("MCPResult should not be nil")
	}
}

func TestRunResult_JSON(t *testing.T) {
	tool := testTool("mytool")
	backend := testLocalBackend("handler1")

	result := RunResult{
		Tool:       tool,
		Backend:    backend,
		Structured: map[string]any{"result": "success"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded RunResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Tool.Name != "mytool" {
		t.Errorf("Tool.Name = %q, want %q", decoded.Tool.Name, "mytool")
	}
	if decoded.Backend.Kind != model.BackendKindLocal {
		t.Errorf("Backend.Kind = %q, want %q", decoded.Backend.Kind, model.BackendKindLocal)
	}
}

func TestRunResult_NilMCPResult(t *testing.T) {
	result := RunResult{
		Tool:       testTool("mytool"),
		Backend:    testLocalBackend("handler1"),
		Structured: "result",
		MCPResult:  nil,
	}

	if result.MCPResult != nil {
		t.Error("MCPResult should be nil for non-MCP backends")
	}
}
