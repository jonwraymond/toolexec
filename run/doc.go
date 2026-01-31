// Package run is the execution layer for MCP-style tools defined in the model package
// and resolved via toolindex.
//
// It:
//   - Accepts a canonical tool ID (namespace:name) plus arguments
//   - Resolves the tool definition and a backend binding
//   - Validates inputs and (optionally) outputs against JSON Schema
//   - Executes the tool across MCP, provider, and local backends
//   - Supports sequential chains with explicit data passing
//
// # Scope
//
// run handles execution and chaining only. No discovery, ranking, or documentation.
// Target MCP protocol version: 2025-11-25 (via model.MCPVersion).
//
// # Resolution
//
// Given a tool ID:
//  1. Attempt Index.GetTool(id) when Index is configured
//  2. If not found, fall back to injected resolvers (ToolResolver, BackendsResolver)
//
// # Backend Selection
//
// When multiple backends exist for the same tool, a configurable BackendSelector
// chooses which to use. The default uses toolindex.DefaultBackendSelector which
// implements priority: local > provider > mcp.
//
// # Validation
//
// Input validation is performed before execution using model.SchemaValidator.
// Output validation is performed after execution when tool.OutputSchema is present.
// Both can be configured via ValidateInput and ValidateOutput options.
//
// # Chains
//
// Chains execute steps sequentially with explicit data passing.
// If UsePrevious is true, the prior step's structured result is injected
// at args["previous"] (overwriting any existing value).
// Chains stop on first error (v1 policy).
//
// # Example
//
//	runner := run.NewRunner(
//	    run.WithIndex(myIndex),
//	    run.WithMCPExecutor(myMCPExecutor),
//	)
//
//	result, err := runner.Run(ctx, "myns:mytool", map[string]any{"input": "value"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Structured)
package run
