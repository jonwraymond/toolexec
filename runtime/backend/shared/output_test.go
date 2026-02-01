package shared

import (
	"testing"
)

func TestExtractOutValue_ValidJSONWithOut(t *testing.T) {
	tests := []struct {
		name          string
		stdout        string
		wantValue     any
		wantRemaining string
	}{
		{
			name:          "simple string value",
			stdout:        `{"__out": "hello world"}`,
			wantValue:     "hello world",
			wantRemaining: "",
		},
		{
			name:          "number value",
			stdout:        `{"__out": 42}`,
			wantValue:     float64(42), // JSON numbers unmarshal to float64
			wantRemaining: "",
		},
		{
			name:          "boolean value",
			stdout:        `{"__out": true}`,
			wantValue:     true,
			wantRemaining: "",
		},
		{
			name:          "null value",
			stdout:        `{"__out": null}`,
			wantValue:     nil,
			wantRemaining: "",
		},
		{
			name:   "object value",
			stdout: `{"__out": {"key": "value", "num": 123}}`,
			wantValue: map[string]any{
				"key": "value",
				"num": float64(123),
			},
			wantRemaining: "",
		},
		{
			name:          "array value",
			stdout:        `{"__out": [1, 2, 3]}`,
			wantValue:     []any{float64(1), float64(2), float64(3)},
			wantRemaining: "",
		},
		{
			name:          "JSON with other fields",
			stdout:        `{"status": "ok", "__out": "result", "timestamp": 123}`,
			wantValue:     "result",
			wantRemaining: "",
		},
		{
			name:          "with leading/trailing whitespace",
			stdout:        `  {"__out": "test"}  `,
			wantValue:     "test",
			wantRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, remaining := ExtractOutValue(tt.stdout)

			if !equalAny(value, tt.wantValue) {
				t.Errorf("ExtractOutValue() value = %v (%T), want %v (%T)", value, value, tt.wantValue, tt.wantValue)
			}

			if remaining != tt.wantRemaining {
				t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, tt.wantRemaining)
			}
		})
	}
}

func TestExtractOutValue_ValidJSONWithoutOut(t *testing.T) {
	tests := []struct {
		name          string
		stdout        string
		wantRemaining string
	}{
		{
			name:          "empty object",
			stdout:        `{}`,
			wantRemaining: `{}`,
		},
		{
			name:          "object without __out",
			stdout:        `{"status": "ok", "result": "data"}`,
			wantRemaining: `{"status": "ok", "result": "data"}`,
		},
		{
			name:          "object with similar key",
			stdout:        `{"out": "value"}`,
			wantRemaining: `{"out": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, remaining := ExtractOutValue(tt.stdout)

			if value != nil {
				t.Errorf("ExtractOutValue() value = %v, want nil", value)
			}

			if remaining != tt.wantRemaining {
				t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, tt.wantRemaining)
			}
		})
	}
}

func TestExtractOutValue_InvalidJSON(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
	}{
		{
			name:   "not JSON",
			stdout: "hello world",
		},
		{
			name:   "malformed JSON",
			stdout: `{"__out": "value"`,
		},
		{
			name:   "JSON array instead of object",
			stdout: `["__out", "value"]`,
		},
		{
			name:   "plain text output",
			stdout: "Error: something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, remaining := ExtractOutValue(tt.stdout)

			if value != nil {
				t.Errorf("ExtractOutValue() value = %v, want nil", value)
			}

			if remaining != tt.stdout {
				t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, tt.stdout)
			}
		})
	}
}

func TestExtractOutValue_MultilineOutput(t *testing.T) {
	tests := []struct {
		name          string
		stdout        string
		wantValue     any
		wantRemaining string
	}{
		{
			name: "JSON on first line",
			stdout: `{"__out": "result"}
Some other output
More output`,
			wantValue: "result",
			wantRemaining: `Some other output
More output`,
		},
		{
			name: "JSON on middle line",
			stdout: `Starting execution...
{"__out": "result"}
Execution complete`,
			wantValue: "result",
			wantRemaining: `Starting execution...
Execution complete`,
		},
		{
			name: "JSON on last line",
			stdout: `Log message 1
Log message 2
{"__out": "final result"}`,
			wantValue: "final result",
			wantRemaining: `Log message 1
Log message 2`,
		},
		{
			name: "multiple JSON lines, only one with __out",
			stdout: `{"status": "running"}
{"__out": "done"}
{"cleanup": "complete"}`,
			wantValue: "done",
			wantRemaining: `{"status": "running"}
{"cleanup": "complete"}`,
		},
		{
			name: "empty lines preserved",
			stdout: `Line 1

{"__out": "value"}

Line 4`,
			wantValue: "value",
			wantRemaining: `Line 1


Line 4`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, remaining := ExtractOutValue(tt.stdout)

			if !equalAny(value, tt.wantValue) {
				t.Errorf("ExtractOutValue() value = %v, want %v", value, tt.wantValue)
			}

			if remaining != tt.wantRemaining {
				t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, tt.wantRemaining)
			}
		})
	}
}

func TestExtractOutValue_EmptyInput(t *testing.T) {
	value, remaining := ExtractOutValue("")

	if value != nil {
		t.Errorf("ExtractOutValue(\"\") value = %v, want nil", value)
	}

	if remaining != "" {
		t.Errorf("ExtractOutValue(\"\") remaining = %q, want \"\"", remaining)
	}
}

func TestExtractOutValue_OnlyWhitespace(t *testing.T) {
	tests := []string{
		"   ",
		"\n\n",
		"\t\t",
		" \n \n ",
	}

	for _, stdout := range tests {
		t.Run("whitespace", func(t *testing.T) {
			value, remaining := ExtractOutValue(stdout)

			if value != nil {
				t.Errorf("ExtractOutValue() value = %v, want nil", value)
			}

			if remaining != stdout {
				t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, stdout)
			}
		})
	}
}

func TestExtractOutValue_MultipleOutLines(t *testing.T) {
	// If there are multiple __out lines, only the first should be extracted
	stdout := `{"__out": "first"}
{"__out": "second"}`

	value, remaining := ExtractOutValue(stdout)

	if value != "first" {
		t.Errorf("ExtractOutValue() value = %v, want \"first\"", value)
	}

	// Second line should remain
	if remaining != "{\"__out\": \"second\"}" {
		t.Errorf("ExtractOutValue() remaining = %q, want %q", remaining, "{\"__out\": \"second\"}")
	}
}

// equalAny compares two any values, handling special cases like maps and slices
func equalAny(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// For maps, need deep comparison
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for k, v := range aMap {
			if !equalAny(v, bMap[k]) {
				return false
			}
		}
		return true
	}

	// For slices, need deep comparison
	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !equalAny(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}

	// For simple types, use ==
	return a == b
}
