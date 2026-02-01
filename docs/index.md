# toolexec

Execution layer providing tool running, code orchestration, and runtime
isolation for the ApertureStack tool framework.

## Packages

| Package | Purpose |
|---------|---------|
| `exec` | **Unified facade** - combines discovery + execution into single API |
| `run` | Core tool execution engine with validation |
| `code` | Code-based tool orchestration for sandboxed execution |
| `runtime` | Sandbox and runtime isolation with security profiles |
| `backend` | Backend registry and resolution |

## Installation

```bash
go get github.com/jonwraymond/toolexec@latest
```

## Quick Start

### Using the Unified Facade (Recommended)

```go
import (
    "github.com/jonwraymond/toolexec/exec"
    "github.com/jonwraymond/tooldiscovery/index"
    "github.com/jonwraymond/tooldiscovery/tooldoc"
)

// Setup discovery infrastructure
idx := index.NewInMemoryIndex()
docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

// Create executor with local handlers
executor, err := exec.New(exec.Options{
    Index: idx,
    Docs:  docs,
    LocalHandlers: map[string]exec.Handler{
        "math-add": func(ctx context.Context, args map[string]any) (any, error) {
            a, b := args["a"].(float64), args["b"].(float64)
            return a + b, nil
        },
    },
})

// Execute a tool
result, err := executor.RunTool(ctx, "math:add", map[string]any{"a": 5, "b": 3})
fmt.Println(result.Value) // 8

// Search for tools
results, _ := executor.SearchTools(ctx, "calculator", 10)

// Chain tools together
chainResult, steps, _ := executor.RunChain(ctx, []exec.Step{
    {ToolID: "text:format", Args: map[string]any{"text": "hello", "style": "upper"}},
    {ToolID: "text:analyze", Args: map[string]any{}, UsePrevious: true},
})
```

### Using the Run Package Directly

```go
import "github.com/jonwraymond/toolexec/run"

// Create runner with index
runner := run.NewRunner(run.WithIndex(idx))

// Execute a tool
result, err := runner.Run(ctx, "github:create_issue", map[string]any{
    "owner": "jonwraymond",
    "repo":  "toolexec",
    "title": "New issue",
})
```

### Register a Local Backend

```go
import "github.com/jonwraymond/toolexec/backend/local"

backend := local.New("math")
backend.RegisterHandler("add", local.ToolDef{
    Name:        "add",
    Description: "Adds two numbers",
    Handler: func(ctx context.Context, args map[string]any) (any, error) {
        a, b := args["a"].(float64), args["b"].(float64)
        return a + b, nil
    },
})
```

## Key Features

- **Unified Facade**: Single API for discovery, execution, and documentation
- **Schema Validation**: Input and output validated against tool schemas
- **Backend Abstraction**: Execute local, provider, or MCP server backends
- **Tool Chaining**: Chain multiple tool calls with `UsePrevious` result passing
- **Security Profiles**: Dev, Standard, and Hardened isolation levels
- **Runtime Isolation**: Sandbox untrusted code with Docker, gVisor, or WASM

## Examples

See [examples](examples.md) for runnable walkthroughs.

## Links

- [Architecture](architecture.md)
- [Schemas and contracts](schemas.md)
- [Design notes](design-notes.md)
- [User journey](user-journey.md)
- [ai-tools-stack documentation](https://jonwraymond.github.io/ai-tools-stack/)
