// Package main demonstrates custom runtime configuration.
//
// This example shows how to:
// - Configure different security profiles
// - Set up runtime backends for code execution
// - Use security profiles to control execution environment
//
// Run with: go run ./examples/runtime
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/exec"
	"github.com/jonwraymond/toolexec/runtime"
	"github.com/jonwraymond/toolfoundation/model"
)

func main() {
	ctx := context.Background()

	// 1. Setup infrastructure
	idx := index.NewInMemoryIndex()
	docs := tooldoc.NewInMemoryStore(tooldoc.StoreOptions{Index: idx})

	// 2. Register tools
	tool := model.Tool{
		Tool: mcp.Tool{
			Name:        "process_data",
			Description: "Processes data with configurable security",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data": map[string]any{"type": "string"},
				},
			},
		},
		Namespace: "processing",
		Tags:      []string{"data", "security"},
	}

	if err := idx.RegisterTool(tool, model.NewLocalBackend("process-handler")); err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// 3. Demonstrate different security profiles
	profiles := []runtime.SecurityProfile{
		runtime.ProfileDev,
		runtime.ProfileStandard,
		runtime.ProfileHardened,
	}

	for _, profile := range profiles {
		fmt.Printf("=== Security Profile: %s ===\n", profile)
		fmt.Printf("Valid: %v\n", profile.IsValid())

		// Create executor with this profile
		executor, err := exec.New(exec.Options{
			Index:           idx,
			Docs:            docs,
			SecurityProfile: profile,
			LocalHandlers: map[string]exec.Handler{
				"process-handler": func(ctx context.Context, args map[string]any) (any, error) {
					data, _ := args["data"].(string)
					return fmt.Sprintf("Processed: %s (profile: %s)", data, profile), nil
				},
			},
			ValidateInput:  false,
			ValidateOutput: false,
		})
		if err != nil {
			log.Printf("Failed to create executor: %v", err)
			continue
		}

		// Execute with this profile
		result, err := executor.RunTool(ctx, "processing:process_data", map[string]any{
			"data": "sample input",
		})
		if err != nil {
			log.Printf("Execution failed: %v", err)
			continue
		}

		fmt.Printf("Result: %v\n", result.Value)
		fmt.Println()
	}

	// 4. Show profile characteristics
	fmt.Println("=== Profile Characteristics ===")
	fmt.Println()
	fmt.Println("ProfileDev:")
	fmt.Println("  - Minimal isolation")
	fmt.Println("  - Fast execution")
	fmt.Println("  - Best for development/testing")
	fmt.Println()
	fmt.Println("ProfileStandard:")
	fmt.Println("  - Container-based isolation")
	fmt.Println("  - Balanced security/performance")
	fmt.Println("  - Suitable for most production use")
	fmt.Println()
	fmt.Println("ProfileHardened:")
	fmt.Println("  - Maximum isolation (gVisor, Firecracker)")
	fmt.Println("  - Strictest security controls")
	fmt.Println("  - For untrusted code execution")
}
