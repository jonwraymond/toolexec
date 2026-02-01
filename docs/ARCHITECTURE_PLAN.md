# toolexec Architecture Improvement Plan

> **✅ COMPLETED:** This plan was fully implemented on 2026-02-01.
> All phases completed: exec/ facade, examples, example tests, coverage improvements, documentation.
> See CHANGELOG.md for details.

## Executive Summary

Architectural review and improvement plan for the toolexec submodule, focusing on better integration with toolfoundation and tooldiscovery, comprehensive examples, and coverage improvements.

**Current State:** 18 packages, 62-94% coverage
**Dependencies:** toolfoundation v0.2.0, tooldiscovery v0.2.1

---

## 1. Current Architecture Overview

### Package Dependency Graph

```
┌─────────────────────────────────────────────────────────────────────┐
│                         EXTERNAL DEPENDENCIES                        │
├─────────────────────────────────────────────────────────────────────┤
│  toolfoundation                    tooldiscovery                     │
│  ├── model.Tool                    ├── index.Index                   │
│  ├── model.ToolBackend             ├── search.BM25Searcher           │
│  └── model.SchemaValidator         └── tooldoc.Store                 │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                           toolexec                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────────┐      ┌──────────────┐      ┌──────────────┐      │
│   │   backend/   │      │     run/     │      │    code/     │      │
│   │              │      │              │      │              │      │
│   │ • Backend    │◄─────│ • Runner     │◄─────│ • Executor   │      │
│   │ • Registry   │      │ • dispatch   │      │ • Engine     │      │
│   │ • Aggregator │      │ • resolve    │      │ • Tools      │      │
│   │ • local/     │      │ • normalize  │      │ • Config     │      │
│   └──────────────┘      └──────────────┘      └──────────────┘      │
│          │                     │                     │               │
│          │                     │                     ▼               │
│          │                     │            ┌──────────────┐         │
│          │                     │            │   runtime/   │         │
│          │                     │            │              │         │
│          │                     └───────────►│ • Runtime    │         │
│          │                                  │ • Backend    │         │
│          │                                  │ • Gateway    │         │
│          │                                  └──────────────┘         │
│          │                                         │                 │
│          │                                         ▼                 │
│          │              ┌────────────────────────────────────────┐   │
│          │              │         runtime/backend/               │   │
│          │              ├────────────────────────────────────────┤   │
│          │              │ unsafe │ docker │ wasm │ gvisor │ ...  │   │
│          │              └────────────────────────────────────────┘   │
│          │                                         │                 │
│          │              ┌────────────────────────────────────────┐   │
│          │              │         runtime/gateway/               │   │
│          │              ├────────────────────────────────────────┤   │
│          │              │         direct    │    proxy           │   │
│          │              └────────────────────────────────────────┘   │
│          │                                                           │
│          └──────────────────────────────────────────────────────────►│
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Execution Flow

```
User Code Request
       │
       ▼
┌─────────────────┐
│ code.Executor   │ ── Orchestrates execution with limits
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ code.Engine     │ ── Pluggable language runtime (Go, Python, etc.)
└────────┬────────┘
         │
         ▼
┌─────────────────────────────┐
│ runtime/toolcodeengine      │ ── Adapter: code.Engine → runtime.Backend
└────────┬────────────────────┘
         │
         ▼
┌─────────────────┐
│ runtime.Runtime │ ── Routes by security profile
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌───────┐ ┌───────┐
│unsafe │ │docker │ ... (10 backend implementations)
└───┬───┘ └───┬───┘
    │         │
    └────┬────┘
         │
         ▼
┌─────────────────┐
│ runtime.Gateway │ ── Tool access from sandbox
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ run.Runner      │ ── Tool execution pipeline
└────────┬────────┘
         │
    ┌────┴────┬────────┐
    ▼         ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐
