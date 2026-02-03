package firecracker

import "context"

// MicroVMRunner executes a Firecracker microVM for a given spec.
//
// Contract:
// - Concurrency: Implementations must be safe for concurrent use.
// - Context: Run must honor cancellation and deadlines.
// - Ownership: Implementations must not mutate the provided spec.
type MicroVMRunner interface {
	Run(ctx context.Context, spec MicroVMSpec) (MicroVMResult, error)
}

// HealthChecker can verify Firecracker availability.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
