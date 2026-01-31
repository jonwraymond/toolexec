// Package backend provides tool execution backend abstractions and registry.
//
// This package defines the core Backend interface and provides infrastructure
// for managing multiple backend sources:
//
//   - Backend interface for tool sources (local, MCP, HTTP, gRPC)
//   - Registry for managing and discovering backends
//   - Aggregator for multi-backend tool execution
//
// # Backend Types
//
// Backends can be:
//
//   - Local: In-process handlers registered directly
//   - MCP: Model Context Protocol servers (stdio, SSE, etc.)
//   - HTTP: RESTful tool APIs
//   - gRPC: High-performance tool services
//
// # Registry
//
// The Registry manages backend lifecycle:
//
//	registry := backend.NewRegistry()
//	registry.Register(localBackend)
//	registry.Register(mcpBackend)
//
//	// List all registered backends
//	for _, b := range registry.List() {
//	    fmt.Printf("%s: %s\n", b.Kind(), b.Name())
//	}
//
// # Aggregator
//
// The Aggregator combines multiple backends for unified tool access:
//
//	agg := backend.NewAggregator(registry)
//	tools, _ := agg.ListAllTools(ctx)
//	result, _ := agg.Execute(ctx, "backend:tool", args)
package backend
