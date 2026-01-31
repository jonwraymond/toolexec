package docker

import (
	"maps"
	"time"
)

// SpecBuilder constructs ContainerSpec with validation.
type SpecBuilder struct {
	spec ContainerSpec
}

// NewSpecBuilder creates a builder for ContainerSpec.
func NewSpecBuilder(image string) *SpecBuilder {
	return &SpecBuilder{
		spec: ContainerSpec{Image: image},
	}
}

// WithCommand sets the command to execute.
func (b *SpecBuilder) WithCommand(cmd ...string) *SpecBuilder {
	b.spec.Command = cmd
	return b
}

// WithWorkingDir sets the working directory inside the container.
func (b *SpecBuilder) WithWorkingDir(dir string) *SpecBuilder {
	b.spec.WorkingDir = dir
	return b
}

// WithEnv adds an environment variable.
func (b *SpecBuilder) WithEnv(key, value string) *SpecBuilder {
	b.spec.Env = append(b.spec.Env, key+"="+value)
	return b
}

// WithEnvs adds multiple environment variables from a slice.
func (b *SpecBuilder) WithEnvs(envs []string) *SpecBuilder {
	b.spec.Env = append(b.spec.Env, envs...)
	return b
}

// WithMount adds a volume mount.
func (b *SpecBuilder) WithMount(m Mount) *SpecBuilder {
	b.spec.Mounts = append(b.spec.Mounts, m)
	return b
}

// WithBindMount adds a bind mount from host to container.
func (b *SpecBuilder) WithBindMount(source, target string, readOnly bool) *SpecBuilder {
	return b.WithMount(Mount{
		Type:     MountTypeBind,
		Source:   source,
		Target:   target,
		ReadOnly: readOnly,
	})
}

// WithTmpfs adds a tmpfs mount at the given target.
func (b *SpecBuilder) WithTmpfs(target string) *SpecBuilder {
	return b.WithMount(Mount{
		Type:   MountTypeTmpfs,
		Target: target,
	})
}

// WithResources sets the resource limits.
func (b *SpecBuilder) WithResources(r ResourceSpec) *SpecBuilder {
	b.spec.Resources = r
	return b
}

// WithMemory sets the memory limit in bytes.
func (b *SpecBuilder) WithMemory(bytes int64) *SpecBuilder {
	b.spec.Resources.MemoryBytes = bytes
	return b
}

// WithCPU sets the CPU quota in microseconds.
func (b *SpecBuilder) WithCPU(quota int64) *SpecBuilder {
	b.spec.Resources.CPUQuota = quota
	return b
}

// WithPidsLimit sets the maximum number of processes.
func (b *SpecBuilder) WithPidsLimit(limit int64) *SpecBuilder {
	b.spec.Resources.PidsLimit = limit
	return b
}

// WithSecurity sets the security specification.
func (b *SpecBuilder) WithSecurity(s SecuritySpec) *SpecBuilder {
	b.spec.Security = s
	return b
}

// WithUser sets the user to run as.
func (b *SpecBuilder) WithUser(user string) *SpecBuilder {
	b.spec.Security.User = user
	return b
}

// WithReadOnlyRootfs sets the root filesystem to read-only.
func (b *SpecBuilder) WithReadOnlyRootfs(readOnly bool) *SpecBuilder {
	b.spec.Security.ReadOnlyRootfs = readOnly
	return b
}

// WithNetworkMode sets the network mode.
func (b *SpecBuilder) WithNetworkMode(mode string) *SpecBuilder {
	b.spec.Security.NetworkMode = mode
	return b
}

// WithNoNetwork disables network access.
func (b *SpecBuilder) WithNoNetwork() *SpecBuilder {
	return b.WithNetworkMode("none")
}

// WithSeccompProfile sets the seccomp profile path.
func (b *SpecBuilder) WithSeccompProfile(path string) *SpecBuilder {
	b.spec.Security.SeccompProfile = path
	return b
}

// WithTimeout sets the execution timeout.
func (b *SpecBuilder) WithTimeout(d time.Duration) *SpecBuilder {
	b.spec.Timeout = d
	return b
}

// WithLabel adds a container label.
func (b *SpecBuilder) WithLabel(key, value string) *SpecBuilder {
	if b.spec.Labels == nil {
		b.spec.Labels = make(map[string]string)
	}
	b.spec.Labels[key] = value
	return b
}

// WithLabels merges labels into the spec.
func (b *SpecBuilder) WithLabels(labels map[string]string) *SpecBuilder {
	if b.spec.Labels == nil {
		b.spec.Labels = make(map[string]string, len(labels))
	}
	maps.Copy(b.spec.Labels, labels)
	return b
}

// Build validates and returns the ContainerSpec.
func (b *SpecBuilder) Build() (ContainerSpec, error) {
	if err := b.spec.Validate(); err != nil {
		return ContainerSpec{}, err
	}
	return b.spec, nil
}

// MustBuild validates and returns the ContainerSpec, panicking on error.
// Use only in tests or when the spec is known to be valid.
func (b *SpecBuilder) MustBuild() ContainerSpec {
	spec, err := b.Build()
	if err != nil {
		panic(err)
	}
	return spec
}
