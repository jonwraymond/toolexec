package firecracker

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/toolexec/runtime"
)

func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})
	if b.Kind() != runtime.BackendFirecracker {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendFirecracker)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.binaryPath != "firecracker" {
		t.Errorf("binaryPath = %q, want %q", b.binaryPath, "firecracker")
	}
	if b.vcpuCount != 1 {
		t.Errorf("vcpuCount = %d, want %d", b.vcpuCount, 1)
	}
	if b.memSizeMB != 128 {
		t.Errorf("memSizeMB = %d, want %d", b.memSizeMB, 128)
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: nil,
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingGateway) {
		t.Errorf("Execute() without gateway error = %v, want %v", err, runtime.ErrMissingGateway)
	}
}
