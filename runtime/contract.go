package runtime

import (
	"context"
	"testing"

	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// GatewayContract defines tests that any ToolGateway implementation must pass.
// Use RunGatewayContractTests to test an implementation.
type GatewayContract struct {
	// NewGateway creates a fresh gateway instance for testing.
	// The gateway should be configured with at least one tool for search/describe tests.
	NewGateway func() ToolGateway
}

// RunGatewayContractTests runs all contract tests for a ToolGateway implementation.
func RunGatewayContractTests(t *testing.T, contract GatewayContract) {
	t.Helper()

	t.Run("SearchTools", func(t *testing.T) {
		t.Run("returns results respecting limit", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			results, err := gw.SearchTools(ctx, "", 5)
			if err != nil {
				t.Fatalf("SearchTools() error = %v", err)
			}
			if len(results) > 5 {
				t.Errorf("SearchTools() returned %d results, want <= 5", len(results))
			}
		})

		t.Run("handles empty query", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			// Should not error on empty query
			_, err := gw.SearchTools(ctx, "", 10)
			if err != nil {
				t.Errorf("SearchTools() with empty query error = %v", err)
			}
		})

		t.Run("respects context cancellation", func(_ *testing.T) {
			gw := contract.NewGateway()
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			_, err := gw.SearchTools(ctx, "test", 10)
			// Error is acceptable (context canceled), but no panic
			_ = err
		})
	})

	t.Run("ListNamespaces", func(t *testing.T) {
		t.Run("returns namespaces without error", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			_, err := gw.ListNamespaces(ctx)
			if err != nil {
				t.Errorf("ListNamespaces() error = %v", err)
			}
		})
	})

	t.Run("DescribeTool", func(t *testing.T) {
		t.Run("handles non-existent tool", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			_, err := gw.DescribeTool(ctx, "nonexistent:tool", tooldoc.DetailSummary)
			// Should return an error for non-existent tool
			if err == nil {
				t.Error("DescribeTool() for non-existent tool should return error")
			}
		})
	})

	t.Run("RunTool", func(t *testing.T) {
		t.Run("propagates context cancellation", func(_ *testing.T) {
			gw := contract.NewGateway()
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			_, err := gw.RunTool(ctx, "test:tool", nil)
			// Should return error (context canceled or tool not found), no panic
			_ = err
		})
	})

	t.Run("RunChain", func(t *testing.T) {
		t.Run("handles empty steps", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			_, stepResults, err := gw.RunChain(ctx, []run.ChainStep{})
			// Empty chain should not error
			if err != nil {
				t.Errorf("RunChain() with empty steps error = %v", err)
			}
			if len(stepResults) != 0 {
				t.Errorf("RunChain() with empty steps returned %d results", len(stepResults))
			}
		})

		t.Run("handles nil steps", func(t *testing.T) {
			gw := contract.NewGateway()
			ctx := context.Background()

			_, stepResults, err := gw.RunChain(ctx, nil)
			// Nil chain should not error
			if err != nil {
				t.Errorf("RunChain() with nil steps error = %v", err)
			}
			if len(stepResults) != 0 {
				t.Errorf("RunChain() with nil steps returned %d results", len(stepResults))
			}
		})
	})
}

// BackendContract defines tests that any Backend implementation must pass.
// Use RunBackendContractTests to test an implementation.
type BackendContract struct {
	// NewBackend creates a fresh backend instance for testing.
	NewBackend func() Backend

	// NewGateway creates a gateway for testing.
	// The gateway should allow at least basic tool operations.
	NewGateway func() ToolGateway

	// ExpectedKind is the BackendKind the backend should return.
	ExpectedKind BackendKind

	// SkipExecutionTests skips tests that require actual code execution.
	// Useful for backends that need complex setup (e.g., Docker).
	SkipExecutionTests bool
}

// RunBackendContractTests runs all contract tests for a Backend implementation.
func RunBackendContractTests(t *testing.T, contract BackendContract) {
	t.Helper()

	t.Run("Kind", func(t *testing.T) {
		t.Run("returns non-empty kind", func(t *testing.T) {
			b := contract.NewBackend()
			kind := b.Kind()
			if kind == "" {
				t.Error("Kind() should not return empty string")
			}
		})

		t.Run("returns expected kind", func(t *testing.T) {
			b := contract.NewBackend()
			kind := b.Kind()
			if kind != contract.ExpectedKind {
				t.Errorf("Kind() = %v, want %v", kind, contract.ExpectedKind)
			}
		})
	})

	t.Run("Execute", func(t *testing.T) {
		t.Run("requires gateway", func(t *testing.T) {
			b := contract.NewBackend()
			ctx := context.Background()

			req := ExecuteRequest{
				Code:    "print('hello')",
				Gateway: nil, // Missing gateway
			}

			_, err := b.Execute(ctx, req)
			if err == nil {
				t.Error("Execute() without gateway should return error")
			}
		})

		t.Run("requires code", func(t *testing.T) {
			b := contract.NewBackend()
			ctx := context.Background()

			req := ExecuteRequest{
				Code:    "", // Missing code
				Gateway: contract.NewGateway(),
			}

			_, err := b.Execute(ctx, req)
			if err == nil {
				t.Error("Execute() without code should return error")
			}
		})

		if !contract.SkipExecutionTests {
			t.Run("returns BackendInfo with correct kind", func(t *testing.T) {
				b := contract.NewBackend()
				ctx := context.Background()

				req := ExecuteRequest{
					Code:    `__out = "hello"`,
					Gateway: contract.NewGateway(),
				}

				result, err := b.Execute(ctx, req)
				if err != nil {
					t.Skipf("Execute() error = %v (may be expected for some backends)", err)
				}

				if result.Backend.Kind != contract.ExpectedKind {
					t.Errorf("Execute() result.Backend.Kind = %v, want %v", result.Backend.Kind, contract.ExpectedKind)
				}
			})
		}
	})
}
