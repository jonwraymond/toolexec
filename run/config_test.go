package run

import (
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
)

func TestConfig_ApplyDefaults_Validator(t *testing.T) {
	cfg := Config{}
	cfg.applyDefaults()

	if cfg.Validator == nil {
		t.Error("applyDefaults() should set Validator to non-nil")
	}
}

func TestConfig_ApplyDefaults_ValidateInput(t *testing.T) {
	// ValidateInput should default to true (set in NewRunner, not applyDefaults)
	runner := NewRunner()
	if !runner.cfg.ValidateInput {
		t.Error("NewRunner() should set ValidateInput to true by default")
	}
}

func TestConfig_ApplyDefaults_ValidateOutput(t *testing.T) {
	// ValidateOutput should default to true (set in NewRunner, not applyDefaults)
	runner := NewRunner()
	if !runner.cfg.ValidateOutput {
		t.Error("NewRunner() should set ValidateOutput to true by default")
	}
}

func TestConfig_ApplyDefaults_BackendSelector(t *testing.T) {
	cfg := Config{}
	cfg.applyDefaults()

	if cfg.BackendSelector == nil {
		t.Error("applyDefaults() should set BackendSelector to non-nil")
	}

	// Verify it behaves like the default selector (local > provider > mcp)
	backends := []model.ToolBackend{
		testMCPBackend("server1"),
		testLocalBackend("handler1"),
		testProviderBackend("provider1", "tool1"),
	}

	selected := cfg.BackendSelector(backends)
	if selected.Kind != model.BackendKindLocal {
		t.Errorf("BackendSelector should prefer local, got %s", selected.Kind)
	}
}

func TestWithIndex(t *testing.T) {
	idx := newMockIndex()
	runner := NewRunner(WithIndex(idx))

	if runner.cfg.Index != idx {
		t.Error("WithIndex() did not set Index")
	}
}

func TestWithValidator(t *testing.T) {
	v := newMockValidator()
	runner := NewRunner(WithValidator(v))

	if runner.cfg.Validator != v {
		t.Error("WithValidator() did not set Validator")
	}
}

func TestWithMCPExecutor(t *testing.T) {
	exec := newMockMCPExecutor()
	runner := NewRunner(WithMCPExecutor(exec))

	if runner.cfg.MCP != exec {
		t.Error("WithMCPExecutor() did not set MCP")
	}
}

func TestWithProviderExecutor(t *testing.T) {
	exec := newMockProviderExecutor()
	runner := NewRunner(WithProviderExecutor(exec))

	if runner.cfg.Provider != exec {
		t.Error("WithProviderExecutor() did not set Provider")
	}
}

func TestWithLocalRegistry(t *testing.T) {
	reg := newMockLocalRegistry()
	runner := NewRunner(WithLocalRegistry(reg))

	if runner.cfg.Local != reg {
		t.Error("WithLocalRegistry() did not set Local")
	}
}

func TestWithValidation(t *testing.T) {
	tests := []struct {
		name       string
		input      bool
		output     bool
		wantInput  bool
		wantOutput bool
	}{
		{"both true", true, true, true, true},
		{"both false", false, false, false, false},
		{"input only", true, false, true, false},
		{"output only", false, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(WithValidation(tt.input, tt.output))
			if runner.cfg.ValidateInput != tt.wantInput {
				t.Errorf("ValidateInput = %v, want %v", runner.cfg.ValidateInput, tt.wantInput)
			}
			if runner.cfg.ValidateOutput != tt.wantOutput {
				t.Errorf("ValidateOutput = %v, want %v", runner.cfg.ValidateOutput, tt.wantOutput)
			}
		})
	}
}

func TestWithBackendSelector(t *testing.T) {
	// Custom selector that always picks MCP
	customSelector := func(backends []model.ToolBackend) model.ToolBackend {
		for _, b := range backends {
			if b.Kind == model.BackendKindMCP {
				return b
			}
		}
		return backends[0]
	}

	runner := NewRunner(WithBackendSelector(customSelector))

	backends := []model.ToolBackend{
		testLocalBackend("handler1"),
		testMCPBackend("server1"),
	}

	selected := runner.cfg.BackendSelector(backends)
	if selected.Kind != model.BackendKindMCP {
		t.Errorf("Custom BackendSelector should pick MCP, got %s", selected.Kind)
	}
}

func TestWithToolResolver(t *testing.T) {
	tool := testTool("resolved-tool")
	resolver := func(_ string) (*model.Tool, error) {
		return &tool, nil
	}

	runner := NewRunner(WithToolResolver(resolver))

	if runner.cfg.ToolResolver == nil {
		t.Error("WithToolResolver() did not set ToolResolver")
	}

	resolved, err := runner.cfg.ToolResolver("any-id")
	if err != nil {
		t.Errorf("ToolResolver returned error: %v", err)
	}
	if resolved.Name != "resolved-tool" {
		t.Errorf("ToolResolver returned wrong tool: %v", resolved)
	}
}

func TestWithBackendsResolver(t *testing.T) {
	backends := []model.ToolBackend{
		testMCPBackend("server1"),
	}
	resolver := func(_ string) ([]model.ToolBackend, error) {
		return backends, nil
	}

	runner := NewRunner(WithBackendsResolver(resolver))

	if runner.cfg.BackendsResolver == nil {
		t.Error("WithBackendsResolver() did not set BackendsResolver")
	}

	resolved, err := runner.cfg.BackendsResolver("any-id")
	if err != nil {
		t.Errorf("BackendsResolver returned error: %v", err)
	}
	if len(resolved) != 1 {
		t.Errorf("BackendsResolver returned wrong backends: %v", resolved)
	}
}

func TestNewRunner_Defaults(t *testing.T) {
	runner := NewRunner()

	// Check defaults are applied
	if runner.cfg.Validator == nil {
		t.Error("NewRunner() should set default Validator")
	}
	if runner.cfg.BackendSelector == nil {
		t.Error("NewRunner() should set default BackendSelector")
	}
	if !runner.cfg.ValidateInput {
		t.Error("NewRunner() should set ValidateInput to true")
	}
	if !runner.cfg.ValidateOutput {
		t.Error("NewRunner() should set ValidateOutput to true")
	}
}

func TestNewRunner_ImplementsRunner(t *testing.T) {
	t.Helper()
	var _ Runner = NewRunner()
}

func TestConfig_BackendSelector_MatchesToolindex(t *testing.T) {
	// Verify our default selector matches index.DefaultBackendSelector behavior
	backends := []model.ToolBackend{
		testMCPBackend("server1"),
		testProviderBackend("provider1", "tool1"),
		testLocalBackend("handler1"),
	}

	cfg := Config{}
	cfg.applyDefaults()

	ourResult := cfg.BackendSelector(backends)
	toolindexResult := index.DefaultBackendSelector(backends)

	if ourResult.Kind != toolindexResult.Kind {
		t.Errorf("BackendSelector mismatch: got %s, toolindex got %s",
			ourResult.Kind, toolindexResult.Kind)
	}
}
