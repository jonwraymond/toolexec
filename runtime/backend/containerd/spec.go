package containerd

import "time"

// ResourceSpec defines container resource limits.
type ResourceSpec struct {
	// MemoryBytes is the memory limit in bytes.
	// Zero means unlimited.
	MemoryBytes int64

	// CPUQuota is the CPU quota in microseconds per 100ms period.
	// Zero means unlimited.
	CPUQuota int64

	// PidsLimit is the maximum number of processes.
	// Zero means unlimited.
	PidsLimit int64

	// DiskBytes is the disk limit in bytes.
	// Zero means unlimited. Not all runtimes support this.
	DiskBytes int64
}

// SecuritySpec defines container security settings.
type SecuritySpec struct {
	// User is the user to run as (e.g., "nobody:nogroup").
	User string

	// ReadOnlyRootfs mounts the root filesystem as read-only.
	ReadOnlyRootfs bool

	// NetworkMode is the network mode: "none", "bridge", "host".
	// "host" is not allowed in sandbox contexts.
	NetworkMode string

	// SeccompProfile is the path to a seccomp profile.
	// Empty uses the runtime's default profile.
	SeccompProfile string

	// Privileged grants extended privileges to the container.
	// Must always be false in sandbox contexts.
	Privileged bool
}

// ContainerSpec defines what to run in a container and how.
type ContainerSpec struct {
	// Image is the container image reference (required).
	Image string

	// Runtime is the containerd runtime to use (e.g., "io.containerd.runc.v2").
	Runtime string

	// Command is the command to execute.
	Command []string

	// WorkingDir is the working directory inside the container.
	WorkingDir string

	// Env contains environment variables in KEY=value format.
	Env []string

	// Resources defines resource limits.
	Resources ResourceSpec

	// Security defines security settings.
	Security SecuritySpec

	// Timeout is the maximum execution duration.
	Timeout time.Duration

	// Labels are container labels for tracking.
	Labels map[string]string
}

// ContainerResult captures the output of container execution.
type ContainerResult struct {
	// ExitCode is the container's exit code.
	ExitCode int

	// Stdout contains the container's stdout output.
	Stdout string

	// Stderr contains the container's stderr output.
	Stderr string

	// Duration is the execution time.
	Duration time.Duration
}
