package code

// Logger is an optional interface for observability during code execution.
// Implementations can log tool calls, timing information, and other events.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Errors: logging must be best-effort; Logf should not panic.
// - Ownership: format/args are read-only.
type Logger interface {
	// Logf logs a formatted message.
	Logf(format string, args ...any)
}
