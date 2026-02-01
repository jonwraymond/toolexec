// Package shared provides common utilities for backend implementations.
package shared

import (
	"encoding/json"
	"strings"
)

// ExtractOutValue extracts the __out value from stdout if present.
// This follows the toolruntime convention where gateway proxies output JSON with __out key.
//
// Returns:
//   - value: the extracted __out value, or nil if not found
//   - remainingStdout: stdout with the JSON line containing __out removed
//
// Behavior:
//   - Searches each line of stdout for valid JSON containing __out
//   - If found, extracts the __out value (first occurrence) and removes that line from stdout
//   - If JSON is invalid or __out is not present, returns nil and original stdout
//   - Non-JSON lines are preserved in remainingStdout
func ExtractOutValue(stdout string) (value any, remainingStdout string) {
	if stdout == "" {
		return nil, ""
	}

	lines := strings.Split(stdout, "\n")
	var outputLines []string
	var foundOut any
	foundLine := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			outputLines = append(outputLines, line)
			continue
		}

		// Try to parse as JSON
		var jsonObj map[string]any
		if err := json.Unmarshal([]byte(trimmed), &jsonObj); err == nil {
			// Check if __out key exists
			if outVal, exists := jsonObj["__out"]; exists && foundLine == -1 {
				// First occurrence of __out
				foundOut = outVal
				foundLine = i
				// Skip this line in output (it's been extracted)
				continue
			}
		}

		// Not JSON with __out, keep the line
		outputLines = append(outputLines, line)
	}

	return foundOut, strings.Join(outputLines, "\n")
}
