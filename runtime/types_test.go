package runtime

import (
	"errors"
	"testing"
	"time"
)

// Test SecurityProfile constants
func TestSecurityProfileConstants(t *testing.T) {
	tests := []struct {
		profile SecurityProfile
		want    string
	}{
		{ProfileDev, "dev"},
		{ProfileStandard, "standard"},
		{ProfileHardened, "hardened"},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			if string(tt.profile) != tt.want {
				t.Errorf("SecurityProfile = %q, want %q", tt.profile, tt.want)
			}
		})
	}
}

func TestSecurityProfileIsValid(t *testing.T) {
	tests := []struct {
		profile SecurityProfile
		valid   bool
	}{
		{ProfileDev, true},
		{ProfileStandard, true},
		{ProfileHardened, true},
		{SecurityProfile("unknown"), false},
		{SecurityProfile(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			got := tt.profile.IsValid()
			if got != tt.valid {
				t.Errorf("SecurityProfile(%q).IsValid() = %v, want %v", tt.profile, got, tt.valid)
			}
		})
	}
}

// Test BackendKind constants
func TestBackendKindConstants(t *testing.T) {
	tests := []struct {
		kind BackendKind
		want string
	}{
		{BackendUnsafeHost, "unsafe_host"},
		{BackendDocker, "docker"},
		{BackendContainerd, "containerd"},
		{BackendKubernetes, "kubernetes"},
		{BackendGVisor, "gvisor"},
		{BackendKata, "kata"},
		{BackendFirecracker, "firecracker"},
		{BackendWASM, "wasm"},
		{BackendTemporal, "temporal"},
		{BackendRemote, "remote"},
		{BackendProxmoxLXC, "proxmox_lxc"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if string(tt.kind) != tt.want {
				t.Errorf("BackendKind = %q, want %q", tt.kind, tt.want)
			}
		})
	}
}

// Test LimitsEnforced struct
func TestLimitsEnforced(t *testing.T) {
	// Test that zero value means nothing enforced
	var noLimits LimitsEnforced
	if noLimits.Timeout || noLimits.ToolCalls || noLimits.Memory {
		t.Error("Zero value LimitsEnforced should have all false")
	}

	// Test fully enforced
	allEnforced := LimitsEnforced{
		Timeout:    true,
		ToolCalls:  true,
		ChainSteps: true,
		Memory:     true,
		CPU:        true,
		Pids:       true,
		Disk:       true,
	}

	if !allEnforced.Timeout || !allEnforced.Memory || !allEnforced.CPU {
		t.Error("All fields should be true when set")
	}
}

