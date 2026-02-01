// Package main demonstrates complete integration across all toolexec layers.
//
// This example shows how to:
// - Use toolfoundation for tool definitions
// - Use tooldiscovery for registration and search
// - Use toolexec for execution with the unified facade
// - Combine all features in a realistic workflow
//
// Run with: go run ./examples/full
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	// Foundation layer - tool types
	"github.com/jonwraymond/toolfoundation/model"

	// Discovery layer - registration and search
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"

	// Execution layer - unified facade
	"github.com/jonwraymond/toolexec/exec"
	"github.com/jonwraymond/toolexec/runtime"
)

func main() {
	ctx := context.Background()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          toolexec Complete Integration Example             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════════
	// LAYER 1: Foundation (toolfoundation)
	// Define tool types using the standard model
	// ═══════════════════════════════════════════════════════════════════════

	fmt.Println("▶ Layer 1: Foundation - Defining tools")

	toolDefs := []struct {
		tool    model.Tool
		doc     tooldoc.DocEntry
		handler exec.Handler
	}{
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "calculate",
					Description: "Performs arithmetic calculations",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"operation": map[string]any{
								"type": "string",
								"enum": []any{"add", "subtract", "multiply", "divide"},
							},
							"a": map[string]any{"type": "number"},
							"b": map[string]any{"type": "number"},
						},
						"required": []any{"operation", "a", "b"},
					},
				},
				Namespace: "math",
				Tags:      []string{"math", "calculator", "arithmetic"},
			},
			doc: tooldoc.DocEntry{
				Summary: "Basic arithmetic calculator supporting +, -, *, /",
				Notes:   "Division by zero returns an error",
				Examples: []tooldoc.ToolExample{
					{Title: "Addition", Args: map[string]any{"operation": "add", "a": 5, "b": 3}},
				},
			},
			handler: calculateHandler,
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "format_text",
					Description: "Formats text with various transformations",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"text": map[string]any{"type": "string"},
							"style": map[string]any{
								"type": "string",
								"enum": []any{"uppercase", "lowercase", "title", "reverse"},
							},
						},
						"required": []any{"text", "style"},
					},
				},
				Namespace: "text",
				Tags:      []string{"text", "format", "string"},
			},
			doc: tooldoc.DocEntry{
				Summary: "Text formatting with multiple style options",
			},
			handler: formatTextHandler,
		},
		{
			tool: model.Tool{
				Tool: mcp.Tool{
					Name:        "analyze_text",
					Description: "Analyzes text and returns statistics",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"text": map[string]any{"type": "string"},
						},
						"required": []any{"text"},
					},
				},
				Namespace: "text",
				Tags:      []string{"text", "analysis", "statistics"},
			},
			doc: tooldoc.DocEntry{
				Summary: "Returns word count, character count, and other stats",
			},
			handler: analyzeTextHandler,
		},
	}

	fmt.Printf("  Defined %d tools\n\n", len(toolDefs))

	// ═══════════════════════════════════════════════════════════════════════
	// LAYER 2: Discovery (tooldiscovery)
	// Register tools and enable search
	// ═══════════════════════════════════════════════════════════════════════

	fmt.Println("▶ Layer 2: Discovery - Registering tools")

	idx := index.NewInMemoryIndex()
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	handlers := make(map[string]exec.Handler)
	for _, td := range toolDefs {
		handlerName := td.tool.Namespace + "-" + td.tool.Name
		if err := idx.RegisterTool(td.tool, model.NewLocalBackend(handlerName)); err != nil {
			log.Fatalf("Failed to register %s: %v", td.tool.Name, err)
		}
		handlers[handlerName] = td.handler

		// Register documentation
		toolID := td.tool.Namespace + ":" + td.tool.Name
		if err := docs.RegisterDoc(toolID, td.doc); err != nil {
			log.Printf("Warning: failed to register doc for %s: %v", toolID, err)
		}

		fmt.Printf("  ✓ Registered %s:%s\n", td.tool.Namespace, td.tool.Name)
	}
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════════
	// LAYER 3: Execution (toolexec)
	// Create executor and run tools
	// ═══════════════════════════════════════════════════════════════════════

	fmt.Println("▶ Layer 3: Execution - Creating executor")

	executor, err := exec.New(exec.Options{
		Index:           idx,
		Docs:            docs,
		LocalHandlers:   handlers,
		SecurityProfile: runtime.ProfileDev,
		ValidateInput:   false,
		ValidateOutput:  false,
	})
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	fmt.Println("  ✓ Executor created with ProfileDev")
	fmt.Println()

	// ═══════════════════════════════════════════════════════════════════════
	// WORKFLOW: Search → Discover → Execute
	// ═══════════════════════════════════════════════════════════════════════

	fmt.Println("▶ Workflow: Search → Discover → Execute")
	fmt.Println()

	// Search for math tools
	fmt.Println("  Searching for 'calculator'...")
	results, _ := executor.SearchTools(ctx, "calculator", 5)
	fmt.Printf("  Found %d results\n", len(results))
	for _, r := range results {
		fmt.Printf("    - %s: %s\n", r.ID, r.ShortDescription)
	}
	fmt.Println()

	// Execute single tool
	fmt.Println("  Executing math:calculate (10 + 5)...")
	result, err := executor.RunTool(ctx, "math:calculate", map[string]any{
		"operation": "add",
		"a":         float64(10),
		"b":         float64(5),
	})
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
	fmt.Printf("    Result: %v (took %v)\n", result.Value, result.Duration)
	fmt.Println()

	// Execute chain
	fmt.Println("  Executing chain: format → analyze")
	chainResult, steps, err := executor.RunChain(ctx, []exec.Step{
		{
			ToolID: "text:format_text",
			Args: map[string]any{
				"text":  "Hello World",
				"style": "uppercase",
			},
		},
		{
			ToolID: "text:analyze_text",
			Args: map[string]any{
				"text": "HELLO WORLD", // Would use previous result in real chain
			},
		},
	})
	if err != nil {
		log.Fatalf("Chain failed: %v", err)
	}

	fmt.Println("    Step results:")
	for i, step := range steps {
		fmt.Printf("      %d. %s: %v\n", i+1, step.ToolID, step.Value)
	}
	fmt.Printf("    Final: %v (took %v)\n", chainResult.Value, chainResult.Duration)
	fmt.Println()

	// Get documentation
	fmt.Println("  Retrieving documentation for math:calculate...")
	doc, err := executor.GetToolDoc(ctx, "math:calculate", tooldoc.DetailFull)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		fmt.Printf("    Summary: %s\n", doc.Summary)
		if doc.Notes != "" {
			fmt.Printf("    Notes: %s\n", doc.Notes)
		}
	}
	fmt.Println()

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Example Complete                         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
}

// ═══════════════════════════════════════════════════════════════════════════
// Tool Handlers
// ═══════════════════════════════════════════════════════════════════════════

func calculateHandler(ctx context.Context, args map[string]any) (any, error) {
	op, _ := args["operation"].(string)
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)

	switch op {
	case "add":
		return a + b, nil
	case "subtract":
		return a - b, nil
	case "multiply":
		return a * b, nil
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
}

func formatTextHandler(ctx context.Context, args map[string]any) (any, error) {
	text, _ := args["text"].(string)
	style, _ := args["style"].(string)

	switch style {
	case "uppercase":
		return strings.ToUpper(text), nil
	case "lowercase":
		return strings.ToLower(text), nil
	case "title":
		return cases.Title(language.Und).String(text), nil
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	default:
		return text, nil
	}
}

func analyzeTextHandler(ctx context.Context, args map[string]any) (any, error) {
	text, _ := args["text"].(string)

	words := strings.Fields(text)
	return map[string]any{
		"characters": len(text),
		"words":      len(words),
		"lines":      strings.Count(text, "\n") + 1,
	}, nil
}
