# toolexec

Execution layer providing tool running, code orchestration, and runtime
isolation for the ApertureStack tool framework.

## Packages

| Package | Purpose |
|---------|---------|
| `run` | Core tool execution engine with validation |
| `code` | Code-based tool orchestration |
| `runtime` | Sandbox and runtime isolation |
| `backend` | Backend registry and resolution |

## Installation

```bash
go get github.com/jonwraymond/toolexec@latest
```

## Quick Start

### Execute a Tool

```go
import (
  "github.com/jonwraymond/toolexec/run"
  "github.com/jonwraymond/tooldiscovery/index"
)

// Create runner with index
runner := run.NewRunner(run.WithIndex(idx))

// Execute a tool
result, err := runner.Run(ctx, "github:create_issue", map[string]any{
  "owner": "jonwraymond",
  "repo":  "toolexec",
  "title": "New issue",
})

if err != nil {
  log.Fatal(err)
}

fmt.Printf("Result: %v\n", result.Output)
```

### Register a Local Backend

```go
import "github.com/jonwraymond/toolexec/backend"

registry := backend.NewRegistry()

// Register a local handler
registry.Register("calculator:add", backend.Local(func(ctx context.Context, args any) (any, error) {
  m := args.(map[string]any)
  return m["a"].(float64) + m["b"].(float64), nil
}))
```

### Code Orchestration (Optional)

```go
import "github.com/jonwraymond/toolexec/code"

executor := code.NewExecutor(runner)

result, err := executor.Execute(ctx, `
  issue := run("github:create_issue", {title: "Bug fix"})
  run("github:add_labels", {issue: issue.number, labels: ["bug"]})
`)
```

## Key Features

- **Schema validation**: Input and output are validated against tool schemas
- **Backend abstraction**: Execute local, provider, or MCP server backends
- **Tool chaining**: Chain multiple tool calls with result passing
- **Runtime isolation**: Sandbox untrusted code with Docker or WASM

## Links

- [design notes](design-notes.md)
- [user journey](user-journey.md)
- [ai-tools-stack documentation](https://jonwraymond.github.io/ai-tools-stack/)
