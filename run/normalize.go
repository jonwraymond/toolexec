package run

import (
	"encoding/json"

	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// normalize creates a RunResult from the dispatch result.
// For MCP backends, it extracts structured content.
// For provider/local backends, it uses the structured result directly.
func (r *DefaultRunner) normalize(tool model.Tool, backend model.ToolBackend, dr *dispatchResult) RunResult {
	result := RunResult{
		Tool:    tool,
		Backend: backend,
	}

	if dr.mcpResult != nil {
		result.MCPResult = dr.mcpResult
		result.Structured = extractStructured(dr.mcpResult)
	} else {
		result.Structured = dr.structured
	}

	return result
}

// extractStructured extracts a structured value from an MCP result.
// Prefers StructuredContent, falls back to best-effort parsing of Content.
func extractStructured(mcpResult *mcp.CallToolResult) any {
	if mcpResult == nil {
		return nil
	}

	// Prefer StructuredContent
	if mcpResult.StructuredContent != nil {
		return mcpResult.StructuredContent
	}

	// Best-effort: try to extract from Content
	return bestEffortStructured(mcpResult.Content)
}

// bestEffortStructured attempts to extract a structured value from MCP content.
// If there's a single text content that's valid JSON, returns the parsed value.
// Otherwise returns the raw text or nil.
func bestEffortStructured(content []mcp.Content) any {
	if len(content) == 0 {
		return nil
	}

	// If there's exactly one text content, try to parse as JSON
	if len(content) == 1 {
		text := extractTextFromContent(content[0])
		if text != "" {
			// Try to parse as JSON
			var parsed any
			if err := json.Unmarshal([]byte(text), &parsed); err == nil {
				return parsed
			}
			// Not JSON, return as string
			return text
		}
	}

	// Multiple content items: collect all text
	var texts []string
	for _, c := range content {
		if text := extractTextFromContent(c); text != "" {
			texts = append(texts, text)
		}
	}

	if len(texts) == 1 {
		return texts[0]
	}
	if len(texts) > 1 {
		return texts
	}

	return nil
}

// extractTextFromContent extracts text from an MCP Content item.
func extractTextFromContent(c mcp.Content) string {
	// MCP Content is an interface; we need to type-switch
	// Only pointer types implement mcp.Content (MarshalJSON has pointer receiver)
	switch v := c.(type) {
	case *mcp.TextContent:
		return v.Text
	default:
		// Try to marshal and check for text field (fallback)
		data, err := json.Marshal(c)
		if err != nil {
			return ""
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return ""
		}
		if text, ok := m["text"].(string); ok {
			return text
		}
		return ""
	}
}
