package code_test

import (
	"fmt"
	"time"

	"github.com/jonwraymond/toolexec/code"
)

func Example_executeParams() {
	params := code.ExecuteParams{
		Language:     "go",
		Code:         `tools.RunTool(ctx, "math:add", map[string]any{"a": 1, "b": 2})`,
		Timeout:      10 * time.Second,
		MaxToolCalls: 5,
	}

	fmt.Printf("Language: %s\n", params.Language)
	fmt.Printf("Timeout: %v\n", params.Timeout)
	fmt.Printf("MaxToolCalls: %d\n", params.MaxToolCalls)
	// Output:
	// Language: go
	// Timeout: 10s
	// MaxToolCalls: 5
}

func Example_executeResult() {
	result := code.ExecuteResult{
		Value:      42,
		DurationMs: 150,
		ToolCalls: []code.ToolCallRecord{
			{
				ToolID:      "math:add",
				Args:        map[string]any{"a": 1, "b": 2},
				Structured:  3,
				DurationMs:  5,
				BackendKind: "local",
			},
		},
		Stdout: "Debug output\n",
	}

	fmt.Printf("Value: %v\n", result.Value)
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	fmt.Printf("Tool calls: %d\n", len(result.ToolCalls))
	fmt.Printf("First tool: %s\n", result.ToolCalls[0].ToolID)
	// Output:
	// Value: 42
	// Duration: 150ms
	// Tool calls: 1
	// First tool: math:add
}

func Example_toolCallRecord() {
	record := code.ToolCallRecord{
		ToolID:      "text:format",
		Args:        map[string]any{"text": "hello", "style": "upper"},
		Structured:  "HELLO",
		DurationMs:  3,
		BackendKind: "local",
	}

	fmt.Printf("Tool: %s\n", record.ToolID)
	fmt.Printf("Duration: %dms\n", record.DurationMs)
	fmt.Printf("Result: %v\n", record.Structured)
	fmt.Printf("Backend: %s\n", record.BackendKind)
	// Output:
	// Tool: text:format
	// Duration: 3ms
	// Result: HELLO
	// Backend: local
}

func Example_errors() {
	// code.ErrConfiguration is returned when Config is invalid
	fmt.Printf("ErrConfiguration: %v\n", code.ErrConfiguration)
	// code.ErrLimitExceeded is returned when limits are hit
	fmt.Printf("ErrLimitExceeded: %v\n", code.ErrLimitExceeded)
	// Output:
	// ErrConfiguration: configuration error
	// ErrLimitExceeded: limit exceeded
}
