package backend_test

import (
	"context"
	"fmt"

	"github.com/jonwraymond/toolexec/backend"
	"github.com/jonwraymond/toolexec/backend/local"
)

func ExampleRegistry() {
	// Create a registry
	reg := backend.NewRegistry()

	// Create and register a local backend
	localBackend := local.New("demo")
	localBackend.RegisterHandler("greet", local.ToolDef{
		Name:        "greet",
		Description: "Greets a user",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			name, _ := args["name"].(string)
			return fmt.Sprintf("Hello, %s!", name), nil
		},
	})

	_ = reg.Register(localBackend)

	// List registered backends
	backends := reg.List()
	fmt.Printf("Registered backends: %d\n", len(backends))

	// Get a specific backend
	b, ok := reg.Get("demo")
	fmt.Printf("Found 'demo': %v\n", ok)
	fmt.Printf("Backend kind: %s\n", b.Kind())
	// Output:
	// Registered backends: 1
	// Found 'demo': true
	// Backend kind: local
}

func ExampleAggregator() {
	// Create registry and backends
	reg := backend.NewRegistry()

	backend1 := local.New("math")
	backend1.RegisterHandler("add", local.ToolDef{
		Name:        "add",
		Description: "Adds two numbers",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return a + b, nil
		},
	})

	backend2 := local.New("text")
	backend2.RegisterHandler("upper", local.ToolDef{
		Name:        "upper",
		Description: "Converts to uppercase",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return "HELLO", nil
		},
	})

	_ = reg.Register(backend1)
	_ = reg.Register(backend2)

	// Create aggregator
	agg := backend.NewAggregator(reg)

	// List tools from all backends
	ctx := context.Background()
	tools, _ := agg.ListAllTools(ctx)
	fmt.Printf("Total tools: %d\n", len(tools))

	// Execute through aggregator (using backend:tool format)
	result, _ := agg.Execute(ctx, "math:add", map[string]any{"a": float64(5), "b": float64(3)})
	fmt.Printf("5 + 3 = %v\n", result)
	// Output:
	// Total tools: 2
	// 5 + 3 = 8
}

func ExampleBackend_lifecycle() {
	// Create a local backend
	b := local.New("example")

	// Register a tool
	b.RegisterHandler("echo", local.ToolDef{
		Name:        "echo",
		Description: "Echoes input",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return args["message"], nil
		},
	})

	ctx := context.Background()

	// Start the backend
	if err := b.Start(ctx); err != nil {
		fmt.Printf("Start failed: %v\n", err)
		return
	}

	// Check if enabled
	fmt.Printf("Enabled: %v\n", b.Enabled())

	// List available tools
	tools, _ := b.ListTools(ctx)
	fmt.Printf("Tools: %d\n", len(tools))

	// Execute a tool
	result, _ := b.Execute(ctx, "echo", map[string]any{"message": "hello"})
	fmt.Printf("Result: %v\n", result)

	// Stop the backend
	_ = b.Stop()
	// Output:
	// Enabled: true
	// Tools: 1
	// Result: hello
}

func ExampleInfo() {
	b := local.New("my-backend")

	info := backend.Info{
		Kind:        b.Kind(),
		Name:        b.Name(),
		Enabled:     b.Enabled(),
		Description: "A demo backend",
		Version:     "1.0.0",
	}

	fmt.Printf("Kind: %s\n", info.Kind)
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Enabled: %v\n", info.Enabled)
	// Output:
	// Kind: local
	// Name: my-backend
	// Enabled: true
}

// Verify interface compliance
var _ backend.Backend = (*local.Backend)(nil)
