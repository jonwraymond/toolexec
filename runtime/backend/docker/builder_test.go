package docker

import (
	"testing"
	"time"
)

func TestSpecBuilder(t *testing.T) {
	t.Run("minimal build", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Image != "alpine:latest" {
			t.Errorf("Image = %q, want %q", spec.Image, "alpine:latest")
		}
	})

	t.Run("empty image fails", func(t *testing.T) {
		_, err := NewSpecBuilder("").Build()
		if err == nil {
			t.Error("Build() with empty image should fail")
		}
	})

	t.Run("with command", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithCommand("echo", "hello", "world").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Command) != 3 {
			t.Errorf("Command length = %d, want 3", len(spec.Command))
		}
		if spec.Command[0] != "echo" {
			t.Errorf("Command[0] = %q, want %q", spec.Command[0], "echo")
		}
	})

	t.Run("with working dir", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithWorkingDir("/app").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.WorkingDir != "/app" {
			t.Errorf("WorkingDir = %q, want %q", spec.WorkingDir, "/app")
		}
	})

	t.Run("with env", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithEnv("FOO", "bar").
			WithEnv("BAZ", "qux").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Env) != 2 {
			t.Errorf("Env length = %d, want 2", len(spec.Env))
		}
		if spec.Env[0] != "FOO=bar" {
			t.Errorf("Env[0] = %q, want %q", spec.Env[0], "FOO=bar")
		}
	})

	t.Run("with envs slice", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithEnvs([]string{"A=1", "B=2"}).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Env) != 2 {
			t.Errorf("Env length = %d, want 2", len(spec.Env))
		}
	})

	t.Run("with mount", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithMount(Mount{
				Type:   MountTypeBind,
				Source: "/host",
				Target: "/container",
			}).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Mounts) != 1 {
			t.Errorf("Mounts length = %d, want 1", len(spec.Mounts))
		}
	})

	t.Run("with bind mount helper", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithBindMount("/host", "/container", true).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Mounts) != 1 {
			t.Fatalf("Mounts length = %d, want 1", len(spec.Mounts))
		}
		m := spec.Mounts[0]
		if m.Type != MountTypeBind {
			t.Errorf("Mount.Type = %q, want %q", m.Type, MountTypeBind)
		}
		if m.Source != "/host" {
			t.Errorf("Mount.Source = %q, want %q", m.Source, "/host")
		}
		if m.Target != "/container" {
			t.Errorf("Mount.Target = %q, want %q", m.Target, "/container")
		}
		if !m.ReadOnly {
			t.Error("Mount.ReadOnly = false, want true")
		}
	})

	t.Run("with tmpfs helper", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithTmpfs("/tmp").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Mounts) != 1 {
			t.Fatalf("Mounts length = %d, want 1", len(spec.Mounts))
		}
		if spec.Mounts[0].Type != MountTypeTmpfs {
			t.Errorf("Mount.Type = %q, want %q", spec.Mounts[0].Type, MountTypeTmpfs)
		}
	})

	t.Run("with resources", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithResources(ResourceSpec{
				MemoryBytes: 256 * 1024 * 1024,
				CPUQuota:    100000,
				PidsLimit:   100,
			}).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Resources.MemoryBytes != 256*1024*1024 {
			t.Errorf("Resources.MemoryBytes = %d, want %d", spec.Resources.MemoryBytes, 256*1024*1024)
		}
	})

	t.Run("with individual resource methods", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithMemory(128 * 1024 * 1024).
			WithCPU(50000).
			WithPidsLimit(50).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Resources.MemoryBytes != 128*1024*1024 {
			t.Errorf("Resources.MemoryBytes = %d, want %d", spec.Resources.MemoryBytes, 128*1024*1024)
		}
		if spec.Resources.CPUQuota != 50000 {
			t.Errorf("Resources.CPUQuota = %d, want %d", spec.Resources.CPUQuota, 50000)
		}
		if spec.Resources.PidsLimit != 50 {
			t.Errorf("Resources.PidsLimit = %d, want %d", spec.Resources.PidsLimit, 50)
		}
	})

	t.Run("with security", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithSecurity(SecuritySpec{
				User:           "nobody:nogroup",
				ReadOnlyRootfs: true,
				NetworkMode:    "none",
			}).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Security.User != "nobody:nogroup" {
			t.Errorf("Security.User = %q, want %q", spec.Security.User, "nobody:nogroup")
		}
		if !spec.Security.ReadOnlyRootfs {
			t.Error("Security.ReadOnlyRootfs = false, want true")
		}
	})

	t.Run("with individual security methods", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithUser("app:app").
			WithReadOnlyRootfs(true).
			WithNetworkMode("bridge").
			WithSeccompProfile("/path/to/profile").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Security.User != "app:app" {
			t.Errorf("Security.User = %q, want %q", spec.Security.User, "app:app")
		}
		if !spec.Security.ReadOnlyRootfs {
			t.Error("Security.ReadOnlyRootfs = false, want true")
		}
		if spec.Security.NetworkMode != "bridge" {
			t.Errorf("Security.NetworkMode = %q, want %q", spec.Security.NetworkMode, "bridge")
		}
		if spec.Security.SeccompProfile != "/path/to/profile" {
			t.Errorf("Security.SeccompProfile = %q, want %q", spec.Security.SeccompProfile, "/path/to/profile")
		}
	})

	t.Run("with no network", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithNoNetwork().
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Security.NetworkMode != "none" {
			t.Errorf("Security.NetworkMode = %q, want %q", spec.Security.NetworkMode, "none")
		}
	})

	t.Run("with timeout", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithTimeout(30 * time.Second).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want %v", spec.Timeout, 30*time.Second)
		}
	})

	t.Run("with label", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithLabel("app", "test").
			WithLabel("env", "dev").
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if spec.Labels["app"] != "test" {
			t.Errorf("Labels[app] = %q, want %q", spec.Labels["app"], "test")
		}
		if spec.Labels["env"] != "dev" {
			t.Errorf("Labels[env] = %q, want %q", spec.Labels["env"], "dev")
		}
	})

	t.Run("with labels map", func(t *testing.T) {
		spec, err := NewSpecBuilder("alpine:latest").
			WithLabels(map[string]string{"a": "1", "b": "2"}).
			Build()
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		if len(spec.Labels) != 2 {
			t.Errorf("Labels length = %d, want 2", len(spec.Labels))
		}
	})

	t.Run("privileged rejected", func(t *testing.T) {
		_, err := NewSpecBuilder("alpine:latest").
			WithSecurity(SecuritySpec{Privileged: true}).
			Build()
		if err == nil {
			t.Error("Build() with privileged should fail")
		}
	})

	t.Run("host network rejected", func(t *testing.T) {
		_, err := NewSpecBuilder("alpine:latest").
			WithNetworkMode("host").
			Build()
		if err == nil {
			t.Error("Build() with host network should fail")
		}
	})
}

