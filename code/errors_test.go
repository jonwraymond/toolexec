package code

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrCodeExecution_Sentinel(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrCodeExecution)
	if !errors.Is(err, ErrCodeExecution) {
		t.Error("expected errors.Is to match ErrCodeExecution")
	}
}

func TestErrConfiguration_Sentinel(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrConfiguration)
	if !errors.Is(err, ErrConfiguration) {
		t.Error("expected errors.Is to match ErrConfiguration")
	}
}

func TestErrLimitExceeded_Sentinel(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", ErrLimitExceeded)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Error("expected errors.Is to match ErrLimitExceeded")
	}
}

func TestCodeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      CodeError
		expected string
	}{
		{
			name: "with line and column",
			err: CodeError{
				Message: "syntax error",
				Line:    10,
				Column:  5,
			},
			expected: "syntax error (line 10, col 5)",
		},
		{
			name: "with line only",
			err: CodeError{
				Message: "undefined variable",
				Line:    3,
				Column:  0,
			},
			expected: "undefined variable (line 3, col 0)",
		},
		{
			name: "no line info",
			err: CodeError{
				Message: "runtime error",
				Line:    0,
				Column:  0,
			},
			expected: "runtime error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCodeError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying cause")
	err := &CodeError{
		Message: "code failed",
		Line:    1,
		Column:  1,
		Err:     underlying,
	}

	// Verify errors.As can extract the CodeError
	var codeErr *CodeError
	if !errors.As(err, &codeErr) {
		t.Error("expected errors.As to extract CodeError")
	}

	// Verify Unwrap returns the underlying error
	unwrapped := errors.Unwrap(err)
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}

	// Verify errors.Is can find the underlying error
	if !errors.Is(err, underlying) {
		t.Error("expected errors.Is to find underlying error")
	}
}

func TestCodeError_Is_ErrCodeExecution(t *testing.T) {
	err := &CodeError{
		Message: "some code error",
		Line:    5,
	}

	if !errors.Is(err, ErrCodeExecution) {
		t.Error("expected CodeError to match ErrCodeExecution sentinel")
	}
}

func TestCodeError_Is_NotOtherSentinels(t *testing.T) {
	err := &CodeError{
		Message: "some code error",
		Line:    5,
	}

	if errors.Is(err, ErrConfiguration) {
		t.Error("CodeError should not match ErrConfiguration")
	}
	if errors.Is(err, ErrLimitExceeded) {
		t.Error("CodeError should not match ErrLimitExceeded")
	}
}
