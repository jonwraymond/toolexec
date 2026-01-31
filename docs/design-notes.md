# toolexec Design Notes

## Overview

toolexec provides the execution layer for the ApertureStack tool framework.
It handles tool execution, code orchestration, and runtime isolation.

## run Package

### Design Decisions

1. **Execution Pipeline**: Every tool call follows a strict pipeline:
   - Validate tool ID format
   - Validate input against schema
   - Resolve tool definition from index
   - Select and invoke backend
   - Normalize result
   - Validate output against schema

2. **Backend Abstraction**: The runner doesn't care how tools are executed.
   It delegates to the backend registry which supports local, provider, and
   MCP server backends.

3. **Result Normalization**: All backends return results in a consistent
   `RunResult` format with output, error, duration, and metadata.

### Error Handling

- Input validation errors include the failing field and constraint
- Backend errors are wrapped with context
- Output validation errors are warnings (logged but not fatal)

## code Package

### Design Decisions

1. **DSL for Orchestration**: Provides a simple DSL for chaining tool calls
   with variable binding and conditional logic.

2. **Runner Integration**: Delegates actual tool execution to the `run`
   package, ensuring consistent validation and error handling.

3. **Build Tag Gating**: Code execution requires the `toolruntime` build tag
   to enable sandbox support.

## runtime Package

### Design Decisions

1. **Runtime Interface**: Abstracts sandbox implementations behind a common
   interface supporting Execute and Cleanup operations.

2. **Security Profiles**:
   - `dev`: Unsafe subprocess execution (local development)
   - `standard`: Docker-based isolation (production)
   - `wasm`: WASM sandbox via wazero (edge/browser)

3. **Resource Limits**: Configurable CPU, memory, and timeout limits for
   sandboxed execution.

### Supported Runtimes

| Runtime | Isolation | Performance | Use Case |
|---------|-----------|-------------|----------|
| Local | None | Fast | Trusted tools |
| Docker | Container | Medium | Production |
| WASM | Sandbox | Varies | Edge/browser |

## backend Package

### Design Decisions

1. **Backend Registry**: Central registry for all backend implementations,
   enabling runtime backend selection.

2. **Backend Kinds**:
   - `local`: In-process Go function
   - `provider`: External tool provider via HTTP/gRPC
   - `mcp`: Remote MCP server via JSON-RPC

3. **Lazy Resolution**: Backends are resolved at execution time, allowing
   dynamic registration and configuration.

## Dependencies

- `github.com/jonwraymond/toolfoundation/model` - Tool definitions
- `github.com/jonwraymond/tooldiscovery/index` - Tool resolution
- `github.com/tetratelabs/wazero` - WASM runtime (optional)

## Links

- [index](index.md)
- [user journey](user-journey.md)
