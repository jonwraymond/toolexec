package docker

import "time"

// MountType defines the type of volume mount.
type MountType string

const (
	// MountTypeBind mounts a host path into the container.
	MountTypeBind MountType = "bind"

	// MountTypeVolume mounts a Docker volume.
	MountTypeVolume MountType = "volume"

	// MountTypeTmpfs mounts a tmpfs (in-memory) filesystem.
	MountTypeTmpfs MountType = "tmpfs"
)

// Mount defines a volume mount for a container.
type Mount struct {
	// Type specifies the mount type: "bind", "volume", or "tmpfs".
	Type MountType

	// Source is the host path (bind) or volume name (volume).
	// Ignored for tmpfs mounts.
	Source string

	// Target is the path inside the container.
	Target string

	// ReadOnly mounts the volume as read-only.
	ReadOnly bool

	// Consistency is the mount consistency mode: "consistent", "cached", "delegated".
	// Only relevant for bind mounts on macOS.
	Consistency string
}

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
	// Image is the Docker image reference (required).
	Image string

	// Command is the command to execute.
	Command []string

	// WorkingDir is the working directory inside the container.
	WorkingDir string

	// Env contains environment variables in KEY=value format.
	Env []string

	// Mounts defines volume mounts for the container.
	Mounts []Mount

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

// StreamEventType identifies the type of streaming event.
type StreamEventType string

const (
	// StreamEventStdout indicates stdout data.
	StreamEventStdout StreamEventType = "stdout"

	// StreamEventStderr indicates stderr data.
	StreamEventStderr StreamEventType = "stderr"

	// StreamEventExit indicates container exit.
	StreamEventExit StreamEventType = "exit"

	// StreamEventError indicates an error occurred.
	StreamEventError StreamEventType = "error"
)

// StreamEvent represents a streaming output event from container execution.
type StreamEvent struct {
	// Type identifies the event type.
	Type StreamEventType

	// Data contains the event payload (stdout/stderr bytes).
	Data []byte

	// ExitCode is set when Type is StreamEventExit.
	ExitCode int

	// Error is set when Type is StreamEventError.
	Error error
}

// DaemonInfo contains Docker daemon metadata.
type DaemonInfo struct {
	// Version is the daemon version.
	Version string

	// APIVersion is the API version.
	APIVersion string

	// OS is the daemon operating system.
	OS string

	// Architecture is the daemon CPU architecture.
	Architecture string

	// RootDir is the daemon's root directory.
	RootDir string
}