│  MCP  │ │Provider│ │ Local │
└───────┘ └───────┘ └───────┘
```

---

## 2. Identified Gaps

### 2.1 Coverage Gaps

| Package | Current | Target | Gap Analysis |
|---------|---------|--------|--------------|
| `backend/` | 67.0% | 90%+ | Registry lifecycle, concurrent access |
| `backend/local/` | 62.9% | 90%+ | Handler edge cases, error paths |
| `code/` | 72.6% | 90%+ | Limit enforcement, timeout handling |
| `runtime/gateway/proxy/` | 68.4% | 85%+ | Connection failures, codec errors |
| `run/` | ~75% | 90%+ | Chain execution, streaming |

### 2.2 Documentation Gaps

| Gap | Severity | Description |
|-----|----------|-------------|
| Empty examples/ | **High** | No runnable examples showing integration |
| Missing architecture doc | **High** | No high-level overview of package relationships |
| Interface contracts | **Medium** | Some contracts lack error semantics |
| Migration guide | **Low** | No guide from direct usage to facades |

### 2.3 Integration Gaps

| Gap | Description |
|-----|-------------|
| No unified facade | Users must understand 4+ packages to use toolexec |
| Bridge patterns unclear | How tooldiscovery → toolexec → runtime flows |
| Backend selection logic | No documented strategy for profile → backend mapping |

---

## 3. Improvement Plan

### Phase 1: Unified Facade Package (NEW)

Create a `toolexec/exec` package that provides a simplified entry point.

```go
// exec/exec.go - Unified facade for tool execution

package exec

import (
    "context"
    "github.com/jonwraymond/tooldiscovery/index"
    "github.com/jonwraymond/tooldiscovery/tooldoc"
    "github.com/jonwraymond/toolexec/run"
    "github.com/jonwraymond/toolexec/code"
    "github.com/jonwraymond/toolexec/runtime"
)

// Exec is the unified facade for tool execution.
// It combines discovery, execution, and runtime management.
type Exec struct {
    index   index.Index
    docs    tooldoc.Store
    runner  run.Runner
    runtime runtime.Runtime
    code    code.Executor
}

// Options configures an Exec instance.
type Options struct {
    // Index provides tool discovery. Required.
    Index index.Index

    // Docs provides tool documentation. Required.
    Docs tooldoc.Store

    // SecurityProfile determines the runtime backend.
    // Default: ProfileDev
    SecurityProfile runtime.SecurityProfile

    // EnableCodeExecution enables the code execution subsystem.
    // Default: false (tool execution only)
    EnableCodeExecution bool

    // MaxToolCalls limits tool calls in code execution.
    // Default: 100
    MaxToolCalls int

    // DefaultLanguage for code execution.
    // Default: "go"
    DefaultLanguage string
}

// New creates a new Exec instance.
func New(opts Options) (*Exec, error)

// RunTool executes a single tool by ID.
func (e *Exec) RunTool(ctx context.Context, toolID string, args map[string]any) (Result, error)

// RunChain executes a sequence of tools.
func (e *Exec) RunChain(ctx context.Context, steps []Step) (Result, []StepResult, error)

// ExecuteCode runs code with tool access.
func (e *Exec) ExecuteCode(ctx context.Context, params CodeParams) (CodeResult, error)

// SearchTools finds tools matching a query.
func (e *Exec) SearchTools(ctx context.Context, query string, limit int) ([]ToolSummary, error)

// GetToolDoc retrieves tool documentation.
func (e *Exec) GetToolDoc(ctx context.Context, toolID string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error)
```

**Files to create:**
- `exec/exec.go` - Main facade
- `exec/result.go` - Unified result types
- `exec/options.go` - Configuration and validation
- `exec/exec_test.go` - Comprehensive tests
- `exec/example_test.go` - pkg.go.dev examples
- `exec/doc.go` - Package documentation

### Phase 2: Comprehensive Examples

Create runnable examples showing the full integration:

```
examples/
├── basic/
│   └── main.go           # Simple tool execution
├── chain/
│   └── main.go           # Sequential tool chaining
├── code/
│   └── main.go           # Code execution with tool access
├── discovery/
│   └── main.go           # Search → Execute workflow
├── streaming/
│   └── main.go           # Streaming tool execution
├── runtime/
│   └── main.go           # Custom runtime configuration
└── full/
    └── main.go           # Complete integration example
