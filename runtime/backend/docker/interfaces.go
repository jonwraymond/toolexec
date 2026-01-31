package docker

import "context"

// ImageResolver checks if an image exists locally and pulls if needed.
// This is an optional interface - backends may assume images are pre-pulled.
type ImageResolver interface {
	// Resolve ensures the image is available locally.
	// Returns the resolved image reference (may include digest).
	// If the image doesn't exist locally, it will be pulled.
	Resolve(ctx context.Context, image string) (string, error)
}

// HealthChecker verifies Docker daemon availability.
// This is an optional interface - backends may skip health checks.
type HealthChecker interface {
	// Ping checks if the Docker daemon is responsive.
	Ping(ctx context.Context) error

	// Info returns daemon information (version, OS, etc.).
	Info(ctx context.Context) (DaemonInfo, error)
}

// StreamRunner provides streaming execution for long-running containers.
// This is an optional extension to ContainerRunner.
type StreamRunner interface {
	ContainerRunner

	// RunStream executes and streams stdout/stderr as events.
	// The returned channel is closed when execution completes.
	// Callers should drain the channel to receive the exit event.
	RunStream(ctx context.Context, spec ContainerSpec) (<-chan StreamEvent, error)
}
