# Examples

toolexec ships with runnable examples that cover execution, chaining,
streaming, discovery integration, and runtime isolation.

## Basic execution

```bash
go run ./examples/basic
```

Shows:
- Local backend registration
- Input validation and execution
- Structured results

## Tool chaining

```bash
go run ./examples/chain
```

Shows:
- Sequential chains
- `UsePrevious` argument injection
- Step results and errors

## Discovery + execution

```bash
go run ./examples/discovery
```

Shows:
- Index + docs store integration
- Search + execute via `exec` facade

## Streaming

```bash
go run ./examples/streaming
```

Shows:
- Streaming events from execution
- Progress + chunk envelopes

## Runtime isolation

```bash
go run ./examples/runtime
```

Shows:
- Security profiles
- Runtime selection (unsafe / docker / wasm)

## Full integration

```bash
go run ./examples/full
```

Shows:
- End-to-end setup with discovery + exec + runtime
