package containerd

import "context"

// ContainerRunner executes a containerd task for a given spec.
//
// Contract:
// - Concurrency: Implementations must be safe for concurrent use.
// - Context: Run must honor cancellation and deadlines.
// - Ownership: Implementations must not mutate the provided spec.
type ContainerRunner interface {
	Run(ctx context.Context, spec ContainerSpec) (ContainerResult, error)
}

// HealthChecker can verify containerd availability.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// ImageResolver resolves/pulls images before execution.
type ImageResolver interface {
	Resolve(ctx context.Context, image string) (string, error)
}
