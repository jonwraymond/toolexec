package docker

import (
	"testing"
)

func TestContainerSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    ContainerSpec
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty image",
			spec:    ContainerSpec{},
			wantErr: true,
			errMsg:  "image is required",
		},
		{
			name: "valid minimal spec",
			spec: ContainerSpec{
				Image: "alpine:latest",
			},
			wantErr: false,
		},
		{
			name: "privileged container rejected",
			spec: ContainerSpec{
				Image: "alpine:latest",
				Security: SecuritySpec{
					Privileged: true,
				},
			},
			wantErr: true,
			errMsg:  "security policy violation",
		},
		{
			name: "host network rejected",
			spec: ContainerSpec{
				Image: "alpine:latest",
				Security: SecuritySpec{
					NetworkMode: "host",
				},
			},
			wantErr: true,
			errMsg:  "host network not allowed",
		},
		{
			name: "valid with all fields",
			spec: ContainerSpec{
				Image:      "alpine:latest",
				Command:    []string{"echo", "hello"},
				WorkingDir: "/app",
				Env:        []string{"FOO=bar"},
				Resources: ResourceSpec{
					MemoryBytes: 256 * 1024 * 1024,
					CPUQuota:    100000,
					PidsLimit:   100,
				},
				Security: SecuritySpec{
					User:           "nobody:nogroup",
					ReadOnlyRootfs: true,
					NetworkMode:    "none",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mount - missing target",
			spec: ContainerSpec{
				Image: "alpine:latest",
				Mounts: []Mount{
					{Type: MountTypeBind, Source: "/host"},
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestSecuritySpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    SecuritySpec
		wantErr bool
	}{
		{
			name:    "empty spec is valid",
			spec:    SecuritySpec{},
			wantErr: false,
		},
		{
			name: "privileged rejected",
			spec: SecuritySpec{
				Privileged: true,
			},
			wantErr: true,
		},
		{
			name: "host network rejected",
			spec: SecuritySpec{
				NetworkMode: "host",
			},
			wantErr: true,
		},
		{
			name: "none network allowed",
			spec: SecuritySpec{
				NetworkMode: "none",
			},
			wantErr: false,
		},
		{
			name: "bridge network allowed",
			spec: SecuritySpec{
				NetworkMode: "bridge",
			},
			wantErr: false,
		},
		{
			name: "full valid spec",
			spec: SecuritySpec{
				User:           "nobody:nogroup",
				ReadOnlyRootfs: true,
				NetworkMode:    "none",
				SeccompProfile: "/path/to/profile.json",
				Privileged:     false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourceSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    ResourceSpec
		wantErr bool
	}{
		{
			name:    "zero values valid",
			spec:    ResourceSpec{},
			wantErr: false,
		},
		{
			name: "positive values valid",
			spec: ResourceSpec{
				MemoryBytes: 256 * 1024 * 1024,
				CPUQuota:    100000,
				PidsLimit:   100,
				DiskBytes:   1024 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "negative memory rejected",
			spec: ResourceSpec{
				MemoryBytes: -1,
			},
			wantErr: true,
		},
		{
			name: "negative cpu quota rejected",
			spec: ResourceSpec{
				CPUQuota: -1,
			},
			wantErr: true,
		},
		{
			name: "negative pids limit rejected",
			spec: ResourceSpec{
				PidsLimit: -1,
			},
			wantErr: true,
		},
		{
			name: "negative disk rejected",
			spec: ResourceSpec{
				DiskBytes: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMountValidate(t *testing.T) {
	tests := []struct {
		name    string
		mount   Mount
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty mount",
			mount:   Mount{},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "missing type",
			mount: Mount{
				Target: "/container/path",
			},
			wantErr: true,
			errMsg:  "mount type is required",
		},
		{
			name: "bind mount without source",
			mount: Mount{
				Type:   MountTypeBind,
				Target: "/container/path",
			},
			wantErr: true,
			errMsg:  "source is required for bind mounts",
		},
		{
			name: "valid bind mount",
			mount: Mount{
				Type:   MountTypeBind,
				Source: "/host/path",
				Target: "/container/path",
			},
			wantErr: false,
		},
		{
			name: "valid bind mount read-only",
			mount: Mount{
				Type:     MountTypeBind,
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
			},
			wantErr: false,
		},
		{
			name: "volume mount without source",
			mount: Mount{
				Type:   MountTypeVolume,
				Target: "/container/path",
			},
			wantErr: true,
			errMsg:  "source is required for volume mounts",
		},
		{
			name: "valid volume mount",
			mount: Mount{
				Type:   MountTypeVolume,
				Source: "my-volume",
				Target: "/container/path",
			},
			wantErr: false,
		},
		{
			name: "tmpfs mount no source needed",
			mount: Mount{
				Type:   MountTypeTmpfs,
				Target: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "unknown mount type",
			mount: Mount{
				Type:   "unknown",
				Target: "/container/path",
			},
			wantErr: true,
			errMsg:  "unknown mount type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mount.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMountTypes(t *testing.T) {
	// Verify mount type constants
	if MountTypeBind != "bind" {
		t.Errorf("MountTypeBind = %q, want %q", MountTypeBind, "bind")
	}
	if MountTypeVolume != "volume" {
		t.Errorf("MountTypeVolume = %q, want %q", MountTypeVolume, "volume")
	}
	if MountTypeTmpfs != "tmpfs" {
		t.Errorf("MountTypeTmpfs = %q, want %q", MountTypeTmpfs, "tmpfs")
	}
}

func TestStreamEventTypes(t *testing.T) {
	// Verify stream event type constants
	if StreamEventStdout != "stdout" {
		t.Errorf("StreamEventStdout = %q, want %q", StreamEventStdout, "stdout")
	}
	if StreamEventStderr != "stderr" {
		t.Errorf("StreamEventStderr = %q, want %q", StreamEventStderr, "stderr")
	}
	if StreamEventExit != "exit" {
		t.Errorf("StreamEventExit = %q, want %q", StreamEventExit, "exit")
	}
	if StreamEventError != "error" {
		t.Errorf("StreamEventError = %q, want %q", StreamEventError, "error")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