func TestSpecBuilderMustBuild(t *testing.T) {
	t.Run("valid spec", func(t *testing.T) {
		spec := NewSpecBuilder("alpine:latest").MustBuild()
		if spec.Image != "alpine:latest" {
			t.Errorf("Image = %q, want %q", spec.Image, "alpine:latest")
		}
	})

	t.Run("invalid spec panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustBuild() with invalid spec should panic")
			}
		}()
		NewSpecBuilder("").MustBuild()
	})
}

func TestSpecBuilderChaining(t *testing.T) {
	// Test that all methods return the builder for chaining
	spec, err := NewSpecBuilder("alpine:latest").
		WithCommand("sh", "-c", "echo hello").
		WithWorkingDir("/app").
		WithEnv("FOO", "bar").
		WithEnvs([]string{"BAZ=qux"}).
		WithBindMount("/host", "/container", true).
		WithTmpfs("/tmp").
		WithMemory(256 * 1024 * 1024).
		WithCPU(100000).
		WithPidsLimit(100).
		WithUser("nobody:nogroup").
		WithReadOnlyRootfs(true).
		WithNoNetwork().
		WithSeccompProfile("/path/to/profile").
		WithTimeout(30 * time.Second).
		WithLabel("app", "test").
		WithLabels(map[string]string{"env": "dev"}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Verify key fields
	if spec.Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", spec.Image, "alpine:latest")
	}
	if len(spec.Command) != 3 {
		t.Errorf("Command length = %d, want 3", len(spec.Command))
	}
	if spec.WorkingDir != "/app" {
		t.Errorf("WorkingDir = %q, want %q", spec.WorkingDir, "/app")
	}
	if len(spec.Env) != 2 {
		t.Errorf("Env length = %d, want 2", len(spec.Env))
	}
	if len(spec.Mounts) != 2 {
		t.Errorf("Mounts length = %d, want 2", len(spec.Mounts))
	}
	if spec.Security.NetworkMode != "none" {
		t.Errorf("Security.NetworkMode = %q, want %q", spec.Security.NetworkMode, "none")
	}
}
