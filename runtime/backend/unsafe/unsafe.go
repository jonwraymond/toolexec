// Package unsafe provides a backend that executes code directly on the host.
// WARNING: This backend provides no isolation. Use only for trusted code in development.
package unsafe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonwraymond/toolexec/runtime"
)

// ExecutionMode determines how code is executed.
type ExecutionMode string

const (
	// ModeInterpreter uses an in-process Go interpreter (yaegi).
	// Faster startup but limited Go compatibility.
	ModeInterpreter ExecutionMode = "interpreter"

	// ModeSubprocess uses `go run` to execute code.
	// Full Go compatibility but requires Go toolchain.
	ModeSubprocess ExecutionMode = "subprocess"
)

// Errors for unsafe backend operations.
var (
	// ErrOptInRequired is returned when RequireOptIn is true and opt-in is not provided.
	ErrOptInRequired = errors.New("unsafe backend requires explicit opt-in")

	// ErrInterpreterNotAvailable is returned when interpreter mode fails.
	ErrInterpreterNotAvailable = errors.New("interpreter not available")

	// ErrSubprocessFailed is returned when subprocess execution fails.
	ErrSubprocessFailed = errors.New("subprocess execution failed")
)

// Logger is the interface for logging.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Errors: logging must be best-effort and must not panic.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Config configures an unsafe backend.
type Config struct {
	// Mode determines how code is executed.
	// Default: ModeInterpreter
	Mode ExecutionMode

	// Logger is an optional logger for backend events.
	Logger Logger

	// RequireOptIn requires explicit opt-in via request metadata.
	// When true, requests must include metadata["unsafeOptIn"] = true.
	RequireOptIn bool
}

// Backend executes code directly on the host without isolation.
// WARNING: This backend provides no security isolation. Use only for trusted code.
type Backend struct {
	mode         ExecutionMode
	logger       Logger
	requireOptIn bool
}

// New creates a new unsafe backend with the given configuration.
func New(cfg Config) *Backend {
	mode := cfg.Mode
	if mode == "" {
		mode = ModeInterpreter
	}

	return &Backend{
		mode:         mode,
		logger:       cfg.Logger,
		requireOptIn: cfg.RequireOptIn,
	}
}

// Kind returns the backend kind identifier.
func (b *Backend) Kind() runtime.BackendKind {
	return runtime.BackendUnsafeHost
}

// Execute runs code on the host without isolation.
func (b *Backend) Execute(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return runtime.ExecuteResult{}, err
	}

	// Check opt-in requirement
	if b.requireOptIn {
		optIn, ok := req.Metadata["unsafeOptIn"].(bool)
		if !ok || !optIn {
			return runtime.ExecuteResult{}, ErrOptInRequired
		}
	}

	// Log UNSAFE warning
	if b.logger != nil {
		b.logger.Warn("UNSAFE: executing code without isolation",
			"mode", b.mode,
			"codeLen", len(req.Code))
	}

	// Apply timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	var result runtime.ExecuteResult
	var err error

	switch b.mode {
	case ModeInterpreter:
		result, err = b.executeInterpreter(ctx, req)
	case ModeSubprocess:
		result, err = b.executeSubprocess(ctx, req)
	default:
		result, err = b.executeInterpreter(ctx, req)
	}

	result.Duration = time.Since(start)
	result.Backend = runtime.BackendInfo{
		Kind:      runtime.BackendUnsafeHost,
		Readiness: runtime.ReadinessProd,
		Details: map[string]any{
			"mode": string(b.mode),
		},
	}

	return result, err
}

// executeInterpreter executes code using an in-process interpreter.
// Note: This is a simplified implementation. A full implementation would use yaegi.
func (b *Backend) executeInterpreter(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	// For now, fall back to subprocess since yaegi integration is complex
	// A full implementation would:
	// 1. Create a yaegi interpreter
	// 2. Inject the gateway as a symbol
	// 3. Execute the code
	// 4. Extract __out value

	return b.executeSubprocess(ctx, req)
}

// executeSubprocess executes code using `go run`.
func (b *Backend) executeSubprocess(ctx context.Context, req runtime.ExecuteRequest) (runtime.ExecuteResult, error) {
	// Create a temporary directory for the code
	tmpDir, err := os.MkdirTemp("", "toolruntime-unsafe-*")
	if err != nil {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: failed to create temp dir: %v", ErrSubprocessFailed, err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Wrap the code in a main function
	wrappedCode := wrapCode(req.Code)

	// Write the code to a file
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(wrappedCode), 0600); err != nil {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: failed to write code: %v", ErrSubprocessFailed, err)
	}

	// Create go.mod
	goMod := `module toolruntime_exec

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0600); err != nil {
		return runtime.ExecuteResult{}, fmt.Errorf("%w: failed to write go.mod: %v", ErrSubprocessFailed, err)
	}

	// Run the code
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := runtime.ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if ctx.Err() != nil {
			return result, fmt.Errorf("%w: %v", runtime.ErrTimeout, ctx.Err())
		}
		return result, fmt.Errorf("%w: %v\nstderr: %s", ErrSubprocessFailed, err, stderr.String())
	}

	// Extract __out value from stdout
	// The wrapped code prints "__OUT__:<value>" at the end
	result.Value = extractOutValue(stdout.String())

	return result, nil
}

// wrapCode wraps user code in a main function with output capture.
func wrapCode(code string) string {
	// Check if code already has package/imports
	hasPackage := strings.Contains(code, "package ")
	hasMain := strings.Contains(code, "func main()")

	if hasPackage && hasMain {
		// Code is already complete
		return code
	}

	// Wrap in main function with __out capture
	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var __out any

	// User code starts here
	%s
	// User code ends here

	// Output __out value
	if __out != nil {
		data, _ := json.Marshal(__out)
		fmt.Printf("__OUT__:%%s\n", string(data))
	}
}
`, code)
}

// extractOutValue extracts the __out value from stdout.
func extractOutValue(stdout string) any {
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "__OUT__:") {
			jsonStr := strings.TrimPrefix(line, "__OUT__:")
			var value any
			if err := json.Unmarshal([]byte(jsonStr), &value); err == nil {
				return value
			}
			return jsonStr
		}
	}
	return nil
}
