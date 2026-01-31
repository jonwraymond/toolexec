// Package toolcode provides the code-mode orchestration layer for executing
// constrained code snippets with access to metatool helper functions.
//
// toolcode sits on top of toolindex, tooldocs, and toolrun to provide a unified
// interface for discovering, documenting, and executing tools from within code
// snippets. It maintains MCP protocol alignment (version 2025-11-25).
//
// # Architecture
//
// The package defines three main interfaces:
//
//   - [Tools]: The metatool environment exposed to code snippets, providing
//     SearchTools, ListNamespaces, DescribeTool, ListToolExamples, RunTool,
//     RunChain, and Println functions.
//
//   - [Engine]: The pluggable code execution engine that runs snippets with
//     access to the Tools environment.
//
//   - [Executor]: The main entry point that orchestrates execution, applying
//     defaults, enforcing limits, and collecting results.
//
// # Execution Limits
//
// The executor enforces two types of limits:
//
//   - Timeout: Applied via context deadline, returns [ErrLimitExceeded]
//   - MaxToolCalls: Tracks tool invocations, returns [ErrLimitExceeded] when exceeded
//
// # Tool Call Tracing
//
// Every tool invocation is recorded in a [ToolCallRecord] containing:
//   - ToolID: The canonical tool identifier
//   - Args: The arguments passed to the tool
//   - Structured: The structured result from successful execution
//   - BackendKind: The backend that executed the tool (mcp, provider, local)
//   - Error/ErrorOp: Error information if the call failed
//   - DurationMs: Execution time in milliseconds
//
// # Result Convention
//
// Code snippets should assign their final result to the `__out` variable.
// The Engine is responsible for extracting this value and returning it
// in [ExecuteResult].Value.
package code
