package exec

import (
	"errors"
	"time"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// Default configuration values.
const (
	DefaultMaxToolCalls = 100
	DefaultLanguage     = "go"
	DefaultTimeout      = 30 * time.Second
)

// Errors returned by Options validation.
var (
	ErrIndexRequired = errors.New("exec: Index is required")
	ErrDocsRequired  = errors.New("exec: Docs store is required")
)

// Options configures an Exec instance.
type Options struct {
	// Index provides tool discovery and registration.
	// Required.
	Index index.Index

	// Docs provides tool documentation.
	// Required.
	Docs tooldoc.Store

	// LocalHandlers maps handler names to handler functions.
	// These are used when a tool's backend is a local backend
	// referencing the handler by name.
	LocalHandlers map[string]Handler

	// MCPExecutor executes MCP backend tools.
	// Optional; if nil, MCP tools cannot be executed.
	MCPExecutor run.MCPExecutor

	// ProviderExecutor executes provider backend tools.
	// Optional; if nil, provider tools cannot be executed.
	ProviderExecutor run.ProviderExecutor

	// SecurityProfile determines the runtime backend for code execution.
	// Default: runtime.ProfileDev
	SecurityProfile runtime.SecurityProfile

	// EnableCodeExecution enables the code execution subsystem.
	// Default: false (tool execution only)
	EnableCodeExecution bool

	// MaxToolCalls limits tool calls in code execution.
	// Default: 100
	MaxToolCalls int

	// DefaultLanguage for code execution.
	// Default: "go"
	DefaultLanguage string

	// DefaultTimeout for tool and code execution.
	// Default: 30s
	DefaultTimeout time.Duration

	// ValidateInput enables input validation before execution.
	// Default: true
	ValidateInput bool

	// ValidateOutput enables output validation after execution.
	// Default: true
	ValidateOutput bool
}

// validate checks that required fields are set.
func (o *Options) validate() error {
	if o.Index == nil {
		return ErrIndexRequired
	}
	if o.Docs == nil {
		return ErrDocsRequired
	}
	return nil
}

// applyDefaults sets default values for unset optional fields.
func (o *Options) applyDefaults() {
	if o.SecurityProfile == "" {
		o.SecurityProfile = runtime.ProfileDev
	}
	if o.MaxToolCalls == 0 {
		o.MaxToolCalls = DefaultMaxToolCalls
	}
	if o.DefaultLanguage == "" {
		o.DefaultLanguage = DefaultLanguage
	}
	if o.DefaultTimeout == 0 {
		o.DefaultTimeout = DefaultTimeout
	}
	// Note: ValidateInput and ValidateOutput default to false (zero value),
	// but we want them to default to true. This is handled in New().
}

// Step defines a single step in a chain execution.
type Step struct {
	// ToolID is the canonical ID of the tool to execute.
	ToolID string

	// Args are the arguments to pass to the tool.
	// If UsePrevious is true and Args is nil, the previous
	// step's result is used as the argument map.
	Args map[string]any

	// UsePrevious indicates that this step should receive
	// the previous step's result. If Args is also set,
	// the previous result is merged into Args under the
	// key "previous" (unless Args already has that key).
	UsePrevious bool

	// StopOnError determines whether chain execution should
	// stop if this step fails. Default is true.
	StopOnError *bool
}

// shouldStopOnError returns whether to stop on error for this step.
func (s Step) shouldStopOnError() bool {
	if s.StopOnError == nil {
		return true
	}
	return *s.StopOnError
}

// CodeParams configures a code execution request.
type CodeParams struct {
	// Language specifies the programming language.
	// If empty, Options.DefaultLanguage is used.
	Language string

	// Code is the source code to execute.
	Code string

	// Timeout overrides Options.DefaultTimeout for this execution.
	// If zero, the default timeout is used.
	Timeout time.Duration

	// MaxToolCalls overrides Options.MaxToolCalls for this execution.
	// If zero, the default limit is used.
	MaxToolCalls int

	// AllowedTools restricts which tools the code can call.
	// If nil or empty, all registered tools are allowed.
	AllowedTools []string

	// Env provides environment variables for the execution.
	Env map[string]string
}
