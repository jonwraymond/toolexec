package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// RuntimeContract defines tests that any Runtime implementation must pass.
type RuntimeContract struct {
	// NewRuntime creates a fresh runtime instance for testing.
	NewRuntime func() Runtime

	// NewGateway creates a gateway for testing.
	NewGateway func() ToolGateway
}

// RunRuntimeContractTests runs all contract tests for a Runtime implementation.
func RunRuntimeContractTests(t *testing.T, contract RuntimeContract) {
	t.Helper()

	t.Run("Execute", func(t *testing.T) {
		t.Run("validates request", func(t *testing.T) {
			rt := contract.NewRuntime()
			ctx := context.Background()

			// Missing gateway
			req := ExecuteRequest{
				Code: "test",
			}

			_, err := rt.Execute(ctx, req)
			if !errors.Is(err, ErrMissingGateway) {
				t.Errorf("Execute() without gateway error = %v, want %v", err, ErrMissingGateway)
			}
		})

		t.Run("propagates context", func(t *testing.T) {
			rt := contract.NewRuntime()
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			req := ExecuteRequest{
				Code:    "test",
				Gateway: contract.NewGateway(),
			}

			_, err := rt.Execute(ctx, req)
			// Should return error (context canceled)
			if err == nil {
				t.Error("Execute() with canceled context should return error")
			}
		})
	})
}

// Test DefaultRuntime implementation
func TestDefaultRuntimeBackendSelection(t *testing.T) {
	devBackend := &mockBackend{
		kind:   BackendUnsafeHost,
		result: ExecuteResult{Value: "dev"},
	}
	standardBackend := &mockBackend{
		kind:   BackendDocker,
		result: ExecuteResult{Value: "standard"},
	}

	rt := NewDefaultRuntime(RuntimeConfig{
		Backends: map[SecurityProfile]Backend{
			ProfileDev:      devBackend,
			ProfileStandard: standardBackend,
		},
	})

	ctx := context.Background()
	gw := &mockToolGateway{}

	tests := []struct {
		name      string
		profile   SecurityProfile
		wantValue any
	}{
		{
			name:      "dev profile uses dev backend",
			profile:   ProfileDev,
			wantValue: "dev",
		},
		{
			name:      "standard profile uses standard backend",
			profile:   ProfileStandard,
			wantValue: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ExecuteRequest{
				Code:    "test",
				Gateway: gw,
				Profile: tt.profile,
			}

			result, err := rt.Execute(ctx, req)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.Value != tt.wantValue {
				t.Errorf("Execute().Value = %v, want %v", result.Value, tt.wantValue)
			}
		})
	}
}

func TestDefaultRuntimeDenyUnsafe(t *testing.T) {
	devBackend := &mockBackend{
		kind:   BackendUnsafeHost,
		result: ExecuteResult{Value: "dev"},
	}

	rt := NewDefaultRuntime(RuntimeConfig{
		Backends: map[SecurityProfile]Backend{
			ProfileDev: devBackend,
		},
		DenyUnsafeProfiles: []SecurityProfile{ProfileStandard, ProfileHardened},
		DefaultProfile:     ProfileDev,
	})

	ctx := context.Background()
	gw := &mockToolGateway{}

	// Requesting standard profile should fail because only dev is available
	// and dev is denied for standard/hardened
	req := ExecuteRequest{
		Code:    "test",
		Gateway: gw,
		Profile: ProfileStandard,
	}

	_, err := rt.Execute(ctx, req)
	if !errors.Is(err, ErrBackendDenied) && !errors.Is(err, ErrRuntimeUnavailable) {
		t.Errorf("Execute() with denied profile error = %v, want ErrBackendDenied or ErrRuntimeUnavailable", err)
	}
}

func TestDefaultRuntimeThreadSafety(t *testing.T) {
	backend := &mockBackend{
		kind:   BackendUnsafeHost,
		result: ExecuteResult{Value: "ok"},
	}

	rt := NewDefaultRuntime(RuntimeConfig{
		Backends: map[SecurityProfile]Backend{
			ProfileDev: backend,
		},
		DefaultProfile: ProfileDev,
	})

	ctx := context.Background()
	gw := &mockToolGateway{}

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	// Run concurrent executions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := ExecuteRequest{
				Code:    "test",
				Gateway: gw,
			}
			_, err := rt.Execute(ctx, req)
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Execute() error = %v", err)
	}
}

func TestDefaultRuntimeMissingBackend(t *testing.T) {
	rt := NewDefaultRuntime(RuntimeConfig{
		Backends: map[SecurityProfile]Backend{
			ProfileDev: &mockBackend{kind: BackendUnsafeHost},
		},
	})

	ctx := context.Background()
	gw := &mockToolGateway{}

	// Request hardened profile which has no backend
	req := ExecuteRequest{
		Code:    "test",
		Gateway: gw,
		Profile: ProfileHardened,
	}

	_, err := rt.Execute(ctx, req)
	if !errors.Is(err, ErrRuntimeUnavailable) {
		t.Errorf("Execute() with missing backend error = %v, want %v", err, ErrRuntimeUnavailable)
	}
}

func TestDefaultRuntimeDefaultProfile(t *testing.T) {
	backend := &mockBackend{
		kind:   BackendUnsafeHost,
		result: ExecuteResult{Value: "ok"},
	}

	rt := NewDefaultRuntime(RuntimeConfig{
		Backends: map[SecurityProfile]Backend{
			ProfileDev: backend,
		},
		DefaultProfile: ProfileDev,
	})

	ctx := context.Background()
	gw := &mockToolGateway{}

	// Request without profile should use default
	req := ExecuteRequest{
		Code:    "test",
		Gateway: gw,
		// No profile specified
	}

	result, err := rt.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Value != "ok" {
		t.Errorf("Execute().Value = %v, want %v", result.Value, "ok")
	}
}

// Test Runtime interface satisfaction
func TestDefaultRuntimeImplementsInterface(t *testing.T) {
	t.Helper()
	var _ Runtime = (*DefaultRuntime)(nil)
}

// Test contract for DefaultRuntime
func TestDefaultRuntimeContract(t *testing.T) {
	RunRuntimeContractTests(t, RuntimeContract{
		NewRuntime: func() Runtime {
			return NewDefaultRuntime(RuntimeConfig{
				Backends: map[SecurityProfile]Backend{
					ProfileDev: &mockBackend{
						kind:   BackendUnsafeHost,
						result: ExecuteResult{Value: "ok"},
					},
				},
				DefaultProfile: ProfileDev,
			})
		},
		NewGateway: func() ToolGateway {
			return &mockToolGateway{}
		},
	})
}
