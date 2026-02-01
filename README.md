# toolexec

[![CI](https://github.com/jonwraymond/toolexec/actions/workflows/ci.yml/badge.svg)](https://github.com/jonwraymond/toolexec/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jonwraymond/toolexec.svg)](https://pkg.go.dev/github.com/jonwraymond/toolexec)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonwraymond/toolexec)](https://goreportcard.com/report/github.com/jonwraymond/toolexec)

Execution layer providing tool running, code orchestration, and runtime
isolation for the ApertureStack tool framework.

## Installation

```bash
go get github.com/jonwraymond/toolexec@latest
```

## Packages

| Package | Description |
|---------|-------------|
| [`exec`](https://pkg.go.dev/github.com/jonwraymond/toolexec/exec) | Unified facade combining discovery + execution |
| [`run`](https://pkg.go.dev/github.com/jonwraymond/toolexec/run) | Core execution pipeline with validation and chaining |
| [`code`](https://pkg.go.dev/github.com/jonwraymond/toolexec/code) | Code-based orchestration with tool access |
| [`runtime`](https://pkg.go.dev/github.com/jonwraymond/toolexec/runtime) | Sandbox runtimes and security profiles |
| [`backend`](https://pkg.go.dev/github.com/jonwraymond/toolexec/backend) | Backend registry and resolution |

## Quick Start

```go
import (
    "github.com/jonwraymond/toolexec/exec"
    "github.com/jonwraymond/tooldiscovery/index"
    "github.com/jonwraymond/tooldiscovery/tooldoc"
)

idx := index.NewInMemoryIndex()
docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

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

result, err := executor.RunTool(ctx, "math:add", map[string]any{"a": 5, "b": 3})
fmt.Println(result.Value) // 8
```

## Features

- **Unified Facade**: Single API for discovery, execution, and documentation
- **Schema Validation**: Input/output validated against tool schemas
- **Backend Abstraction**: Local, provider, or MCP server execution
- **Tool Chaining**: Sequential chains with explicit data passing
- **Runtime Isolation**: Docker, gVisor, and WASM runtimes

## Documentation

- **MkDocs site**: https://jonwraymond.github.io/toolexec/
- [Schemas and contracts](./docs/schemas.md)
- [Architecture](./docs/architecture.md)
- [Design notes](./docs/design-notes.md)
- [User journey](./docs/user-journey.md)
- [Examples](./docs/examples.md)
- [Contributing](./CONTRIBUTING.md)

## Examples

```bash
go run ./examples/basic
go run ./examples/chain
go run ./examples/discovery
go run ./examples/streaming
go run ./examples/runtime
go run ./examples/full
```

## License

MIT License - see [LICENSE](./LICENSE)
