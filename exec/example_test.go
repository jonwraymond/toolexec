package exec_test

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/search"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/exec"
	"github.com/jonwraymond/toolfoundation/model"
)

func ExampleNew() {
	// Create discovery index and documentation store
	idx := index.NewInMemoryIndex(index.IndexOptions{
		Searcher: search.NewBM25Searcher(search.BM25Config{}),
	})
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Create executor with local handler
	executor, err := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]exec.Handler{
			"greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
				name, _ := args["name"].(string)
				return fmt.Sprintf("Hello, %s!", name), nil
			},
		},
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Executor created:", executor != nil)
	// Output:
	// Executor created: true
}

func ExampleExec_RunTool() {
	// Setup
	idx := index.NewInMemoryIndex(index.IndexOptions{
		Searcher: search.NewBM25Searcher(search.BM25Config{}),
	})
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Register a greeting tool
	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "greet",
			Description: "Greets a user",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
		Namespace: "demo",
	}
	_ = idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))

	// Create executor
	executor, _ := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]exec.Handler{
			"greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
				name, _ := args["name"].(string)
				return fmt.Sprintf("Hello, %s!", name), nil
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})

	// Execute the tool
	ctx := context.Background()
	result, err := executor.RunTool(ctx, "demo:greet", map[string]any{"name": "World"})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Result:", result.Value)
	// Output:
	// Result: Hello, World!
}

func ExampleExec_RunChain() {
	// Setup
	idx := index.NewInMemoryIndex(index.IndexOptions{
		Searcher: search.NewBM25Searcher(search.BM25Config{}),
	})
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Register an add tool
	addTool := model.Tool{
		Tool: mcp.Tool{
			Name:        "add",
			Description: "Adds two numbers",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{"type": "number"},
					"b": map[string]any{"type": "number"},
				},
			},
		},
		Namespace: "math",
	}
	_ = idx.RegisterTool(addTool, model.NewLocalBackend("add-handler"))

	// Create executor
	executor, _ := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
		LocalHandlers: map[string]exec.Handler{
			"add-handler": func(ctx context.Context, args map[string]any) (any, error) {
				a, _ := args["a"].(float64)
				b, _ := args["b"].(float64)
				return a + b, nil
			},
		},
		ValidateInput:  false,
		ValidateOutput: false,
	})

	// Execute a chain of operations
	ctx := context.Background()
	result, steps, err := executor.RunChain(ctx, []exec.Step{
		{ToolID: "math:add", Args: map[string]any{"a": float64(5), "b": float64(3)}},
		{ToolID: "math:add", Args: map[string]any{"a": float64(10), "b": float64(2)}},
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Steps completed:", len(steps))
	fmt.Println("Final result:", result.Value)
	// Output:
	// Steps completed: 2
	// Final result: 12
}

func ExampleExec_SearchTools() {
	// Setup - use default lexical searcher for simplicity
	idx := index.NewInMemoryIndex()
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// Register some tools (InputSchema is required)
	tools := []model.Tool{
		{
			Tool: mcp.Tool{
				Name:        "greet",
				Description: "Greets a user with a friendly message",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: "demo",
		},
		{
			Tool: mcp.Tool{
				Name:        "farewell",
				Description: "Says goodbye to a user",
				InputSchema: map[string]any{"type": "object"},
			},
			Namespace: "demo",
		},
	}
	for _, t := range tools {
		_ = idx.RegisterTool(t, model.NewLocalBackend("handler"))
	}

	// Create executor
	executor, _ := exec.New(exec.Options{
		Index: idx,
		Docs:  docs,
	})

	// Search for greeting-related tools
	ctx := context.Background()
	results, err := executor.SearchTools(ctx, "greet", 5)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Found tools:", len(results))
	if len(results) > 0 {
		fmt.Println("Top result:", results[0].ID)
	}
	// Output:
	// Found tools: 1
	// Top result: demo:greet
}
