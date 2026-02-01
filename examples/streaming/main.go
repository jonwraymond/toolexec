// Package main demonstrates streaming tool execution.
//
// This example shows how to:
// - Use the run.Runner directly for streaming
// - Handle streaming events from tool execution
// - Process partial results as they arrive
//
// Run with: go run ./examples/streaming
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	ctx := context.Background()

	// 1. Setup infrastructure
	idx := index.NewInMemoryIndex()

	// 2. Register a streaming-capable tool
	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "generate_report",
			Description: "Generates a report with streaming output",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sections": map[string]any{
						"type":        "integer",
						"description": "Number of sections to generate",
					},
				},
			},
		},
		Namespace: "reports",
		Tags:      []string{"streaming", "generation"},
	}

	if err := idx.RegisterTool(tool, model.NewLocalBackend("report-handler")); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// 3. Create a local registry with streaming simulation
	localReg := &simpleRegistry{
		handlers: map[string]run.LocalHandler{
			"report-handler": func(ctx context.Context, args map[string]any) (any, error) {
				sections := 3
				if s, ok := args["sections"].(float64); ok {
					sections = int(s)
				}

				var report string
				for i := 1; i <= sections; i++ {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					default:
						// Simulate work
						time.Sleep(100 * time.Millisecond)
						report += fmt.Sprintf("Section %d: Content for section %d\n", i, i)
					}
				}
				return report, nil
			},
		},
	}

	// 4. Create runner directly (exec facade doesn't expose streaming yet)
	runner := run.NewRunner(
		run.WithIndex(idx),
		run.WithLocalRegistry(localReg),
		run.WithValidation(false, false),
	)

	// 5. Execute with streaming (note: local backend returns complete result)
	fmt.Println("=== Streaming Tool Execution ===")
	fmt.Println("Executing report generation...")
	fmt.Println()

	start := time.Now()
	stream, err := runner.RunStream(ctx, "reports:generate_report", map[string]any{
		"sections": float64(5),
	})

	if err != nil {
		// Local backends may not support streaming - fall back to regular execution
		if err == run.ErrStreamNotSupported {
			fmt.Println("Streaming not supported, using regular execution...")
			result, err := runner.Run(ctx, "reports:generate_report", map[string]any{
				"sections": float64(5),
			})
			if err != nil {
				log.Fatalf("Execution failed: %v", err)
			}
			fmt.Printf("Result:\n%v\n", result.Structured)
			fmt.Printf("Duration: %v\n", time.Since(start))
			return
		}
		log.Fatalf("Stream failed: %v", err)
	}

	// 6. Process streaming events
	for event := range stream {
		switch event.Kind {
		case run.StreamEventProgress:
			fmt.Printf("[PROGRESS] %v\n", event.Data)
		case run.StreamEventChunk:
			fmt.Printf("[CHUNK] %v\n", event.Data)
		case run.StreamEventDone:
			fmt.Printf("[DONE]\n%v\n", event.Data)
		case run.StreamEventError:
			fmt.Printf("[ERROR] %v\n", event.Err)
		}
	}

	fmt.Printf("\nTotal duration: %v\n", time.Since(start))
}

// simpleRegistry implements run.LocalRegistry
type simpleRegistry struct {
	handlers map[string]run.LocalHandler
}

func (r *simpleRegistry) Get(name string) (run.LocalHandler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}
