# toolexec Design Notes

## Overview

toolexec provides the execution layer for the ApertureStack tool framework.
It handles tool execution, code orchestration, and runtime isolation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          exec                                │
│              (Unified Facade - Single Entry Point)          │
│  SearchTools, RunTool, RunChain, GetToolDoc                 │
└─────────────────────────┬───────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          v               v               v
    ┌──────────┐   ┌──────────────┐  ┌──────────┐
    │  index   │   │     run      │  │ tooldoc  │
    │(discover)│   │  (execute)   │  │  (docs)  │
    └──────────┘   └──────┬───────┘  └──────────┘
                          │
                          v
                   ┌──────────────┐
                   │   backend    │
                   │  (registry)  │
                   └──────────────┘
                          │
          ┌───────────────┼───────────────┐
          v               v               v
    ┌──────────┐   ┌──────────────┐  ┌──────────┐
    │  local   │   │     mcp      │  │ provider │
    │ handlers │   │   servers    │  │   APIs   │
    └──────────┘   └──────────────┘  └──────────┘
```

## exec Package (Unified Facade)

### Design Decisions

1. **Single Entry Point**: The `exec.Exec` type provides a unified API combining
   discovery (search, describe) with execution (run, chain). Users don't need
   to understand multiple packages for basic operations.

2. **Options Pattern**: Configuration via `exec.Options` allows:
   - Custom index and doc store
   - Local handler registration via map
   - MCP and provider executors
   - Input/output validation toggles
   - Security profile selection

3. **Result Types**: Consistent `Result` and `StepResult` types wrap lower-level
   `run.RunResult` with additional context (toolID, duration, error).

4. **Handler Function**: Simple `func(ctx, args) (any, error)` signature for
   local tool handlers, avoiding the need to understand backend interfaces.

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

3. **Runtime Integration**: Code execution can be isolated by wiring a
   `runtime.Runtime` via the `runtime/toolcodeengine` adapter.

## runtime Package

### Design Decisions

1. **Runtime Interface**: Abstracts sandbox implementations behind a common
   interface supporting Execute and Cleanup operations.

2. **Security Profiles**:
   - `ProfileDev`: Unsafe host execution (development only)
   - `ProfileStandard`: Container-based isolation
   - `ProfileHardened`: Maximum isolation (seccomp + VM/VM-like backends)

3. **Resource Limits**: Configurable CPU, memory, and timeout limits for
   sandboxed execution.
4. **Gateway Requirement**: Every execution request must include a
   `ToolGateway` to broker tool discovery/execution for sandboxed code.

### Supported Runtimes

| Runtime | Isolation | Performance | Use Case |
|---------|-----------|-------------|----------|
| Unsafe host | None | Fast | Trusted/dev |
| Docker | Container | Medium | Production |
| WASM | Sandbox | Varies | Edge/browser |

### Runtime Backend Matrix

Readiness tiers:
- **prod**: production-ready
- **beta**: usable, still evolving
- **stub**: placeholder or incomplete

| BackendKind | Readiness | Isolation | Requirements | Notes |
|-------------|-----------|-----------|--------------|-------|
| `BackendUnsafeHost` | prod | None | Go toolchain (subprocess mode) | Dev-only, explicit opt-in supported |
| `BackendDocker` | prod | Container | Docker daemon + ContainerRunner | Standard isolation |
| `BackendContainerd` | beta | Container | containerd client | Infrastructure-native |
| `BackendKubernetes` | beta | Pod/Job | kubeconfig/client | Cluster execution |
| `BackendGVisor` | beta | Sandbox | gVisor/runsc | Stronger isolation |
| `BackendKata` | beta | VM | Kata runtime | VM-level isolation |
| `BackendFirecracker` | beta | MicroVM | Firecracker runtime | Strongest isolation |
| `BackendWASM` | beta | Sandbox | wazero | In-process WASM |
| `BackendTemporal` | stub | Workflow | Temporal client | Orchestrated execution |
| `BackendRemote` | beta | Remote | HTTP service | External runtime with signed requests |
| `BackendProxmoxLXC` | beta | Container | Proxmox API + runtime service | LXC-backed runtime service |

## Toolcode ↔ Runtime Contract

The `code` package uses the `runtime/toolcodeengine` adapter to bridge
code execution with runtime backends. The adapter maps `code.ExecuteParams`
to `runtime.ExecuteRequest`, preserving:

- **Security profile** selection
- **Resource limits** (timeouts, tool-call/chain limits)
- **ToolGateway** injection for tool discovery/execution

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
