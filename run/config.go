package run

import (
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

// Config controls resolution, validation, and dispatch behavior.
type Config struct {
	// Resolution

	// Index is the tool registry for lookup.
	Index index.Index

	// ToolResolver is a fallback function to resolve tools when Index is not
	// configured or returns ErrNotFound.
	ToolResolver func(id string) (*model.Tool, error)

	// BackendsResolver is a fallback function to resolve backends when Index
	// is not configured or returns ErrNotFound.
	BackendsResolver func(id string) ([]model.ToolBackend, error)

	// BackendSelector chooses which backend to use when multiple are available.
	// Defaults to index.DefaultBackendSelector (local > provider > mcp).
	BackendSelector index.BackendSelector

	// Validation

	// Validator validates tool inputs and outputs against JSON Schema.
	// Defaults to model.NewDefaultValidator().
	Validator model.SchemaValidator

	// ValidateInput enables input validation before execution.
	// Defaults to true.
	ValidateInput bool

	// ValidateOutput enables output validation after execution.
	// Defaults to true.
	ValidateOutput bool

	// Executors

	// MCP is the executor for MCP backend tools.
	MCP MCPExecutor

	// Provider is the executor for provider backend tools.
	Provider ProviderExecutor

	// Local is the registry for local handler functions.
	Local LocalRegistry
}

// applyDefaults sets default values for unset Config fields.
func (c *Config) applyDefaults() {
	if c.Validator == nil {
		c.Validator = model.NewDefaultValidator()
	}
	if c.BackendSelector == nil {
		c.BackendSelector = index.DefaultBackendSelector
	}
}

// ConfigOption is a functional option for configuring a Runner.
type ConfigOption func(*Config)

// WithIndex sets the tool index for resolution.
func WithIndex(idx index.Index) ConfigOption {
	return func(c *Config) {
		c.Index = idx
	}
}

// WithValidator sets a custom schema validator.
func WithValidator(v model.SchemaValidator) ConfigOption {
	return func(c *Config) {
		c.Validator = v
	}
}

// WithMCPExecutor sets the MCP executor.
func WithMCPExecutor(exec MCPExecutor) ConfigOption {
	return func(c *Config) {
		c.MCP = exec
	}
}

// WithProviderExecutor sets the provider executor.
func WithProviderExecutor(exec ProviderExecutor) ConfigOption {
	return func(c *Config) {
		c.Provider = exec
	}
}

// WithLocalRegistry sets the local handler registry.
func WithLocalRegistry(reg LocalRegistry) ConfigOption {
	return func(c *Config) {
		c.Local = reg
	}
}

// WithValidation sets whether to validate inputs and outputs.
func WithValidation(input, output bool) ConfigOption {
	return func(c *Config) {
		c.ValidateInput = input
		c.ValidateOutput = output
	}
}

// WithBackendSelector sets a custom backend selector function.
func WithBackendSelector(selector index.BackendSelector) ConfigOption {
	return func(c *Config) {
		c.BackendSelector = selector
	}
}

// WithToolResolver sets a fallback tool resolver function.
func WithToolResolver(resolver func(id string) (*model.Tool, error)) ConfigOption {
	return func(c *Config) {
		c.ToolResolver = resolver
	}
}

// WithBackendsResolver sets a fallback backends resolver function.
func WithBackendsResolver(resolver func(id string) ([]model.ToolBackend, error)) ConfigOption {
	return func(c *Config) {
		c.BackendsResolver = resolver
	}
}
