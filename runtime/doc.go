// Package toolruntime provides execution runtime and isolation boundaries for
// code-oriented orchestration. It sits underneath toolcode and provides:
//
//   - Backend-agnostic runtime interface for executing code in sandboxed environments
//   - Pluggable sandbox backends (from unsafe development mode to hardened isolation)
//   - Clean trust boundary for running untrusted code that can still call tools
//   - ToolGateway abstraction for exposing tool discovery and execution to sandboxes
//
// The runtime enforces security through SecurityProfiles that determine which
// backends are allowed and what resource limits apply. The ToolGateway provides
// a proxy interface for sandboxed code to discover and execute tools without
// direct access to host resources.
//
// # Architecture
//
// The main types are:
//
//   - Runtime: Main execution interface that routes requests to backends
//   - Backend: Sandbox implementation (see Backend Kinds below)
//   - ToolGateway: Interface for tool operations exposed to sandboxed code
//   - ExecuteRequest/ExecuteResult: Request/response types for execution
//
// # Security Profiles
//
// Three security profiles are supported:
//
//   - ProfileDev: Development mode with minimal restrictions (unsafe)
//   - ProfileStandard: Standard isolation (no network, read-only rootfs)
//   - ProfileHardened: Maximum isolation with seccomp, gVisor/Kata/microVM
//
// # Backend Kinds
//
// The following execution backends are supported:
//
//   - BackendUnsafeHost: Direct host execution (dev only, no isolation)
//   - BackendDocker: Docker containers with cgroups and seccomp
//   - BackendContainerd: Containerd for infrastructure-native deployments
//   - BackendKubernetes: Short-lived pods/jobs with scheduling
//   - BackendGVisor: Strong isolation via gVisor/runsc
//   - BackendKata: VM-level isolation via Kata Containers
//   - BackendFirecracker: MicroVM isolation (strongest)
//   - BackendWASM: WebAssembly in-process isolation
//   - BackendTemporal: Workflow orchestration (composes with sandbox backends)
//   - BackendRemote: Generic remote execution service
//
// # Security Requirements
//
// All non-unsafe backends MUST:
//
//  1. Run as non-root
//  2. Enforce timeouts and cancellation
//  3. Enforce tool call and chain step limits
//  4. Deny host filesystem access by default
//  5. Deny network egress by default
//  6. Provide resource controls where available
//  7. Treat tool schemas/docs/annotations as untrusted input
//
// Backends that cannot enforce a given limit must report that clearly
// via the LimitsEnforced field in ExecuteResult.
package runtime
