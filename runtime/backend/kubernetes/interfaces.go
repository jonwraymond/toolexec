package kubernetes

import "context"

// PodRunner executes a Kubernetes pod/job for a given spec.
//
// Contract:
// - Concurrency: Implementations must be safe for concurrent use.
// - Context: Run must honor cancellation and deadlines.
// - Ownership: Implementations must not mutate the provided spec.
type PodRunner interface {
	Run(ctx context.Context, spec PodSpec) (PodResult, error)
}

// HealthChecker can verify Kubernetes cluster availability.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// ImageResolver optionally resolves/pulls images before execution.
// For Kubernetes this is typically a no-op but allows custom registries.
type ImageResolver interface {
	Resolve(ctx context.Context, image string) (string, error)
}
