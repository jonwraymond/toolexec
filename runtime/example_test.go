package runtime_test

import (
	"context"
	"fmt"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

func Example_securityProfiles() {
	// Available security profiles
	profiles := []runtime.SecurityProfile{
		runtime.ProfileDev,
		runtime.ProfileStandard,
		runtime.ProfileHardened,
	}

	fmt.Println("Security Profiles:")
	for _, p := range profiles {
		fmt.Printf("  %s (valid: %v)\n", p, p.IsValid())
	}

	// Invalid profile
	invalid := runtime.SecurityProfile("unknown")
	fmt.Printf("  %s (valid: %v)\n", invalid, invalid.IsValid())
	// Output:
	// Security Profiles:
	//   dev (valid: true)
	//   standard (valid: true)
	//   hardened (valid: true)
	//   unknown (valid: false)
}

func Example_executeRequest() {
	req := runtime.ExecuteRequest{
		Code:     `print("Hello, World!")`,
		Language: "python",
		Profile:  runtime.ProfileDev,
		Timeout:  30 * time.Second,
		Limits: runtime.Limits{
			MaxToolCalls:   100,
			MemoryBytes:    512 * 1024 * 1024, // 512MB
			CPUQuotaMillis: 60000,             // 60s
		},
	}

	fmt.Printf("Language: %s\n", req.Language)
	fmt.Printf("Profile: %s\n", req.Profile)
	fmt.Printf("Timeout: %v\n", req.Timeout)
	fmt.Printf("MaxToolCalls: %d\n", req.Limits.MaxToolCalls)
	// Output:
	// Language: python
	// Profile: dev
	// Timeout: 30s
	// MaxToolCalls: 100
}

func Example_executeResult() {
	result := runtime.ExecuteResult{
		Value:    42,
		Stdout:   "Computation complete\n",
		Duration: 150 * time.Millisecond,
		ToolCalls: []runtime.ToolCallRecord{
			{
				ToolID:      "math:add",
				BackendKind: "local",
				Duration:    5 * time.Millisecond,
			},
		},
	}

	fmt.Printf("Value: %v\n", result.Value)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Tool calls: %d\n", len(result.ToolCalls))
	// Output:
	// Value: 42
	// Duration: 150ms
	// Tool calls: 1
}

func ExampleDefaultRuntime() {
	// Create a mock backend for the example
	backend := &mockBackend{
		result: runtime.ExecuteResult{
			Value:    "executed",
			Duration: 100 * time.Millisecond,
		},
	}

	// Create runtime with configuration
	rt := runtime.NewDefaultRuntime(runtime.RuntimeConfig{
		Backends: map[runtime.SecurityProfile]runtime.Backend{
			runtime.ProfileDev: backend,
		},
		DefaultProfile: runtime.ProfileDev,
	})

	// Execute code (requires a Gateway)
	ctx := context.Background()
	result, err := rt.Execute(ctx, runtime.ExecuteRequest{
		Code:     `return 42`,
		Language: "go",
		Gateway:  &mockGateway{},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result.Value)
	fmt.Printf("Duration: %v\n", result.Duration)
	// Output:
	// Result: executed
	// Duration: 100ms
}

func Example_limits() {
	limits := runtime.Limits{
		MaxToolCalls:   50,
		MaxChainSteps:  10,
		CPUQuotaMillis: 60000,              // 60 seconds
		MemoryBytes:    1024 * 1024 * 1024, // 1GB
		PidsMax:        100,
	}

	fmt.Printf("Max tool calls: %d\n", limits.MaxToolCalls)
	fmt.Printf("Max chain steps: %d\n", limits.MaxChainSteps)
	fmt.Printf("CPU quota: %dms\n", limits.CPUQuotaMillis)
	fmt.Printf("Memory: %dMB\n", limits.MemoryBytes/(1024*1024))
	// Output:
	// Max tool calls: 50
	// Max chain steps: 10
	// CPU quota: 60000ms
	// Memory: 1024MB
}

func Example_errors() {
	fmt.Printf("ErrMissingGateway: %v\n", runtime.ErrMissingGateway)
	fmt.Printf("ErrRuntimeUnavailable: %v\n", runtime.ErrRuntimeUnavailable)
	fmt.Printf("ErrBackendDenied: %v\n", runtime.ErrBackendDenied)
	// Output:
	// ErrMissingGateway: gateway is required
	// ErrRuntimeUnavailable: runtime unavailable
	// ErrBackendDenied: backend denied by policy
}

// mockBackend is a minimal Backend implementation for examples.
type mockBackend struct {
	result runtime.ExecuteResult
}

func (b *mockBackend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	return b.result, nil
}

func (b *mockBackend) Kind() runtime.BackendKind { return "mock" }

// mockGateway is a minimal ToolGateway implementation for examples.
type mockGateway struct{}

func (g *mockGateway) SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	return nil, nil
}

func (g *mockGateway) ListNamespaces(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (g *mockGateway) DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, nil
}

func (g *mockGateway) ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	return nil, nil
}

func (g *mockGateway) RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error) {
	return run.RunResult{}, nil
}

func (g *mockGateway) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, nil
}

// Verify interface compliance
var _ runtime.Backend = (*mockBackend)(nil)
var _ runtime.ToolGateway = (*mockGateway)(nil)
