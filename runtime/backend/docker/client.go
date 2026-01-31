package docker

import "context"

// ContainerRunner is the primary interface for container execution.
// Implementations may use Docker SDK, containerd, testcontainers, or mocks.
//
// The interface is intentionally minimal following Go best practices:
// "The bigger the interface, the weaker the abstraction."
//
// Implementations are expected to:
//   - Create a container from the spec
//   - Start and wait for container completion
//   - Capture stdout/stderr
//   - Remove the container after execution
//   - Respect context cancellation and spec timeout
type ContainerRunner interface {
	// Run executes code in a container and returns the result.
	// The container lifecycle (create, start, wait, remove) is atomic.
	//
	// The implementation must:
	//   - Validate the spec before execution
	//   - Respect ctx cancellation
	//   - Respect spec.Timeout if set
	//   - Capture stdout and stderr
	//   - Return the exit code
	//   - Clean up the container on completion or error
	Run(ctx context.Context, spec ContainerSpec) (ContainerResult, error)
}
