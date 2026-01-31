package code

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToolCallRecord_JSONSerialization(t *testing.T) {
	record := ToolCallRecord{
		ToolID:      "namespace:tool",
		Args:        map[string]any{"key": "value"},
		Structured:  map[string]any{"result": 42},
		BackendKind: "mcp",
		Error:       "some error",
		ErrorOp:     "run",
		DurationMs:  150,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify JSON tags
	jsonStr := string(data)
	expectedTags := []string{
		`"toolId"`,
		`"args"`,
		`"structured"`,
		`"backendKind"`,
		`"error"`,
		`"errorOp"`,
		`"durationMs"`,
	}
	for _, tag := range expectedTags {
		if !contains(jsonStr, tag) {
			t.Errorf("expected JSON to contain %s, got: %s", tag, jsonStr)
		}
	}

	// Roundtrip
	var decoded ToolCallRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ToolID != record.ToolID {
		t.Errorf("ToolID: got %q, want %q", decoded.ToolID, record.ToolID)
	}
	if decoded.DurationMs != record.DurationMs {
		t.Errorf("DurationMs: got %d, want %d", decoded.DurationMs, record.DurationMs)
	}
}

func TestToolCallRecord_JSONOmitEmpty(t *testing.T) {
	record := ToolCallRecord{
		ToolID:     "tool",
		DurationMs: 100,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// These should be omitted when empty/nil
	shouldOmit := []string{`"args"`, `"structured"`, `"backendKind"`, `"error"`, `"errorOp"`}
	for _, field := range shouldOmit {
		if contains(jsonStr, field) {
			t.Errorf("expected %s to be omitted, got: %s", field, jsonStr)
		}
	}

	// Required fields should always be present
	shouldPresent := []string{`"toolId"`, `"durationMs"`}
	for _, field := range shouldPresent {
		if !contains(jsonStr, field) {
			t.Errorf("expected %s to be present, got: %s", field, jsonStr)
		}
	}
}

func TestToolCallRecord_ZeroValue(t *testing.T) {
	var record ToolCallRecord

	if record.ToolID != "" {
		t.Errorf("expected empty ToolID, got %q", record.ToolID)
	}
	if record.Args != nil {
		t.Errorf("expected nil Args, got %v", record.Args)
	}
	if record.Structured != nil {
		t.Errorf("expected nil Structured, got %v", record.Structured)
	}
	if record.DurationMs != 0 {
		t.Errorf("expected zero DurationMs, got %d", record.DurationMs)
	}
}

func TestExecuteParams_JSONSerialization(t *testing.T) {
	params := ExecuteParams{
		Language:     "go",
		Code:         "return 42",
		Timeout:      5 * time.Second,
		MaxToolCalls: 10,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	expectedTags := []string{`"language"`, `"code"`, `"timeout"`, `"maxToolCalls"`}
	for _, tag := range expectedTags {
		if !contains(jsonStr, tag) {
			t.Errorf("expected JSON to contain %s, got: %s", tag, jsonStr)
		}
	}

	// Roundtrip
	var decoded ExecuteParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Language != params.Language {
		t.Errorf("Language: got %q, want %q", decoded.Language, params.Language)
	}
	if decoded.Code != params.Code {
		t.Errorf("Code: got %q, want %q", decoded.Code, params.Code)
	}
}

func TestExecuteParams_Defaults(t *testing.T) {
	var params ExecuteParams

	if params.Language != "" {
		t.Errorf("expected empty Language, got %q", params.Language)
	}
	if params.Code != "" {
		t.Errorf("expected empty Code, got %q", params.Code)
	}
	if params.Timeout != 0 {
		t.Errorf("expected zero Timeout, got %v", params.Timeout)
	}
	if params.MaxToolCalls != 0 {
		t.Errorf("expected zero MaxToolCalls, got %d", params.MaxToolCalls)
	}
}

func TestExecuteResult_JSONSerialization(t *testing.T) {
	result := ExecuteResult{
		Value:  map[string]any{"answer": 42},
		Stdout: "hello\n",
		Stderr: "warning\n",
		ToolCalls: []ToolCallRecord{
			{ToolID: "tool1", DurationMs: 100},
		},
		DurationMs: 500,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	expectedTags := []string{`"value"`, `"stdout"`, `"stderr"`, `"toolCalls"`, `"durationMs"`}
	for _, tag := range expectedTags {
		if !contains(jsonStr, tag) {
			t.Errorf("expected JSON to contain %s, got: %s", tag, jsonStr)
		}
	}

	// Roundtrip
	var decoded ExecuteResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Stdout != result.Stdout {
		t.Errorf("Stdout: got %q, want %q", decoded.Stdout, result.Stdout)
	}
	if decoded.DurationMs != result.DurationMs {
		t.Errorf("DurationMs: got %d, want %d", decoded.DurationMs, result.DurationMs)
	}
}

func TestExecuteResult_JSONOmitEmpty(t *testing.T) {
	result := ExecuteResult{
		DurationMs: 100,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// These should be omitted when empty/nil
	shouldOmit := []string{`"value"`, `"stdout"`, `"stderr"`, `"toolCalls"`}
	for _, field := range shouldOmit {
		if contains(jsonStr, field) {
			t.Errorf("expected %s to be omitted, got: %s", field, jsonStr)
		}
	}

	// durationMs should always be present
	if !contains(jsonStr, `"durationMs"`) {
		t.Errorf("expected durationMs to be present, got: %s", jsonStr)
	}
}

func TestExecuteResult_ZeroValue(t *testing.T) {
	var result ExecuteResult

	if result.Value != nil {
		t.Errorf("expected nil Value, got %v", result.Value)
	}
	if result.Stdout != "" {
		t.Errorf("expected empty Stdout, got %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Errorf("expected empty Stderr, got %q", result.Stderr)
	}
	if result.ToolCalls != nil {
		t.Errorf("expected nil ToolCalls, got %v", result.ToolCalls)
	}
	if result.DurationMs != 0 {
		t.Errorf("expected zero DurationMs, got %d", result.DurationMs)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
