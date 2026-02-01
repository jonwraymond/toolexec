// Package main demonstrates basic tool execution with the exec facade.
//
// This example shows how to:
// - Create a tool index and documentation store
// - Register a tool with a local handler
// - Execute the tool using the unified exec facade
//
// Run with: go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/exec"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	ctx := context.Background()

	// 1. Create tool discovery infrastructure
	idx := index.NewInMemoryIndex()
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// 2. Define and register a tool
	greetTool := model.Tool{
		Tool: mcp.Tool{
			Name:        "greet",
			Description: "Greets a user by name",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "The name to greet",
					},
				},
				"required": []any{"name"},
			},
		},
		Namespace: "demo",
		Tags:      []string{"greeting", "demo"},
	}

	if err := idx.RegisterTool(greetTool, model.NewLocalBackend("greet-handler")); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// 3. Create the executor with local handler
	executor, err := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]exec.Handler{
			"greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
				name, ok := args["name"].(string)
				if !ok {
					return nil, fmt.Errorf("name must be a string")
				}
				return fmt.Sprintf("Hello, %s! Welcome to toolexec.", name), nil
			},
		},
		ValidateInput:  false, // Disable for demo simplicity
		ValidateOutput: false,
	})
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// 4. Execute the tool
	result, err := executor.RunTool(ctx, "demo:greet", map[string]any{
		"name": "World",
	})
	if err != nil {
		log.Fatalf("Tool execution failed: %v", err)
	}

	fmt.Printf("Result: %v\n", result.Value)
	fmt.Printf("Duration: %v\n", result.Duration)
}