```

#### examples/basic/main.go

```go
// Demonstrates basic tool execution with toolexec.
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jonwraymond/tooldiscovery/index"
    "github.com/jonwraymond/tooldiscovery/search"
    "github.com/jonwraymond/tooldiscovery/tooldoc"
    "github.com/jonwraymond/toolexec/exec"
    "github.com/jonwraymond/toolfoundation/model"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    ctx := context.Background()

    // 1. Setup tool discovery (from tooldiscovery)
    idx := index.NewInMemoryIndex(index.IndexOptions{
        Searcher: search.NewBM25Searcher(search.BM25Config{
            NameBoost: 3,
            TagsBoost: 2,
        }),
    })
    docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

    // 2. Register a sample tool
    tool := model.Tool{
        Tool: mcp.Tool{
            Name:        "greet",
            Description: "Greets a user by name",
            InputSchema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "name": map[string]any{"type": "string"},
                },
                "required": []any{"name"},
            },
        },
        Namespace: "demo",
    }

    // Register with local handler
    if err := idx.RegisterTool(tool, model.NewLocalBackend("greet-handler")); err != nil {
        log.Fatal(err)
    }

    // Add documentation
    docs.RegisterDoc("demo:greet", tooldoc.DocEntry{
        Summary: "Greets a user with a friendly message",
        Notes:   "Returns a greeting string",
        Examples: []tooldoc.ToolExample{
            {Title: "Basic greeting", Args: map[string]any{"name": "World"}},
        },
    })

    // 3. Create executor with unified facade
    executor, err := exec.New(exec.Options{
        Index: idx,
        Docs:  docs,
        LocalHandlers: map[string]exec.Handler{
            "greet-handler": func(ctx context.Context, args map[string]any) (any, error) {
                name := args["name"].(string)
                return fmt.Sprintf("Hello, %s!", name), nil
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // 4. Execute the tool
    result, err := executor.RunTool(ctx, "demo:greet", map[string]any{"name": "World"})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %v\n", result.Value)
    // Output: Result: Hello, World!
}
```

#### examples/discovery/main.go

```go
// Demonstrates search → execute workflow combining tooldiscovery and toolexec.
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jonwraymond/tooldiscovery/discovery"
    "github.com/jonwraymond/toolexec/exec"
)

func main() {
    ctx := context.Background()

    // 1. Create discovery facade (from tooldiscovery)
    disc, err := discovery.New(discovery.Options{})
    if err != nil {
        log.Fatal(err)
    }

    // 2. Register tools (simplified)
    registerDemoTools(disc)

    // 3. Create executor using discovery's index
    executor, err := exec.New(exec.Options{
        Index: disc.Index(),
        Docs:  disc.DocStore(),
    })
    if err != nil {
        log.Fatal(err)
    }

    // 4. Search for tools
    results, err := disc.Search(ctx, "file operations", 5)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d tools:\n", len(results))
    for _, r := range results {
        fmt.Printf("  - %s (score: %.2f)\n", r.Summary.ID, r.Score)
    }

    // 5. Execute the top result
    if len(results) > 0 {
        topTool := results[0].Summary.ID
        result, err := executor.RunTool(ctx, topTool, map[string]any{
            "path": "/tmp/example.txt",
        })
        if err != nil {
            log.Printf("Execution failed: %v", err)
            return
        }
        fmt.Printf("Executed %s: %v\n", topTool, result.Value)
    }
}
```

#### examples/full/main.go

```go
// Complete integration example showing all layers working together.
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    // Foundation layer
    "github.com/jonwraymond/toolfoundation/model"

    // Discovery layer
    "github.com/jonwraymond/tooldiscovery/discovery"
    "github.com/jonwraymond/tooldiscovery/tooldoc"

    // Execution layer
    "github.com/jonwraymond/toolexec/exec"
    "github.com/jonwraymond/toolexec/runtime"
)

func main() {
    ctx := context.Background()

    // ═══════════════════════════════════════════════════════════════
    // LAYER 1: Foundation (toolfoundation)
    // Define tool types and schemas
    // ═══════════════════════════════════════════════════════════════

    tools := []struct {
        tool    model.Tool
        backend model.ToolBackend
        doc     tooldoc.DocEntry
        handler exec.Handler
    }{
        {
            tool: model.Tool{
                Tool: mcp.Tool{
                    Name:        "calculate",
                    Description: "Performs basic arithmetic",
                    InputSchema: calculateSchema(),
                },
                Namespace: "math",
                Tags:      []string{"math", "calculator"},
            },
            backend: model.NewLocalBackend("calc-handler"),
            doc: tooldoc.DocEntry{
                Summary: "Basic arithmetic operations",
                Notes:   "Supports add, subtract, multiply, divide",
            },
            handler: calculateHandler,
        },
        // ... more tools
    }

    // ═══════════════════════════════════════════════════════════════
    // LAYER 2: Discovery (tooldiscovery)
    // Register and search for tools
    // ═══════════════════════════════════════════════════════════════

    disc, _ := discovery.New(discovery.Options{})

    for _, t := range tools {
        disc.RegisterTool(t.tool, t.backend, &t.doc)
    }

    // Search demonstration
    results, _ := disc.Search(ctx, "arithmetic", 10)
    fmt.Printf("Discovery found %d tools for 'arithmetic'\n", len(results))

    // ═══════════════════════════════════════════════════════════════
    // LAYER 3: Execution (toolexec)
    // Execute tools with proper runtime management
    // ═══════════════════════════════════════════════════════════════

    handlers := make(map[string]exec.Handler)
    for _, t := range tools {
        handlers[t.backend.Local.Name] = t.handler
    }

    executor, _ := exec.New(exec.Options{
        Index:           disc.Index(),
        Docs:            disc.DocStore(),
        LocalHandlers:   handlers,
        SecurityProfile: runtime.ProfileDev,

        // Code execution settings
        EnableCodeExecution: true,
        MaxToolCalls:        50,
        DefaultLanguage:     "go",
        DefaultTimeout:      30 * time.Second,
    })

    // Single tool execution
    result, _ := executor.RunTool(ctx, "math:calculate", map[string]any{
        "operation": "add",
        "a":         10,
        "b":         20,
    })
    fmt.Printf("10 + 20 = %v\n", result.Value)

    // Chain execution
    chainResult, steps, _ := executor.RunChain(ctx, []exec.Step{
        {ToolID: "math:calculate", Args: map[string]any{"operation": "add", "a": 5, "b": 3}},
        {ToolID: "math:calculate", Args: map[string]any{"operation": "multiply", "a": 0, "b": 2}, UsePrevious: true},
    })
    fmt.Printf("Chain result: %v (steps: %d)\n", chainResult.Value, len(steps))

    // ═══════════════════════════════════════════════════════════════
    // LAYER 4: Code Execution (with tool access)
    // ═══════════════════════════════════════════════════════════════

    codeResult, _ := executor.ExecuteCode(ctx, exec.CodeParams{
        Language: "go",
        Code: `
            // This code runs in a sandbox with tool access
            result, _ := tools.Run("math:calculate", map[string]any{
                "operation": "add",
                "a": 100,
                "b": 200,
            })
            return result
        `,
        Timeout: 10 * time.Second,
    })
    fmt.Printf("Code execution result: %v\n", codeResult.Value)
    fmt.Printf("Tool calls made: %d\n", len(codeResult.ToolCalls))
}
```

### Phase 3: Example Tests (pkg.go.dev)

Create example tests for each core package:

**Files to create:**
- `run/example_test.go` - Runner examples
- `code/example_test.go` - Executor examples
- `backend/example_test.go` - Backend/registry examples
- `runtime/example_test.go` - Runtime examples
- `exec/example_test.go` - Unified facade examples

### Phase 4: Coverage Improvements

#### backend/ (67% → 90%+)

Add tests for:
- Registry concurrent access
- Backend lifecycle (Start/Stop)
- Aggregator with multiple backends
- Error paths (backend unavailable, tool not found)

#### backend/local/ (62.9% → 90%+)

Add tests for:
- Handler registration/unregistration
- Concurrent handler execution
- Panic recovery in handlers
- Context cancellation

#### code/ (72.6% → 90%+)

Add tests for:
- MaxToolCalls enforcement
- MaxChainSteps enforcement
- Timeout handling
- Engine error propagation

#### runtime/gateway/proxy/ (68.4% → 85%+)

Add tests for:
- Connection failures
- Codec errors
- Timeout scenarios
- Large payload handling

### Phase 5: Documentation

**Files to create:**
- `docs/architecture.md` - Package hierarchy and data flow
- `docs/integration.md` - How to integrate with tooldiscovery
- `docs/security-profiles.md` - Runtime security configuration
- `docs/error-handling.md` - Error types and handling patterns
- `docs/migration.md` - Upgrading from direct package usage

---

## 4. Integration Patterns

### Pattern 1: Discovery → Execution

```go
// Search for tools, then execute
disc, _ := discovery.New(discovery.Options{})
exec, _ := exec.New(exec.Options{Index: disc.Index(), Docs: disc.DocStore()})

results, _ := disc.Search(ctx, "query", 10)
for _, r := range results {
    result, _ := exec.RunTool(ctx, r.Summary.ID, args)
}
```

### Pattern 2: Code with Tool Access

```go
// Execute code that can call tools
exec, _ := exec.New(exec.Options{
    Index: idx,
    Docs:  docs,
    EnableCodeExecution: true,
})

result, _ := exec.ExecuteCode(ctx, exec.CodeParams{
    Code: `tools.Run("ns:tool", args)`,
})
```

### Pattern 3: Chain Execution

```go
// Execute tools in sequence
result, steps, _ := exec.RunChain(ctx, []exec.Step{
    {ToolID: "ns:tool1", Args: args1},
    {ToolID: "ns:tool2", UsePrevious: true}, // Uses tool1's result
})
```

### Pattern 4: Custom Runtime

```go
// Configure specific backend for security
exec, _ := exec.New(exec.Options{
    Index:           idx,
    Docs:            docs,
    SecurityProfile: runtime.ProfileHardened,
    RuntimeBackends: map[runtime.SecurityProfile]runtime.Backend{
        runtime.ProfileHardened: gvisor.NewBackend(gvisor.Config{}),
    },
})
```

---

## 5. File Summary

| Phase | File | Action | Est. Lines |
|-------|------|--------|------------|
| 1 | `exec/exec.go` | Create | ~200 |
| 1 | `exec/result.go` | Create | ~80 |
| 1 | `exec/options.go` | Create | ~100 |
| 1 | `exec/exec_test.go` | Create | ~400 |
| 1 | `exec/example_test.go` | Create | ~150 |
| 1 | `exec/doc.go` | Create | ~50 |
| 2 | `examples/basic/main.go` | Create | ~80 |
| 2 | `examples/chain/main.go` | Create | ~100 |
| 2 | `examples/code/main.go` | Create | ~120 |
| 2 | `examples/discovery/main.go` | Create | ~100 |
| 2 | `examples/streaming/main.go` | Create | ~100 |
| 2 | `examples/runtime/main.go` | Create | ~120 |
| 2 | `examples/full/main.go` | Create | ~200 |
| 3 | `run/example_test.go` | Create | ~100 |
| 3 | `code/example_test.go` | Create | ~100 |
| 3 | `backend/example_test.go` | Create | ~80 |
| 3 | `runtime/example_test.go` | Create | ~100 |
| 4 | `backend/backend_test.go` | Expand | ~150 |
| 4 | `backend/local/local_test.go` | Expand | ~150 |
| 4 | `code/executor_test.go` | Expand | ~200 |
| 4 | `runtime/gateway/proxy/*_test.go` | Expand | ~150 |
| 5 | `docs/architecture.md` | Create | ~200 |
| 5 | `docs/integration.md` | Create | ~150 |
| 5 | `docs/security-profiles.md` | Create | ~100 |
| 5 | `docs/error-handling.md` | Create | ~150 |

---

## 6. Verification

### Unit Tests
```bash
go test ./... -v -race -cover
```

### Coverage Target
```bash
go test ./... -coverprofile=cover.out
go tool cover -func=cover.out | grep total
# Target: 85%+ overall
```

### Example Execution
```bash
go run examples/basic/main.go
go run examples/full/main.go
```

---

## 7. Implementation Order

1. **Phase 1: exec/ facade** - Unified entry point (enables Phase 2)
2. **Phase 2: examples/** - Runnable integration examples
3. **Phase 3: example_test.go** - pkg.go.dev documentation
4. **Phase 4: coverage** - Test gap closure
5. **Phase 5: docs/** - Written documentation

Each phase can be committed independently and provides incremental value.
