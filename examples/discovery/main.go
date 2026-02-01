// Package main demonstrates the search → execute workflow.
//
// This example shows how to:
// - Register multiple tools with rich metadata
// - Search for tools by query
// - Execute discovered tools dynamically
// - Use progressive disclosure for tool documentation
//
// Run with: go run ./examples/discovery
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	// 2. Register a variety of tools
	toolDefs := []struct {
		name      string
		namespace string
		desc      string
		tags      []string
		handler   string
	}{
		{"create_file", "files", "Creates a new file with content", []string{"filesystem", "write"}, "create-file"},
		{"read_file", "files", "Reads content from a file", []string{"filesystem", "read"}, "read-file"},
		{"list_files", "files", "Lists files in a directory", []string{"filesystem", "list"}, "list-files"},
		{"search_code", "code", "Searches for patterns in code", []string{"search", "grep"}, "search-code"},
		{"format_code", "code", "Formats code according to style", []string{"format", "lint"}, "format-code"},
		{"send_email", "comms", "Sends an email message", []string{"email", "notification"}, "send-email"},
		{"post_slack", "comms", "Posts a message to Slack", []string{"chat", "notification"}, "post-slack"},
	}

	handlers := make(map[string]exec.Handler)
	for _, td := range toolDefs {
		tool := model.Tool{
			Tool: mcp.Tool{
				Name:        td.name,
				Description: td.desc,
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: td.namespace,
			Tags:      td.tags,
		}
		if err := idx.RegisterTool(tool, model.NewLocalBackend(td.handler)); err != nil {
			log.Fatalf("Failed to register %s: %v", td.name, err)
		}

		// Create a demo handler that echoes the tool info
		handlerName := td.handler
		toolName := td.name
		handlers[handlerName] = func(ctx context.Context, args map[string]any) (any, error) {
			return fmt.Sprintf("Executed %s with args: %v", toolName, args), nil
		}
	}

	// 3. Create executor
	executor, err := exec.New(exec.Options{
		Index:          idx,
		Docs:           docs,
		LocalHandlers:  handlers,
		ValidateInput:  false,
		ValidateOutput: false,
	})
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// 4. Demonstrate search → execute workflow
	queries := []string{"file operations", "search", "notification"}

	for _, query := range queries {
		fmt.Printf("=== Searching for: %q ===\n", query)

		results, err := executor.SearchTools(ctx, query, 3)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		if len(results) == 0 {
			fmt.Println("No tools found")
			continue
		}

		fmt.Printf("Found %d tools:\n", len(results))
		for i, r := range results {
			fmt.Printf("  %d. %s - %s\n", i+1, r.ID, r.ShortDescription)
		}

		// Execute the top result
		topTool := results[0]
		fmt.Printf("\nExecuting top result: %s\n", topTool.ID)

		result, err := executor.RunTool(ctx, topTool.ID, map[string]any{
			"example": "argument",
		})
		if err != nil {
			log.Printf("Execution failed: %v", err)
			continue
		}

		fmt.Printf("Result: %v\n", result.Value)
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println()
	}

	// 5. Demonstrate documentation retrieval
	fmt.Println("=== Tool Documentation ===")
	doc, err := executor.GetToolDoc(ctx, "files:create_file", tooldoc.DetailSummary)
	if err != nil {
		log.Printf("Failed to get doc: %v", err)
	} else {
		fmt.Printf("Summary: %s\n", doc.Summary)
	}
}
