package run

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func TestErrToolNotFound_Is(t *testing.T) {
	err := ErrToolNotFound
	if !errors.Is(err, ErrToolNotFound) {
		t.Error("errors.Is(ErrToolNotFound, ErrToolNotFound) = false, want true")
	}

	wrapped := fmt.Errorf("lookup failed: %w", ErrToolNotFound)
	if !errors.Is(wrapped, ErrToolNotFound) {
		t.Error("errors.Is(wrapped, ErrToolNotFound) = false, want true")
	}
}

func TestErrNoBackends_Is(t *testing.T) {
	err := ErrNoBackends
	if !errors.Is(err, ErrNoBackends) {
		t.Error("errors.Is(ErrNoBackends, ErrNoBackends) = false, want true")
	}
}

func TestErrValidation_Is(t *testing.T) {
	err := ErrValidation
	if !errors.Is(err, ErrValidation) {
		t.Error("errors.Is(ErrValidation, ErrValidation) = false, want true")
	}
}

func TestErrExecution_Is(t *testing.T) {
	err := ErrExecution
	if !errors.Is(err, ErrExecution) {
		t.Error("errors.Is(ErrExecution, ErrExecution) = false, want true")
	}
}

func TestErrOutputValidation_Is(t *testing.T) {
	err := ErrOutputValidation
	if !errors.Is(err, ErrOutputValidation) {
		t.Error("errors.Is(ErrOutputValidation, ErrOutputValidation) = false, want true")
	}
}

func TestErrStreamNotSupported_Is(t *testing.T) {
	err := ErrStreamNotSupported
	if !errors.Is(err, ErrStreamNotSupported) {
		t.Error("errors.Is(ErrStreamNotSupported, ErrStreamNotSupported) = false, want true")
	}
}

func TestToolError_Error(t *testing.T) {
	backend := testMCPBackend("server1")
	innerErr := errors.New("connection timeout")

	err := &ToolError{
		ToolID:  "myns:mytool",
		Backend: &backend,
		Op:      "execute",
		Err:     innerErr,
	}

	msg := err.Error()

	// Should contain tool ID
	if !containsString(msg, "myns:mytool") {
		t.Errorf("Error() should contain tool ID, got %q", msg)
	}

	// Should contain operation
	if !containsString(msg, "execute") {
		t.Errorf("Error() should contain operation, got %q", msg)
	}

	// Should contain underlying error
	if !containsString(msg, "connection timeout") {
		t.Errorf("Error() should contain underlying error, got %q", msg)
	}
}

func TestToolError_Error_NoBackend(t *testing.T) {
	err := &ToolError{
		ToolID: "mytool",
		Op:     "resolve",
		Err:    ErrToolNotFound,
	}

	msg := err.Error()
	if !containsString(msg, "mytool") {
		t.Errorf("Error() should contain tool ID, got %q", msg)
	}
	if !containsString(msg, "resolve") {
		t.Errorf("Error() should contain operation, got %q", msg)
	}
}

func TestToolError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &ToolError{
		ToolID: "mytool",
		Op:     "execute",
		Err:    innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), innerErr)
	}
}

func TestToolError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    *ToolError
		target error
		want   bool
	}{
		{
			name: "wraps ErrToolNotFound",
			err: &ToolError{
				ToolID: "mytool",
				Op:     "resolve",
				Err:    ErrToolNotFound,
			},
			target: ErrToolNotFound,
			want:   true,
		},
		{
			name: "wraps ErrValidation",
			err: &ToolError{
				ToolID: "mytool",
				Op:     "validate_input",
				Err:    ErrValidation,
			},
			target: ErrValidation,
			want:   true,
		},
		{
			name: "wraps ErrExecution",
			err: &ToolError{
				ToolID: "mytool",
				Op:     "execute",
				Err:    ErrExecution,
			},
			target: ErrExecution,
			want:   true,
		},
		{
			name: "does not match unrelated error",
			err: &ToolError{
				ToolID: "mytool",
				Op:     "execute",
				Err:    ErrExecution,
			},
			target: ErrToolNotFound,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolError_WrapsToolindexErrNotFound(t *testing.T) {
	// ToolError wrapping index.ErrNotFound should match ErrToolNotFound
	err := &ToolError{
		ToolID: "mytool",
		Op:     "resolve",
		Err:    index.ErrNotFound,
	}

	if !errors.Is(err, ErrToolNotFound) {
		t.Error("ToolError wrapping index.ErrNotFound should match ErrToolNotFound")
	}
}

func TestToolError_Is_WithNestedToolindex(t *testing.T) {
	// Test deeper nesting: index.ErrNotFound wrapped in another error
	innerErr := fmt.Errorf("lookup failed: %w", index.ErrNotFound)
	err := &ToolError{
		ToolID: "mytool",
		Op:     "resolve",
		Err:    innerErr,
	}

	if !errors.Is(err, ErrToolNotFound) {
		t.Error("ToolError with nested index.ErrNotFound should match ErrToolNotFound")
	}
}

func TestToolError_Is_BackendContext(t *testing.T) {
	backend := testLocalBackend("handler1")
	err := &ToolError{
		ToolID:  "mytool",
		Backend: &backend,
		Op:      "execute",
		Err:     errors.New("handler panic"),
	}

	msg := err.Error()
	if !containsString(msg, "local") {
		t.Errorf("Error() should mention backend kind, got %q", msg)
	}
}

func TestToolError_Op_Values(t *testing.T) {
	// Verify common Op values work correctly
	ops := []string{"resolve", "validate_input", "execute", "validate_output", "normalize"}

	for _, op := range ops {
		err := &ToolError{
			ToolID: "mytool",
			Op:     op,
			Err:    errTest,
		}
		if !containsString(err.Error(), op) {
			t.Errorf("Error() should contain Op %q, got %q", op, err.Error())
		}
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		toolID   string
		backend  *model.ToolBackend
		op       string
		err      error
		wantNil  bool
		wantType bool
	}{
		{
			name:    "nil error returns nil",
			toolID:  "mytool",
			op:      "execute",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "wraps non-nil error",
			toolID:   "mytool",
			op:       "execute",
			err:      errTest,
			wantType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapError(tt.toolID, tt.backend, tt.op, tt.err)
			if tt.wantNil && got != nil {
				t.Errorf("WrapError() = %v, want nil", got)
			}
			if tt.wantType {
				var toolErr *ToolError
				if !errors.As(got, &toolErr) {
					t.Errorf("WrapError() should return *ToolError")
				}
			}
		})
	}
}

// containsString is a helper to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
