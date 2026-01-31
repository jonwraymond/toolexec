package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// mockGateway implements runtime.ToolGateway for testing
type mockGateway struct{}

func (m *mockGateway) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, nil
}

func (m *mockGateway) ListNamespaces(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockGateway) DescribeTool(_ context.Context, _ string, _ tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	return tooldoc.ToolDoc{}, nil
}

func (m *mockGateway) ListToolExamples(_ context.Context, _ string, _ int) ([]tooldoc.ToolExample, error) {
	return nil, nil
}

func (m *mockGateway) RunTool(_ context.Context, _ string, _ map[string]any) (run.RunResult, error) {
	return run.RunResult{}, nil
}

func (m *mockGateway) RunChain(_ context.Context, _ []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	return run.RunResult{}, nil, nil
}

// TestBackendImplementsInterface verifies Backend satisfies runtime.Backend
func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})

	if b.Kind() != runtime.BackendDocker {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendDocker)
	}
}

func TestBackendRequiresGateway(t *testing.T) {
	b := New(Config{})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: nil,
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingGateway) {
		t.Errorf("Execute() without gateway error = %v, want %v", err, runtime.ErrMissingGateway)
	}
}

func TestBackendRequiresCode(t *testing.T) {
	b := New(Config{})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, runtime.ErrMissingCode) {
		t.Errorf("Execute() without code error = %v, want %v", err, runtime.ErrMissingCode)
	}
}

func TestBackendContractCompliance(t *testing.T) {
	runtime.RunBackendContractTests(t, runtime.BackendContract{
		NewBackend: func() runtime.Backend {
			return New(Config{})
		},
		NewGateway: func() runtime.ToolGateway {
			return &mockGateway{}
		},
		ExpectedKind:       runtime.BackendDocker,
		SkipExecutionTests: true, // Docker may not be available
	})
}

func TestBackendProfileRestrictions(t *testing.T) {
	b := New(Config{})

	tests := []struct {
		profile        runtime.SecurityProfile
		expectNetwork  bool
		expectReadOnly bool
	}{
		{runtime.ProfileStandard, false, true},
		{runtime.ProfileHardened, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			// This is a design validation test - actual Docker restrictions
			// would be tested in integration tests
			opts := b.containerOptions(tt.profile, runtime.Limits{})

			if opts.NetworkDisabled != !tt.expectNetwork {
				t.Errorf("profile %v NetworkDisabled = %v, want %v",
					tt.profile, opts.NetworkDisabled, !tt.expectNetwork)
			}
			if opts.ReadOnlyRootfs != tt.expectReadOnly {
				t.Errorf("profile %v ReadOnlyRootfs = %v, want %v",
					tt.profile, opts.ReadOnlyRootfs, tt.expectReadOnly)
			}
		})
	}
}

func TestBackendResourceLimits(t *testing.T) {
	b := New(Config{})

	limits := runtime.Limits{
		MemoryBytes:    256 * 1024 * 1024, // 256MB
		CPUQuotaMillis: 1000,              // 1 CPU
		PidsMax:        100,
	}

	opts := b.containerOptions(runtime.ProfileStandard, limits)

	if opts.MemoryLimit != limits.MemoryBytes {
		t.Errorf("MemoryLimit = %d, want %d", opts.MemoryLimit, limits.MemoryBytes)
	}
	if opts.CPUQuota != limits.CPUQuotaMillis*1000 {
		t.Errorf("CPUQuota = %d, want %d", opts.CPUQuota, limits.CPUQuotaMillis*1000)
	}
	if opts.PidsLimit != limits.PidsMax {
		t.Errorf("PidsLimit = %d, want %d", opts.PidsLimit, limits.PidsMax)
	}
}

func TestBackendRequiresClient(t *testing.T) {
	b := New(Config{}) // No client configured

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrClientNotConfigured) {
		t.Errorf("Execute() without client error = %v, want %v", err, ErrClientNotConfigured)
	}
}

func TestBackendWithMockClient(t *testing.T) {
	mockRunner := &MockContainerRunner{
		RunFunc: func(_ context.Context, spec ContainerSpec) (ContainerResult, error) {
			// Verify spec is built correctly
			if spec.Image != "toolruntime-sandbox:latest" {
				t.Errorf("spec.Image = %q, want %q", spec.Image, "toolruntime-sandbox:latest")
			}
			return ContainerResult{
				ExitCode: 0,
				Stdout:   "hello world",
				Stderr:   "",
			}, nil
		},
	}

	b := New(Config{
		Client: mockRunner,
	})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: &mockGateway{},
	}

	result, err := b.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Stdout != "hello world" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello world")
	}
}