// Test Limits validation
func TestLimitsValidate(t *testing.T) {
	tests := []struct {
		name    string
		limits  Limits
		wantErr bool
	}{
		{
			name:    "zero values valid (unlimited)",
			limits:  Limits{},
			wantErr: false,
		},
		{
			name: "positive values valid",
			limits: Limits{
				MaxToolCalls:   10,
				MaxChainSteps:  5,
				CPUQuotaMillis: 1000,
				MemoryBytes:    1024 * 1024 * 100,
				PidsMax:        100,
				DiskBytes:      1024 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name:    "negative MaxToolCalls invalid",
			limits:  Limits{MaxToolCalls: -1},
			wantErr: true,
		},
		{
			name:    "negative MaxChainSteps invalid",
			limits:  Limits{MaxChainSteps: -1},
			wantErr: true,
		},
		{
			name:    "negative CPUQuotaMillis invalid",
			limits:  Limits{CPUQuotaMillis: -1},
			wantErr: true,
		},
		{
			name:    "negative MemoryBytes invalid",
			limits:  Limits{MemoryBytes: -1},
			wantErr: true,
		},
		{
			name:    "negative PidsMax invalid",
			limits:  Limits{PidsMax: -1},
			wantErr: true,
		},
		{
			name:    "negative DiskBytes invalid",
			limits:  Limits{DiskBytes: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.limits.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Limits.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test ExecuteRequest validation
func TestExecuteRequestValidate(t *testing.T) {
	// Create a minimal mock gateway for valid requests
	mockGateway := &mockToolGateway{}

	tests := []struct {
		name    string
		req     ExecuteRequest
		wantErr error
	}{
		{
			name: "valid minimal request",
			req: ExecuteRequest{
				Code:    "print('hello')",
				Gateway: mockGateway,
			},
			wantErr: nil,
		},
		{
			name: "valid full request",
			req: ExecuteRequest{
				Language: "go",
				Code:     "fmt.Println(\"hello\")",
				Timeout:  30 * time.Second,
				Limits: Limits{
					MaxToolCalls: 10,
				},
				Profile:  ProfileStandard,
				Gateway:  mockGateway,
				Metadata: map[string]any{"key": "value"},
			},
			wantErr: nil,
		},
		{
			name: "missing gateway",
			req: ExecuteRequest{
				Code: "print('hello')",
			},
			wantErr: ErrMissingGateway,
		},
		{
			name: "missing code",
			req: ExecuteRequest{
				Gateway: mockGateway,
			},
			wantErr: ErrMissingCode,
		},
		{
			name: "invalid limits",
			req: ExecuteRequest{
				Code:    "print('hello')",
				Gateway: mockGateway,
				Limits:  Limits{MaxToolCalls: -1},
			},
			wantErr: ErrInvalidLimits,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ExecuteRequest.Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ExecuteRequest.Validate() expected error %v, got nil", tt.wantErr)
				} else if !errorIs(err, tt.wantErr) {
					t.Errorf("ExecuteRequest.Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}

// Test ToolCallRecord
func TestToolCallRecord(t *testing.T) {
	record := ToolCallRecord{
		ToolID:      "namespace:tool",
		BackendKind: "mcp",
		Duration:    100 * time.Millisecond,
		ErrorOp:     "",
	}

	if record.ToolID != "namespace:tool" {
		t.Errorf("ToolCallRecord.ToolID = %q, want %q", record.ToolID, "namespace:tool")
	}
	if record.BackendKind != "mcp" {
		t.Errorf("ToolCallRecord.BackendKind = %q, want %q", record.BackendKind, "mcp")
	}
	if record.Duration != 100*time.Millisecond {
		t.Errorf("ToolCallRecord.Duration = %v, want %v", record.Duration, 100*time.Millisecond)
	}
}

// Test ExecuteResult
func TestExecuteResult(t *testing.T) {
	result := ExecuteResult{
		Value:  "output",
		Stdout: "stdout content",
		Stderr: "stderr content",
		ToolCalls: []ToolCallRecord{
			{ToolID: "test:tool", Duration: 50 * time.Millisecond},
		},
		Duration: 200 * time.Millisecond,
		Backend: BackendInfo{
			Kind:      BackendDocker,
			Readiness: ReadinessProd,
			Details:   map[string]any{"container": "abc123"},
		},
	}

	if result.Value != "output" {
		t.Errorf("ExecuteResult.Value = %v, want %v", result.Value, "output")
	}
	if result.Stdout != "stdout content" {
		t.Errorf("ExecuteResult.Stdout = %q, want %q", result.Stdout, "stdout content")
	}
	if result.Stderr != "stderr content" {
		t.Errorf("ExecuteResult.Stderr = %q, want %q", result.Stderr, "stderr content")
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("len(ExecuteResult.ToolCalls) = %d, want %d", len(result.ToolCalls), 1)
	}
	if result.Duration != 200*time.Millisecond {
		t.Errorf("ExecuteResult.Duration = %v, want %v", result.Duration, 200*time.Millisecond)
	}
	if result.Backend.Kind != BackendDocker {
		t.Errorf("ExecuteResult.Backend.Kind = %v, want %v", result.Backend.Kind, BackendDocker)
	}
}

// Test BackendInfo
func TestBackendInfo(t *testing.T) {
	info := BackendInfo{
		Kind:      BackendUnsafeHost,
		Readiness: ReadinessProd,
		Details: map[string]any{
			"pid": 12345,
		},
	}

	if info.Kind != BackendUnsafeHost {
		t.Errorf("BackendInfo.Kind = %v, want %v", info.Kind, BackendUnsafeHost)
	}
	if info.Readiness != ReadinessProd {
		t.Errorf("BackendInfo.Readiness = %v, want %v", info.Readiness, ReadinessProd)
	}
	if info.Details["pid"] != 12345 {
		t.Errorf("BackendInfo.Details[\"pid\"] = %v, want %v", info.Details["pid"], 12345)
	}
}

// errorIs is a helper that checks if err wraps or matches target
func errorIs(err, target error) bool {
	return errors.Is(err, target)
}
