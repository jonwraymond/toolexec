# toolexec User Journey

## Overview

This guide walks through using toolexec to execute tools, from simple
single-tool calls to complex orchestrated workflows.

## 1. Installation

```bash
go get github.com/jonwraymond/toolexec@latest
```

## 2. Quick Start with the Unified Facade (Recommended)

The `exec` package provides a unified facade combining discovery and execution:

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
        "calculator-add": func(ctx context.Context, args map[string]any) (any, error) {
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

// Get tool documentation
doc, _ := executor.GetToolDoc(ctx, "math:add", tooldoc.DetailFull)
```

## 3. Execute a Single Tool

```go
result, err := executor.RunTool(ctx, "github:create_issue", map[string]any{
    "owner": "jonwraymond",
    "repo":  "toolexec",
    "title": "Bug report",
    "body":  "Description here",
})

if err != nil {
    log.Fatalf("Execution failed: %v", err)
}

fmt.Printf("Created issue: %v\n", result.Value)
fmt.Printf("Duration: %v\n", result.Duration)
```

## 4. Chain Tool Calls

Use `UsePrevious` to pass results between steps:

```go
result, steps, err := executor.RunChain(ctx, []exec.Step{
    {ToolID: "github:create_issue", Args: map[string]any{
        "title": "Bug report",
    }},
    {ToolID: "github:add_labels", Args: map[string]any{
        "labels": []string{"bug"},
    }, UsePrevious: true}, // Injects previous result as "previous" arg
})

fmt.Printf("Final result: %v\n", result.Value)
for i, step := range steps {
    fmt.Printf("Step %d: %s → %v\n", i+1, step.ToolID, step.Value)
}
```

## 5. Register Local Tools

Register tools using the backend/local package:

```go
import "github.com/jonwraymond/toolexec/backend/local"

backend := local.New("calculator")
backend.RegisterHandler("add", local.ToolDef{
    Name:        "add",
    Description: "Adds two numbers",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "a": map[string]any{"type": "number"},
            "b": map[string]any{"type": "number"},
        },
    },
    Handler: func(ctx context.Context, args map[string]any) (any, error) {
        a, b := args["a"].(float64), args["b"].(float64)
        return a + b, nil
    },
})
```

## 6. Using the Run Package Directly (Advanced)

For more control, use the `run` package directly:

```go
import "github.com/jonwraymond/toolexec/run"

runner := run.NewRunner(
    run.WithIndex(idx),
    run.WithLocalRegistry(localReg),
    run.WithValidation(true, true),
)

result, err := runner.Run(ctx, "ns:tool", args)
```

## 7. Code Orchestration (Sandboxed Execution)

The `code` package enables executing user-provided code that can call tools:

```go
import "github.com/jonwraymond/toolexec/code"

// Create code executor with limits
codeExec := code.NewExecutor(code.Config{
    Index:        idx,
    Docs:         docs,
    Run:          runner,
    MaxToolCalls: 50,
    Timeout:      30 * time.Second,
})

result, err := codeExec.Execute(ctx, code.ExecuteParams{
    Language: "go",
    Code: `
        // Access tools via the tools interface
        result, _ := tools.RunTool(ctx, "math:add", map[string]any{"a": 1, "b": 2})
        return result.Structured
    `,
})
```

## 8. Runtime Isolation (Security Profiles)

toolexec supports three security profiles for different isolation levels:

| Profile | Isolation | Use Case |
|---------|-----------|----------|
| `ProfileDev` | None | Development/testing |
| `ProfileStandard` | Container | Production |
| `ProfileHardened` | VM/gVisor | Untrusted code |

```go
import (
    "github.com/jonwraymond/toolexec/runtime"
    "github.com/jonwraymond/toolexec/runtime/backend/unsafe"
    "github.com/jonwraymond/toolexec/runtime/gateway/direct"
)

// Gateway exposes tool discovery + execution to sandboxed code
gateway := direct.New(direct.Config{
    Index:  idx,
    Docs:   docs,
    Runner: runner,
})

// Runtime with security profile selection
rt := runtime.NewDefaultRuntime(runtime.RuntimeConfig{
    Backends: map[runtime.SecurityProfile]runtime.Backend{
        runtime.ProfileDev: unsafe.New(unsafe.Config{RequireOptIn: true}),
        // runtime.ProfileStandard: docker.New(dockerConfig),
        // runtime.ProfileHardened: gvisor.New(gvisorConfig),
    },
    DefaultProfile: runtime.ProfileDev,
})

// Execute code in the runtime
result, err := rt.Execute(ctx, runtime.ExecuteRequest{
    Language: "go",
    Code:     `__out = 1 + 1`,
    Profile:  runtime.ProfileDev,
    Gateway:  gateway,
    Limits: runtime.Limits{
        MaxToolCalls:   100,
        MemoryBytes:    512 * 1024 * 1024, // 512MB
        CPUQuotaMillis: 60000,             // 60s
    },
})
```

For container isolation, use `runtime/backend/docker` with `ProfileStandard`.
For maximum isolation, use `runtime/backend/gvisor` with `ProfileHardened`.

## Execution Flow

```
Client              exec.Exec            run.Runner           Backend
  |                     |                     |                    |
  |-- RunTool(id) ------|                     |                    |
  |                     |-- Run(id, args) ----|                    |
  |                     |                     |-- Validate --------|
  |                     |                     |-- Resolve ---------|
  |                     |                     |-- Execute ---------|
  |                     |                     |<-- Result ---------|
  |                     |<-- RunResult -------|                    |
  |<-- Result ----------|                     |                    |
```

## Examples

See the [examples](examples.md) page for runnable examples (source lives in
`toolexec/examples/`):

- `basic/` - Simple tool execution
- `chain/` - Sequential tool chaining
- `discovery/` - Search and execute workflow
- `streaming/` - Streaming execution events
- `runtime/` - Security profile configuration
- `full/` - Complete integration example

## Next Steps

- Add observability with [toolops/observe](https://github.com/jonwraymond/toolops)
- Expose via MCP with [metatools-mcp](https://github.com/jonwraymond/metatools-mcp)
