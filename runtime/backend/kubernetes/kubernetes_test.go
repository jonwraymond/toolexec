package kubernetes

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
	if b.Kind() != runtime.BackendKubernetes {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendKubernetes)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.namespace != "default" {
		t.Errorf("namespace = %q, want %q", b.namespace, "default")
	}
	if b.image != "toolruntime-sandbox:latest" {
		t.Errorf("image = %q, want %q", b.image, "toolruntime-sandbox:latest")
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