func TestBackendWithHealthChecker(t *testing.T) {
	t.Run("healthy daemon", func(t *testing.T) {
		mockRunner := &MockContainerRunner{
			RunFunc: func(_ context.Context, _ ContainerSpec) (ContainerResult, error) {
				return ContainerResult{ExitCode: 0}, nil
			},
		}
		mockHealth := &MockHealthChecker{
			PingFunc: func(_ context.Context) error {
				return nil
			},
		}

		b := New(Config{
			Client:        mockRunner,
			HealthChecker: mockHealth,
		})

		ctx := context.Background()
		req := runtime.ExecuteRequest{
			Code:    "print('hello')",
			Gateway: &mockGateway{},
		}

		_, err := b.Execute(ctx, req)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})

	t.Run("unhealthy daemon", func(t *testing.T) {
		mockRunner := &MockContainerRunner{}
		mockHealth := &MockHealthChecker{
			PingFunc: func(_ context.Context) error {
				return errors.New("connection refused")
			},
		}

		b := New(Config{
			Client:        mockRunner,
			HealthChecker: mockHealth,
		})

		ctx := context.Background()
		req := runtime.ExecuteRequest{
			Code:    "print('hello')",
			Gateway: &mockGateway{},
		}

		_, err := b.Execute(ctx, req)
		if !errors.Is(err, ErrDaemonUnavailable) {
			t.Errorf("Execute() error = %v, want %v", err, ErrDaemonUnavailable)
		}
	})
}

func TestBackendWithImageResolver(t *testing.T) {
	resolvedImage := ""
	mockRunner := &MockContainerRunner{
		RunFunc: func(_ context.Context, spec ContainerSpec) (ContainerResult, error) {
			resolvedImage = spec.Image
			return ContainerResult{ExitCode: 0}, nil
		},
	}
	mockResolver := &MockImageResolver{
		ResolveFunc: func(_ context.Context, image string) (string, error) {
			return image + "@sha256:abc123", nil
		},
	}

	b := New(Config{
		ImageName:     "my-image:v1",
		Client:        mockRunner,
		ImageResolver: mockResolver,
	})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resolvedImage != "my-image:v1@sha256:abc123" {
		t.Errorf("resolved image = %q, want %q", resolvedImage, "my-image:v1@sha256:abc123")
	}
}

func TestBackendBuildSpec(t *testing.T) {
	b := New(Config{
		SeccompPath: "/path/to/seccomp.json",
	})

	req := runtime.ExecuteRequest{
		Code:    "print('hello')",
		Gateway: &mockGateway{},
		Profile: runtime.ProfileHardened,
		Limits: runtime.Limits{
			MemoryBytes:    256 * 1024 * 1024,
			CPUQuotaMillis: 1000,
			PidsMax:        100,
		},
	}

	spec, err := b.buildSpec("test-image:latest", req, runtime.ProfileHardened)
	if err != nil {
		t.Fatalf("buildSpec() error = %v", err)
	}

	// Verify image
	if spec.Image != "test-image:latest" {
		t.Errorf("Image = %q, want %q", spec.Image, "test-image:latest")
	}

	// Verify security
	if spec.Security.NetworkMode != "none" {
		t.Errorf("Security.NetworkMode = %q, want %q", spec.Security.NetworkMode, "none")
	}
	if !spec.Security.ReadOnlyRootfs {
		t.Error("Security.ReadOnlyRootfs = false, want true")
	}
	if spec.Security.SeccompProfile != "/path/to/seccomp.json" {
		t.Errorf("Security.SeccompProfile = %q, want %q", spec.Security.SeccompProfile, "/path/to/seccomp.json")
	}

	// Verify resources
	if spec.Resources.MemoryBytes != 256*1024*1024 {
		t.Errorf("Resources.MemoryBytes = %d, want %d", spec.Resources.MemoryBytes, 256*1024*1024)
	}
	if spec.Resources.CPUQuota != 1000*1000 { // milliseconds to microseconds
		t.Errorf("Resources.CPUQuota = %d, want %d", spec.Resources.CPUQuota, 1000*1000)
	}
	if spec.Resources.PidsLimit != 100 {
		t.Errorf("Resources.PidsLimit = %d, want %d", spec.Resources.PidsLimit, 100)
	}

	// Verify labels
	if spec.Labels["runtime.profile"] != "hardened" {
		t.Errorf("Labels[runtime.profile] = %q, want %q", spec.Labels["runtime.profile"], "hardened")
	}
	if spec.Labels["runtime.backend"] != "docker" {
		t.Errorf("Labels[runtime.backend] = %q, want %q", spec.Labels["runtime.backend"], "docker")
	}
}

func TestClientError(t *testing.T) {
	t.Run("with container ID", func(t *testing.T) {
		err := &ClientError{
			Op:          "start",
			Image:       "alpine:latest",
			ContainerID: "abc123",
			Err:         errors.New("permission denied"),
		}
		expected := "docker start alpine:latest (abc123): permission denied"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("without container ID", func(t *testing.T) {
		err := &ClientError{
			Op:    "pull",
			Image: "alpine:latest",
			Err:   errors.New("not found"),
		}
		expected := "docker pull alpine:latest: not found"
		if err.Error() != expected {
			t.Errorf("Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		innerErr := errors.New("inner error")
		err := &ClientError{
			Op:    "create",
			Image: "alpine:latest",
			Err:   innerErr,
		}
		if !errors.Is(err, innerErr) {
			t.Error("Unwrap() should return inner error")
		}
	})
}
