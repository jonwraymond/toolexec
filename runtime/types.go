package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// SecurityProfile determines the security level for execution.
// Higher security profiles impose more restrictions but provide better isolation.
type SecurityProfile string

const (
	// ProfileDev is development mode with minimal restrictions.
	// WARNING: This profile runs code with host access - use only for development.
	ProfileDev SecurityProfile = "dev"

	// ProfileStandard provides standard isolation.
	// Includes: no network access, read-only rootfs, resource limits.
	ProfileStandard SecurityProfile = "standard"

	// ProfileHardened provides maximum isolation.
	// Includes: seccomp profiles, stricter resource limits, additional syscall filtering.
	ProfileHardened SecurityProfile = "hardened"
)

// IsValid returns true if the SecurityProfile is a known valid value.
func (p SecurityProfile) IsValid() bool {
	switch p {
	case ProfileDev, ProfileStandard, ProfileHardened:
		return true
	default:
		return false
	}
}

// BackendKind identifies the type of execution backend.
type BackendKind string

const (
	// BackendUnsafeHost runs code directly on the host.
	// WARNING: No isolation - use only for trusted code in development.
	BackendUnsafeHost BackendKind = "unsafe_host"

	// BackendDocker runs code in Docker containers.
	// Good default isolation with cgroups, read-only rootfs, user remapping, and seccomp.
	BackendDocker BackendKind = "docker"

	// BackendContainerd runs code via containerd directly.
	// Similar to Docker but more infrastructure-native for servers/agents.
	BackendContainerd BackendKind = "containerd"

	// BackendKubernetes executes snippets in short-lived pods/jobs.
	// Isolation depends on configured runtime class; best for scheduling and multi-tenant controls.
	BackendKubernetes BackendKind = "kubernetes"

	// BackendGVisor runs code with gVisor (runsc) for stronger isolation.
	// Appropriate for untrusted multi-tenant execution.
	BackendGVisor BackendKind = "gvisor"

	// BackendKata runs code in Kata Containers for VM-level isolation.
	// Stronger isolation than plain containers.
	BackendKata BackendKind = "kata"

	// BackendFirecracker runs code in Firecracker microVMs.
	// Strongest isolation; higher complexity and operational cost.
	BackendFirecracker BackendKind = "firecracker"

	// BackendWASM runs code compiled to WebAssembly.
	// Strong in-process isolation; requires constrained SDK surface.
	BackendWASM BackendKind = "wasm"

	// BackendTemporal treats snippet execution as a Temporal workflow/activity.
	// Useful for long-running or resumable executions.
	// Note: Temporal is orchestration, not isolation - must compose with sandbox backends.
	BackendTemporal BackendKind = "temporal"

	// BackendRemote executes code on a remote runtime service.
	// Generic target for dedicated runtime services, batch systems, or job runners.
	BackendRemote BackendKind = "remote"

	// BackendProxmoxLXC executes code in a Proxmox LXC container.
	// Requires a runtime service inside the container.
	BackendProxmoxLXC BackendKind = "proxmox_lxc"
)

// BackendReadiness indicates the maturity of a backend implementation.
type BackendReadiness string

const (
	// ReadinessProd indicates production-ready support.
	ReadinessProd BackendReadiness = "prod"
	// ReadinessBeta indicates usable but still evolving support.
	ReadinessBeta BackendReadiness = "beta"
	// ReadinessStub indicates placeholder or incomplete support.
	ReadinessStub BackendReadiness = "stub"
)

// Limits specifies resource limits for execution.
// Zero values represent "unlimited" for that resource.
type Limits struct {
	// MaxToolCalls limits the number of tool invocations.
	// Zero means unlimited.
	MaxToolCalls int

	// MaxChainSteps limits the number of steps in a tool chain.
	// Zero means unlimited.
	MaxChainSteps int

	// CPUQuotaMillis limits CPU time in milliseconds.
	// Zero means unlimited.
	CPUQuotaMillis int64

	// MemoryBytes limits memory usage in bytes.
	// Zero means unlimited.
	MemoryBytes int64

	// PidsMax limits the number of processes/threads.
	// Zero means unlimited.
	PidsMax int64

	// DiskBytes limits disk usage in bytes.
	// Zero means unlimited.
	DiskBytes int64
}

