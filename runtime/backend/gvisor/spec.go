package gvisor

import "time"

// ResourceSpec defines sandbox resource limits.
type ResourceSpec struct {
	MemoryBytes int64
	CPUQuota    int64
	PidsLimit   int64
	DiskBytes   int64
}

// SecuritySpec defines sandbox security settings.
type SecuritySpec struct {
	User           string
	ReadOnlyRootfs bool
	NetworkMode    string
	SeccompProfile string
	Privileged     bool
}

// SandboxSpec defines what to run inside the gVisor sandbox.
type SandboxSpec struct {
	Image      string
	Command    []string
	WorkingDir string
	Env        []string
	Resources  ResourceSpec
	Security   SecuritySpec
	Platform   string
	RunscPath  string
	RootDir    string
	Timeout    time.Duration
	Labels     map[string]string
}

// SandboxResult captures the output of a gVisor execution.
type SandboxResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}
