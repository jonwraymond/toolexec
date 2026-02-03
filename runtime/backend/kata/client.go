package kata

import "context"

// SandboxRunner executes Kata containers for a given spec.
//
// Contract:
// - Concurrency: Implementations must be safe for concurrent use.
// - Context: Run must honor cancellation and deadlines.
// - Ownership: Implementations must not mutate the provided spec.
type SandboxRunner interface {
	Run(ctx context.Context, spec SandboxSpec) (SandboxResult, error)
}

// HealthChecker can verify kata-runtime availability.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// ImageResolver resolves/pulls images before execution.
type ImageResolver interface {
	Resolve(ctx context.Context, image string) (string, error)
}
