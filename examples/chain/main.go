// Package main demonstrates sequential tool chaining with the exec facade.
//
// This example shows how to:
// - Execute multiple tools in sequence
// - Pass results between steps using UsePrevious
// - Access individual step results
//
// Run with: go run ./examples/chain
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

	// 1. Setup infrastructure
	idx := index.NewInMemoryIndex()
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// 2. Register math tools
	mathSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a":        map[string]any{"type": "number"},
			"b":        map[string]any{"type": "number"},
			"previous": map[string]any{"type": "number", "description": "Result from previous step"},
		},
	}

	tools := []struct {
		name    string
		desc    string
		handler string
	}{
		{"add", "Adds two numbers", "add-handler"},
		{"multiply", "Multiplies two numbers", "multiply-handler"},
		{"subtract", "Subtracts b from a", "subtract-handler"},
	}

	for _, t := range tools {
		tool := model.Tool{
			Tool: mcp.Tool{
				Name:        t.name,
				Description: t.desc,
				InputSchema: mathSchema,
			},
			Namespace: "math",
			Tags:      []string{"math", "arithmetic"},
		}
		if err := idx.RegisterTool(tool, model.NewLocalBackend(t.handler)); err != nil {
			log.Fatalf("Failed to register %s: %v", t.name, err)
		}
	}

	// 3. Create executor with handlers
	executor, err := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]exec.Handler{
			"add-handler": func(ctx context.Context, args map[string]any) (any, error) {
				a := toFloat(args["a"])
				b := toFloat(args["b"])
				if prev, ok := args["previous"]; ok {
					a = toFloat(prev)
				}
				return a + b, nil
			},
			"multiply-handler": func(ctx context.Context, args map[string]any) (any, error) {
				a := toFloat(args["a"])
				b := toFloat(args["b"])
				if prev, ok := args["previous"]; ok {
					a = toFloat(prev)
				}
				return a * b, nil
			},
			"subtract-handler": func(ctx context.Context, args map[string]any) (any, error) {
				a := toFloat(args["a"])
				b := toFloat(args["b"])
				if prev, ok := args["previous"]; ok {
					a = toFloat(prev)
				}
				return a - b, nil
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// 4. Execute a chain: (5 + 3) * 2 - 1 = 15
	fmt.Println("Computing: (5 + 3) * 2 - 1")
	fmt.Println()

	result, steps, err := executor.RunChain(ctx, []exec.Step{
		{
			ToolID: "math:add",
			Args:   map[string]any{"a": float64(5), "b": float64(3)},
		},
		{
			ToolID:      "math:multiply",
			Args:        map[string]any{"b": float64(2)},
			UsePrevious: true, // Uses result of add as 'a'
		},
		{
			ToolID:      "math:subtract",
			Args:        map[string]any{"b": float64(1)},
			UsePrevious: true, // Uses result of multiply as 'a'
		},
	})
	if err != nil {
		log.Fatalf("Chain execution failed: %v", err)
	}

	// 5. Show step-by-step results
	for i, step := range steps {
		status := "✓"
		if !step.OK() {
			status = "✗"
		}
		fmt.Printf("Step %d [%s] %s: %v\n", i+1, status, step.ToolID, step.Value)
	}

	fmt.Println()
	fmt.Printf("Final result: %v\n", result.Value)
	fmt.Printf("Total duration: %v\n", result.Duration)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