// Validate checks that all limit values are valid (non-negative).
func (l Limits) Validate() error {
	if l.MaxToolCalls < 0 {
		return fmt.Errorf("%w: MaxToolCalls cannot be negative", ErrInvalidLimits)
	}
	if l.MaxChainSteps < 0 {
		return fmt.Errorf("%w: MaxChainSteps cannot be negative", ErrInvalidLimits)
	}
	if l.CPUQuotaMillis < 0 {
		return fmt.Errorf("%w: CPUQuotaMillis cannot be negative", ErrInvalidLimits)
	}
	if l.MemoryBytes < 0 {
		return fmt.Errorf("%w: MemoryBytes cannot be negative", ErrInvalidLimits)
	}
	if l.PidsMax < 0 {
		return fmt.Errorf("%w: PidsMax cannot be negative", ErrInvalidLimits)
	}
	if l.DiskBytes < 0 {
		return fmt.Errorf("%w: DiskBytes cannot be negative", ErrInvalidLimits)
	}
	return nil
}

// ExecuteRequest specifies the parameters for code execution.
type ExecuteRequest struct {
	// Language specifies the programming language of the code.
	// If empty, the backend's default language is used.
	Language string

	// Code is the source code to execute.
	// Required.
	Code string

	// Timeout specifies the maximum duration for execution.
	// If zero, the backend's default timeout is used.
	Timeout time.Duration

	// Limits specifies resource limits for execution.
	Limits Limits

	// Profile specifies the security profile to use.
	// If empty, the runtime's default profile is used.
	Profile SecurityProfile

	// Gateway is the tool gateway exposed to the executed code.
	// Required.
	Gateway ToolGateway

	// Metadata contains arbitrary metadata for the execution.
	Metadata map[string]any
}

// Validate checks that the request is valid.
func (r ExecuteRequest) Validate() error {
	if r.Gateway == nil {
		return ErrMissingGateway
	}
	if r.Code == "" {
		return ErrMissingCode
	}
	if err := r.Limits.Validate(); err != nil {
		return err
	}
	return nil
}

// ExecuteResult contains the outcome of code execution.
type ExecuteResult struct {
	// Value is the final result of the code execution.
	// Typically captured via the __out variable convention.
	Value any

	// Stdout contains any output written to stdout.
	Stdout string

	// Stderr contains any output written to stderr.
	Stderr string

	// ToolCalls records all tool invocations made during execution.
	ToolCalls []ToolCallRecord

	// Duration is the total execution time.
	Duration time.Duration

	// Backend contains information about the backend that executed the code.
	Backend BackendInfo

	// LimitsEnforced reports which limits the backend was able to enforce.
	// Backends that cannot enforce a given limit should set that field to false.
	// This allows callers to know when limits degraded gracefully.
	LimitsEnforced LimitsEnforced
}

// LimitsEnforced reports which resource limits were actually enforced by the backend.
// Backends that cannot enforce a limit should set that field to false.
type LimitsEnforced struct {
	// Timeout indicates whether the timeout was enforced.
	Timeout bool

	// ToolCalls indicates whether tool call limits were enforced.
	ToolCalls bool

	// ChainSteps indicates whether chain step limits were enforced.
	ChainSteps bool

	// Memory indicates whether memory limits were enforced.
	Memory bool

	// CPU indicates whether CPU limits were enforced.
	CPU bool

	// Pids indicates whether process limits were enforced.
	Pids bool

	// Disk indicates whether disk limits were enforced.
	Disk bool
}

// ToolCallRecord captures information about a single tool invocation.
type ToolCallRecord struct {
	// ToolID is the canonical identifier of the tool that was called.
	ToolID string

	// BackendKind indicates which backend executed the tool.
	BackendKind string

	// Duration is the execution time for this tool call.
	Duration time.Duration

	// ErrorOp indicates the operation that failed, if any.
	ErrorOp string
}

// BackendInfo contains information about the execution backend.
type BackendInfo struct {
	// Kind identifies the type of backend.
	Kind BackendKind

	// Readiness indicates the maturity tier of the backend.
	Readiness BackendReadiness

	// Details contains backend-specific information.
	Details map[string]any
}

// ToolGateway is the interface for tool operations exposed to sandboxed code.
// It provides a proxy for tool discovery and execution while maintaining
// the trust boundary between the sandbox and the host.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods must honor cancellation/deadlines and return ctx.Err() when canceled.
// - Ownership: args are read-only; results are caller-owned snapshots.
type ToolGateway interface {
	// SearchTools searches for tools matching the query.
	SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error)

	// ListNamespaces returns all available tool namespaces.
	ListNamespaces(ctx context.Context) ([]string, error)

	// DescribeTool returns documentation for a tool at the specified detail level.
	DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error)

	// ListToolExamples returns up to maxExamples usage examples for a tool.
	ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error)

	// RunTool executes a single tool and returns the result.
	RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error)

	// RunChain executes a sequence of tool calls.
	RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error)
}
