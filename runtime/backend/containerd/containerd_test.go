package containerd

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
	if b.Kind() != runtime.BackendContainerd {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendContainerd)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.imageRef != "toolruntime-sandbox:latest" {
		t.Errorf("imageRef = %q, want %q", b.imageRef, "toolruntime-sandbox:latest")
	}
	if b.namespace != "default" {
		t.Errorf("namespace = %q, want %q", b.namespace, "default")
	}
	if b.socketPath != "/run/containerd/containerd.sock" {
		t.Errorf("socketPath = %q, want %q", b.socketPath, "/run/containerd/containerd.sock")
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
