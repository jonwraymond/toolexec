// Package exec provides a unified facade for tool execution in the toolexec ecosystem.
//
// The exec package simplifies tool execution by combining discovery, execution, and
// result handling into a single, cohesive API. It integrates with tooldiscovery for
// tool registration and search, and with run for the underlying execution pipeline.
//
// # Overview
//
// Instead of working with multiple packages directly, users can create an [Exec] instance
// that handles the complete workflow:
//
//   - Tool registration via tooldiscovery's index
//   - Local handler management
//   - Tool search and discovery
//   - Single tool and chain execution
//   - Documentation retrieval
//
// # Basic Usage
//
//	// Create discovery index and documentation store
//	idx := index.NewInMemoryIndex(index.IndexOptions{
//	    Searcher: search.NewBM25Searcher(search.DefaultConfig()),
//	})
//	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})
//
//	// Register a tool
//	tool := model.Tool{
//	    Tool: mcp.Tool{Name: "greet", Description: "Greets a user"},
//	    Namespace: "demo",
//	}
//	idx.RegisterTool(tool, model.NewLocalBackend("greet-handler"))
//
//	// Create executor with local handler
//	executor, err := exec.New(exec.Options{
//	    Index: idx,
//	    Docs:  docs,
//	    LocalHandlers: map[string]exec.Handler{
//	        "greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
//	            return fmt.Sprintf("Hello, %s!", args["name"]), nil
//	        },
//	    },
//	})
//
//	// Execute the tool
//	result, err := executor.RunTool(ctx, "demo:greet", map[string]any{"name": "World"})
//
// # Search and Execute
//
// The package supports a search-then-execute workflow:
//
//	results, _ := executor.SearchTools(ctx, "greeting tools", 5)
//	if len(results) > 0 {
//	    result, _ := executor.RunTool(ctx, results[0].ID, args)
//	}
//
// # Chain Execution
//
// Execute multiple tools in sequence, optionally passing results between steps:
//
//	result, steps, err := executor.RunChain(ctx, []exec.Step{
//	    {ToolID: "ns:tool1", Args: map[string]any{"x": 1}},
//	    {ToolID: "ns:tool2", UsePrevious: true}, // receives tool1's result
//	})
//
// # Integration
//
// The exec package integrates with:
//
//   - [github.com/jonwraymond/tooldiscovery/index] for tool registration and lookup
//   - [github.com/jonwraymond/tooldiscovery/tooldoc] for tool documentation
//   - [github.com/jonwraymond/toolexec/run] for the underlying execution pipeline
//   - [github.com/jonwraymond/toolfoundation/model] for tool and backend types
package exec
