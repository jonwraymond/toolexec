package firecracker

import "time"

// VMResourceSpec defines resource limits for microVMs.
type VMResourceSpec struct {
	VCPUCount int
	MemSizeMB int
}

// VMConfig defines microVM configuration.
type VMConfig struct {
	KernelPath string
	RootfsPath string
	SocketPath string
}

// MicroVMSpec defines what to run inside a Firecracker microVM.
type MicroVMSpec struct {
	Image      string
	Command    []string
	WorkingDir string
	Env        []string
	Resources  VMResourceSpec
	Config     VMConfig
	Timeout    time.Duration
	Labels     map[string]string
}

// MicroVMResult captures the output of a microVM execution.
type MicroVMResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}
