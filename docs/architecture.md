# Architecture Overview

toolexec is the execution layer for MCP-style tools. It focuses on **running
validated tools** across different backends and **isolating untrusted code**
with configurable runtimes.

## Core Packages

| Package | Responsibility |
|---------|----------------|
| `exec` | Unified facade that composes discovery + execution + docs |
| `run` | Execution pipeline with validation + chaining |
| `backend` | Backend registry and resolution |
| `runtime` | Sandbox runtimes and security profiles |
| `code` | Orchestration of code with tool access |

## Execution Flow

1. **Resolve** tool definition and backend binding
2. **Validate input** against JSON Schema
3. **Execute** tool on backend (local, provider, MCP)
4. **Normalize** results into structured output
5. **Validate output** (if OutputSchema present)

## Chaining

Chains execute sequentially. If `UsePrevious` is true, the prior step’s
structured result is injected into `args["previous"]` for the next step.

## Runtime Isolation

The `runtime` package provides isolation levels via security profiles:

- **Dev**: local / unsafe execution with explicit opt‑in
- **Standard**: container or sandbox runtime
- **Hardened**: strongest isolation (Docker/gVisor/WASM)

Concrete runtime SDK clients (Kubernetes, Proxmox, remote HTTP) live in
`toolexec-integrations` and are injected into the core backends via interfaces.

## Observability

Execution surfaces timing and tool call metadata in `exec.Result` and
`run.RunResult`, enabling tracing and audits in higher layers.

## Related Docs

- [Schemas and Contracts](schemas.md)
- [Design Notes](design-notes.md)
- [User Journey](user-journey.md)
