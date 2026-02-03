package kubernetes

import "time"

// ResourceSpec defines Kubernetes resource limits.
type ResourceSpec struct {
	MemoryBytes int64
	CPUQuota    int64
	PidsLimit   int64
	DiskBytes   int64
}

// SecuritySpec defines Kubernetes security settings.
type SecuritySpec struct {
	User           string
	ReadOnlyRootfs bool
	NetworkMode    string
}

// PodSpec defines what to run inside a Kubernetes pod/job.
type PodSpec struct {
	Namespace        string
	Image            string
	Command          []string
	Args             []string
	WorkingDir       string
	Env              []string
	RuntimeClassName string
	ServiceAccount   string
	Resources        ResourceSpec
	Security         SecuritySpec
	Timeout          time.Duration
	Labels           map[string]string
}

// PodResult captures the output of pod execution.
type PodResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}
