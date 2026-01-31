package docker

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockContainerRunner is a test double for ContainerRunner.
type MockContainerRunner struct {
	RunFunc func(ctx context.Context, spec ContainerSpec) (ContainerResult, error)
}

func (m *MockContainerRunner) Run(ctx context.Context, spec ContainerSpec) (ContainerResult, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, spec)
	}
	return ContainerResult{}, nil
}

// MockImageResolver is a test double for ImageResolver.
type MockImageResolver struct {
	ResolveFunc func(ctx context.Context, image string) (string, error)
}

func (m *MockImageResolver) Resolve(ctx context.Context, image string) (string, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(ctx, image)
	}
	return image, nil
}

// MockHealthChecker is a test double for HealthChecker.
type MockHealthChecker struct {
	PingFunc func(ctx context.Context) error
	InfoFunc func(ctx context.Context) (DaemonInfo, error)
}

func (m *MockHealthChecker) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func (m *MockHealthChecker) Info(ctx context.Context) (DaemonInfo, error) {
	if m.InfoFunc != nil {
		return m.InfoFunc(ctx)
	}
	return DaemonInfo{}, nil
}

// ContainerRunnerContract defines contract tests for ContainerRunner implementations.
type ContainerRunnerContract struct {
	NewRunner func() ContainerRunner
}

// RunContainerRunnerContractTests runs contract tests against a ContainerRunner implementation.
func RunContainerRunnerContractTests(t *testing.T, contract ContainerRunnerContract) {
	t.Helper()

	t.Run("Run", func(t *testing.T) {
		t.Run("requires valid spec", func(t *testing.T) {
			runner := contract.NewRunner()
			ctx := context.Background()

			// Empty image should fail
			_, err := runner.Run(ctx, ContainerSpec{})
			if err == nil {
				t.Error("Run() with empty spec should return error")
			}
		})

		t.Run("respects context cancellation", func(t *testing.T) {
			runner := contract.NewRunner()
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			spec := ContainerSpec{Image: "alpine:latest"}
			_, err := runner.Run(ctx, spec)
			if err == nil {
				t.Error("Run() with cancelled context should return error")
			}
		})
	})
}

func TestMockContainerRunner(t *testing.T) {
	t.Run("default returns empty result", func(t *testing.T) {
		runner := &MockContainerRunner{}
		result, err := runner.Run(context.Background(), ContainerSpec{Image: "test"})
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("ExitCode = %d, want 0", result.ExitCode)
		}
	})

	t.Run("RunFunc is called", func(t *testing.T) {
		called := false
		runner := &MockContainerRunner{
			RunFunc: func(_ context.Context, spec ContainerSpec) (ContainerResult, error) {
				called = true
				if spec.Image != "test:latest" {
					t.Errorf("spec.Image = %q, want %q", spec.Image, "test:latest")
				}
				return ContainerResult{
					ExitCode: 0,
					Stdout:   "hello world",
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		result, err := runner.Run(context.Background(), ContainerSpec{Image: "test:latest"})
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
		if !called {
			t.Error("RunFunc was not called")
		}
		if result.Stdout != "hello world" {
			t.Errorf("Stdout = %q, want %q", result.Stdout, "hello world")
		}
	})

	t.Run("RunFunc can return error", func(t *testing.T) {
		expectedErr := errors.New("container failed")
		runner := &MockContainerRunner{
			RunFunc: func(_ context.Context, _ ContainerSpec) (ContainerResult, error) {
				return ContainerResult{}, expectedErr
			},
		}

		_, err := runner.Run(context.Background(), ContainerSpec{Image: "test"})
		if !errors.Is(err, expectedErr) {
			t.Errorf("Run() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockImageResolver(t *testing.T) {
	t.Run("default returns input image", func(t *testing.T) {
		resolver := &MockImageResolver{}
		resolved, err := resolver.Resolve(context.Background(), "alpine:latest")
		if err != nil {
			t.Errorf("Resolve() error = %v", err)
		}
		if resolved != "alpine:latest" {
			t.Errorf("Resolve() = %q, want %q", resolved, "alpine:latest")
		}
	})

	t.Run("ResolveFunc is called", func(t *testing.T) {
		resolver := &MockImageResolver{
			ResolveFunc: func(_ context.Context, image string) (string, error) {
				return image + "@sha256:abc123", nil
			},
		}

		resolved, err := resolver.Resolve(context.Background(), "alpine:latest")
		if err != nil {
			t.Errorf("Resolve() error = %v", err)
		}
		if resolved != "alpine:latest@sha256:abc123" {
			t.Errorf("Resolve() = %q, want %q", resolved, "alpine:latest@sha256:abc123")
		}
	})
}

func TestMockHealthChecker(t *testing.T) {
	t.Run("default Ping succeeds", func(t *testing.T) {
		checker := &MockHealthChecker{}
		err := checker.Ping(context.Background())
		if err != nil {
			t.Errorf("Ping() error = %v", err)
		}
	})

	t.Run("PingFunc is called", func(t *testing.T) {
		called := false
		checker := &MockHealthChecker{
			PingFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := checker.Ping(context.Background())
		if err != nil {
			t.Errorf("Ping() error = %v", err)
		}
		if !called {
			t.Error("PingFunc was not called")
		}
	})

	t.Run("default Info returns empty", func(t *testing.T) {
		checker := &MockHealthChecker{}
		info, err := checker.Info(context.Background())
		if err != nil {
			t.Errorf("Info() error = %v", err)
		}
		if info.Version != "" {
			t.Errorf("Version = %q, want empty", info.Version)
		}
	})

	t.Run("InfoFunc is called", func(t *testing.T) {
		checker := &MockHealthChecker{
			InfoFunc: func(_ context.Context) (DaemonInfo, error) {
				return DaemonInfo{
					Version:      "24.0.0",
					APIVersion:   "1.43",
					OS:           "linux",
					Architecture: "amd64",
				}, nil
			},
		}

		info, err := checker.Info(context.Background())
		if err != nil {
			t.Errorf("Info() error = %v", err)
		}
		if info.Version != "24.0.0" {
			t.Errorf("Version = %q, want %q", info.Version, "24.0.0")
		}
	})
}

// Verify interfaces are satisfied at compile time
var (
	_ ContainerRunner = (*MockContainerRunner)(nil)
	_ ImageResolver   = (*MockImageResolver)(nil)
	_ HealthChecker   = (*MockHealthChecker)(nil)
)
