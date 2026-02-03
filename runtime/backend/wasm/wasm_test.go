package wasm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

var minimalWasmModule = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

func TestBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.Backend = (*Backend)(nil)
}

func TestBackendKind(t *testing.T) {
	b := New(Config{})
	if b.Kind() != runtime.BackendWASM {
		t.Errorf("Kind() = %v, want %v", b.Kind(), runtime.BackendWASM)
	}
}

func TestBackendDefaults(t *testing.T) {
	b := New(Config{})
	if b.runtime != "wazero" {
		t.Errorf("runtime = %q, want %q", b.runtime, "wazero")
	}
	if b.maxMemoryPages != 256 {
		t.Errorf("maxMemoryPages = %d, want %d", b.maxMemoryPages, 256)
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

func TestBackendRequiresClient(t *testing.T) {
	b := New(Config{})
	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}
	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrClientNotConfigured) {
		t.Errorf("Execute() without client error = %v, want %v", err, ErrClientNotConfigured)
	}
}

func TestBackendExecuteSuccess(t *testing.T) {
	mockClient := &mockWasmRunner{
		result: Result{
			ExitCode: 0,
			Stdout:   "hello world",
			Stderr:   "",
			Duration: 100 * time.Millisecond,
		},
	}
	b := New(Config{
		Client:     mockClient,
		EnableWASI: true,
	})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:     "test code",
		Gateway:  &mockGateway{},
		Timeout:  5 * time.Second,
		Metadata: map[string]any{"wasm_module": minimalWasmModule},
	}

	result, err := b.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result.Stdout != "hello world" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello world")
	}
	if result.Backend.Kind != runtime.BackendWASM {
		t.Errorf("Backend.Kind = %v, want %v", result.Backend.Kind, runtime.BackendWASM)
	}
}

func TestBackendHealthCheckFailure(t *testing.T) {
	mockClient := &mockWasmRunner{}
	mockHealth := &mockHealthChecker{
		pingErr: errors.New("runtime not available"),
	}
	b := New(Config{
		Client:        mockClient,
		HealthChecker: mockHealth,
	})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:     "test",
		Gateway:  &mockGateway{},
		Metadata: map[string]any{"wasm_module": minimalWasmModule},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrWASMRuntimeNotAvailable) {
		t.Errorf("Execute() with health check failure error = %v, want %v", err, ErrWASMRuntimeNotAvailable)
	}
}

func TestBackendContextCancellation(t *testing.T) {
	mockClient := &mockWasmRunner{
		delay: 1 * time.Second,
	}
	b := New(Config{
		Client: mockClient,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := runtime.ExecuteRequest{
		Code:     "test",
		Gateway:  &mockGateway{},
		Metadata: map[string]any{"wasm_module": minimalWasmModule},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() with cancelled context error = %v, want %v", err, context.Canceled)
	}
}

func TestBackendBuildSpecProfiles(t *testing.T) {
	b := New(Config{
		EnableWASI:           true,
		AllowedHostFunctions: []string{"log"},
		MaxMemoryPages:       512,
	})

	tests := []struct {
		profile       runtime.SecurityProfile
		wantNetwork   bool
		wantClock     bool
		wantHostFuncs bool
	}{
		{runtime.ProfileDev, false, true, true},
		{runtime.ProfileStandard, false, true, true},
		{runtime.ProfileHardened, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			req := runtime.ExecuteRequest{
				Code:     "test",
				Gateway:  &mockGateway{},
				Metadata: map[string]any{"wasm_module": minimalWasmModule},
			}
			spec := b.buildSpec(req, tt.profile)

			if spec.Security.EnableNetwork != tt.wantNetwork {
				t.Errorf("EnableNetwork = %v, want %v", spec.Security.EnableNetwork, tt.wantNetwork)
			}
			if spec.Security.EnableClock != tt.wantClock {
				t.Errorf("EnableClock = %v, want %v", spec.Security.EnableClock, tt.wantClock)
			}
			hasHostFuncs := len(spec.Security.AllowedHostFunctions) > 0
			if hasHostFuncs != tt.wantHostFuncs {
				t.Errorf("HasHostFunctions = %v, want %v", hasHostFuncs, tt.wantHostFuncs)
			}
		})
	}
}

func TestBackendBuildSpecMemoryLimit(t *testing.T) {
	b := New(Config{
		MaxMemoryPages: 256,
	})

	req := runtime.ExecuteRequest{
		Code:     "test",
		Gateway:  &mockGateway{},
		Metadata: map[string]any{"wasm_module": minimalWasmModule},
		Limits: runtime.Limits{
			MemoryBytes: 32 * 1024 * 1024, // 32MB
		},
	}

	spec := b.buildSpec(req, runtime.ProfileStandard)

	// 32MB / 64KB per page = 512 pages
	expectedPages := uint32(512)
	if spec.Resources.MemoryPages != expectedPages {
		t.Errorf("MemoryPages = %d, want %d", spec.Resources.MemoryPages, expectedPages)
	}
}

func TestBackendMissingModule(t *testing.T) {
	mockClient := &mockWasmRunner{}
	b := New(Config{Client: mockClient})

	ctx := context.Background()
	req := runtime.ExecuteRequest{
		Code:    "test",
		Gateway: &mockGateway{},
	}

	_, err := b.Execute(ctx, req)
	if !errors.Is(err, ErrInvalidModule) {
		t.Errorf("Execute() missing module error = %v, want %v", err, ErrInvalidModule)
	}
}

// Mock implementations

type mockGateway struct{}

func (m *mockGateway) SearchTools(_ context.Context, _ string, _ int) ([]index.Summary, error) {
	return nil, nil //nolint:nilnil
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

// Ensure mockGateway implements ToolGateway
var _ runtime.ToolGateway = (*mockGateway)(nil)

type mockWasmRunner struct {
	result Result
	err    error
	delay  time.Duration
}

func (m *mockWasmRunner) Run(ctx context.Context, _ Spec) (Result, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.result, m.err
}

type mockHealthChecker struct {
	pingErr error
	info    RuntimeInfo
	infoErr error
}

func (m *mockHealthChecker) Ping(_ context.Context) error {
	return m.pingErr
}

func (m *mockHealthChecker) Info(_ context.Context) (RuntimeInfo, error) {
	return m.info, m.infoErr
}

// Interface compliance checks
var (
	_ Runner        = (*mockWasmRunner)(nil)
	_ HealthChecker = (*mockHealthChecker)(nil)
)
