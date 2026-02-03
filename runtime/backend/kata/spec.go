package kata

import "time"

// ResourceSpec defines Kata resource limits.
type ResourceSpec struct {
	MemoryBytes int64
	CPUQuota    int64
	PidsLimit   int64
	DiskBytes   int64
}

// SecuritySpec defines Kata security settings.
type SecuritySpec struct {
	User           string
	ReadOnlyRootfs bool
	NetworkMode    string
	SeccompProfile string
	Privileged     bool
}

// SandboxSpec defines what to run inside Kata Containers.
type SandboxSpec struct {
	Image      string
	Runtime    string
	Hypervisor string
	KernelPath string
	ImagePath  string
	Command    []string
	WorkingDir string
	Env        []string
	Resources  ResourceSpec
	Security   SecuritySpec
	Timeout    time.Duration
	Labels     map[string]string
}

// SandboxResult captures the output of a Kata execution.
type SandboxResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}
