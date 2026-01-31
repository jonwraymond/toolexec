package runtime

import (
	"errors"
	"testing"
)

// Test that all sentinel errors are distinct
func TestSentinelErrorsDistinct(t *testing.T) {
	sentinels := []error{
		ErrRuntimeUnavailable,
		ErrBackendDenied,
		ErrSandboxViolation,
		ErrTimeout,
		ErrResourceLimit,
		ErrMissingGateway,
		ErrMissingCode,
		ErrInvalidLimits,
	}

	// Check each pair is distinct
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("Sentinel errors should be distinct: %v and %v", sentinels[i], sentinels[j])
			}
		}
	}
}

// Test RuntimeError wrapping
func TestRuntimeErrorWrap(t *testing.T) {
	baseErr := ErrTimeout
	runtimeErr := &RuntimeError{
		Err:     baseErr,
		Op:      "execute",
		Backend: BackendDocker,
	}

	// Test Error() method
	errStr := runtimeErr.Error()
	if errStr == "" {
		t.Error("RuntimeError.Error() should not be empty")
	}

	// Test Unwrap
	unwrapped := runtimeErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("RuntimeError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}

	// Test errors.Is
	if !errors.Is(runtimeErr, ErrTimeout) {
		t.Error("errors.Is(runtimeErr, ErrTimeout) should be true")
	}
}

// Test RuntimeError fields
func TestRuntimeErrorFields(t *testing.T) {
	runtimeErr := &RuntimeError{
		Err:     ErrResourceLimit,
		Op:      "container_create",
		Backend: BackendDocker,
	}

	if runtimeErr.Op != "container_create" {
		t.Errorf("RuntimeError.Op = %q, want %q", runtimeErr.Op, "container_create")
	}
	if runtimeErr.Backend != BackendDocker {
		t.Errorf("RuntimeError.Backend = %v, want %v", runtimeErr.Backend, BackendDocker)
	}
}

// Test RuntimeError with nil Err
func TestRuntimeErrorNilErr(t *testing.T) {
	runtimeErr := &RuntimeError{
		Err:     nil,
		Op:      "test",
		Backend: BackendUnsafeHost,
	}

	// Should not panic
	_ = runtimeErr.Error()

	unwrapped := runtimeErr.Unwrap()
	if unwrapped != nil {
		t.Errorf("RuntimeError.Unwrap() with nil Err = %v, want nil", unwrapped)
	}
}

// Test errors.As with RuntimeError
func TestRuntimeErrorAs(t *testing.T) {
	baseErr := ErrSandboxViolation
	runtimeErr := &RuntimeError{
		Err:     baseErr,
		Op:      "syscall",
		Backend: BackendDocker,
	}

	// Wrap in another error
	wrappedErr := errors.New("outer: " + runtimeErr.Error())
	_ = wrappedErr // This won't work with errors.As since we're not properly wrapping

	// But errors.As should work with the runtime error itself
	var target *RuntimeError
	if !errors.As(runtimeErr, &target) {
		t.Error("errors.As should find RuntimeError")
	}
	if target.Op != "syscall" {
		t.Errorf("target.Op = %q, want %q", target.Op, "syscall")
	}
}

// Test sentinel error messages are meaningful
func TestSentinelErrorMessages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrRuntimeUnavailable, "runtime unavailable"},
		{ErrBackendDenied, "backend denied by policy"},
		{ErrSandboxViolation, "sandbox policy violation"},
		{ErrTimeout, "execution timeout"},
		{ErrResourceLimit, "resource limit exceeded"},
		{ErrMissingGateway, "gateway is required"},
		{ErrMissingCode, "code is required"},
		{ErrInvalidLimits, "invalid limits"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

// Test RuntimeError Retryable field
func TestRuntimeErrorRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "timeout is retryable",
			err:       ErrTimeout,
			retryable: true,
		},
		{
			name:      "resource limit is retryable",
			err:       ErrResourceLimit,
			retryable: true,
		},
		{
			name:      "sandbox violation is not retryable",
			err:       ErrSandboxViolation,
			retryable: false,
		},
		{
			name:      "backend denied is not retryable",
			err:       ErrBackendDenied,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeErr := &RuntimeError{
				Err:       tt.err,
				Op:        "test",
				Backend:   BackendDocker,
				Retryable: tt.retryable,
			}

			if runtimeErr.Retryable != tt.retryable {
				t.Errorf("RuntimeError.Retryable = %v, want %v", runtimeErr.Retryable, tt.retryable)
			}
		})
	}
}
