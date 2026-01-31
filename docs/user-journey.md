# toolexec User Journey

## Overview

This guide walks through using toolexec to execute tools, from simple
single-tool calls to complex orchestrated workflows.

## 1. Installation

```bash
go get github.com/jonwraymond/toolexec@latest
```

## 2. Set Up the Runner

```go
import (
  "github.com/jonwraymond/toolexec/run"
  "github.com/jonwraymond/toolexec/backend"
  "github.com/jonwraymond/tooldiscovery/index"
)

// Create index and register tools
idx := index.NewInMemoryIndex()
// ... register tools ...

// Create backend registry
backends := backend.NewRegistry()

// Create runner
runner := run.NewRunner(
  run.WithIndex(idx),
  run.WithBackends(backends),
)
```

## 3. Execute a Single Tool

```go
result, err := runner.Run(ctx, "github:create_issue", map[string]any{
  "owner": "jonwraymond",
  "repo":  "toolexec",
  "title": "Bug report",
  "body":  "Description here",
})

if err != nil {
  log.Fatalf("Execution failed: %v", err)
}

fmt.Printf("Created issue #%v\n", result.Output.(map[string]any)["number"])
fmt.Printf("Duration: %v\n", result.Duration)
```

## 4. Register Local Tool Backends

```go
// Register a local calculator
backends.Register("calculator:add", backend.Local(func(ctx context.Context, args any) (any, error) {
  m := args.(map[string]any)
  a := m["a"].(float64)
  b := m["b"].(float64)
  return map[string]any{"result": a + b}, nil
}))

// Now you can execute it
result, _ := runner.Run(ctx, "calculator:add", map[string]any{"a": 5, "b": 3})
// result.Output = {"result": 8}
```

## 5. Chain Tool Calls

```go
// Execute a chain of tools
results, err := runner.RunChain(ctx, []run.ChainStep{
  {ToolID: "github:create_issue", Args: map[string]any{
    "title": "Bug report",
  }},
  {ToolID: "github:add_labels", Args: map[string]any{
    "issue":  "{{prev.number}}", // Reference previous result
    "labels": []string{"bug"},
  }},
})
```

## 6. Code Orchestration (Advanced)

```go
import "github.com/jonwraymond/toolexec/code"

executor := code.NewExecutor(runner)

result, err := executor.Execute(ctx, `
  // Create an issue
  issue := run("github:create_issue", {
    title: "Bug report",
    body: "Found a problem"
  })

  // Add labels based on title
  if (issue.title.contains("bug")) {
    run("github:add_labels", {
      issue: issue.number,
      labels: ["bug", "needs-triage"]
    })
  }

  // Return the issue
  issue
`)
```

## 7. Runtime Isolation (Sandbox)

```go
import "github.com/jonwraymond/toolexec/runtime"

// Create Docker runtime for untrusted code
rt := runtime.NewDockerRuntime(runtime.DockerConfig{
  Image:   "toolruntime-sandbox:latest",
  Timeout: 30 * time.Second,
  Memory:  "256m",
  CPUs:    "0.5",
})

// Execute with isolation
result, err := rt.Execute(ctx, tool, args)

// Clean up
defer rt.Cleanup()
```

## Execution Flow

```
Client              run.Runner           backend.Registry        Backend
  |                     |                       |                    |
  |-- Run(id, args) ----|                       |                    |
  |                     |-- Validate input -----|                    |
  |                     |-- GetTool(id) --------|                    |
  |                     |<-- Tool def ----------|                    |
  |                     |-- Resolve backend ----|                    |
  |                     |                       |-- Get backend -----|
  |                     |                       |<-- Backend --------|
  |                     |-- Execute ------------|                    |
  |                     |                       |                    |
  |                     |<-- Raw result --------|--------------------|
  |                     |-- Validate output ----|                    |
  |<-- RunResult -------|                       |                    |
```

## Next Steps

- Add observability with [toolops/observe](https://github.com/jonwraymond/toolops)
- Expose via MCP with [metatools-mcp](https://github.com/jonwraymond/metatools-mcp)
