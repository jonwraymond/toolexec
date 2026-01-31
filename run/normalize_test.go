package run

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNormalize_MCP_StructuredContent(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	structured := map[string]any{"result": "success"}
	dr := &dispatchResult{
		mcpResult: testMCPResultStructured(structured),
	}

	result := runner.normalize(tool, backend, dr)

	if result.MCPResult == nil {
		t.Error("MCPResult should not be nil")
	}

	m, ok := result.Structured.(map[string]any)
	if !ok {
		t.Fatalf("Structured type = %T, want map[string]any", result.Structured)
	}
	if m["result"] != "success" {
		t.Errorf("Structured[result] = %v, want %q", m["result"], "success")
	}
}

func TestNormalize_MCP_BestEffort_SingleTextJSON(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	// MCP result with JSON text content (no StructuredContent)
	dr := &dispatchResult{
		mcpResult: testMCPResultJSON(map[string]any{"parsed": "json"}),
	}

	result := runner.normalize(tool, backend, dr)

	m, ok := result.Structured.(map[string]any)
	if !ok {
		t.Fatalf("Structured type = %T, want map[string]any (best-effort JSON parse)", result.Structured)
	}
	if m["parsed"] != "json" {
		t.Errorf("Structured[parsed] = %v, want %q", m["parsed"], "json")
	}
}

func TestNormalize_MCP_BestEffort_NonJSON(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	// MCP result with plain text (not JSON)
	dr := &dispatchResult{
		mcpResult: testMCPResult("plain text result"),
	}

	result := runner.normalize(tool, backend, dr)

	// Should return the text as-is since it's not JSON
	text, ok := result.Structured.(string)
	if !ok {
		t.Fatalf("Structured type = %T, want string", result.Structured)
	}
	if text != "plain text result" {
		t.Errorf("Structured = %q, want %q", text, "plain text result")
	}
}

func TestNormalize_MCP_BestEffort_MultipleContent(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	// MCP result with multiple text content items
	dr := &dispatchResult{
		mcpResult: &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "line 1"},
				&mcp.TextContent{Text: "line 2"},
			},
		},
	}

	result := runner.normalize(tool, backend, dr)

	// Multiple content should return slice of texts
	texts, ok := result.Structured.([]string)
	if !ok {
		t.Fatalf("Structured type = %T, want []string", result.Structured)
	}
	if len(texts) != 2 {
		t.Errorf("len(Structured) = %d, want 2", len(texts))
	}
}

func TestNormalize_DirectStructured(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testLocalBackend("handler1")

	// Direct structured result (from provider/local)
	dr := &dispatchResult{
		structured: map[string]any{"direct": "result"},
	}

	result := runner.normalize(tool, backend, dr)

	if result.MCPResult != nil {
		t.Error("MCPResult should be nil for non-MCP backends")
	}

	m, ok := result.Structured.(map[string]any)
	if !ok {
		t.Fatalf("Structured type = %T, want map[string]any", result.Structured)
	}
	if m["direct"] != "result" {
		t.Errorf("Structured[direct] = %v, want %q", m["direct"], "result")
	}
}

func TestNormalize_EmptyMCPResult(t *testing.T) {
	runner := NewRunner()

	tool := testTool("mytool")
	backend := testMCPBackend("server1")

	dr := &dispatchResult{
		mcpResult: &mcp.CallToolResult{},
	}

	result := runner.normalize(tool, backend, dr)

	if result.MCPResult == nil {
		t.Error("MCPResult should not be nil")
	}
	if result.Structured != nil {
		t.Errorf("Structured = %v, want nil", result.Structured)
	}
}
