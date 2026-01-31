package run

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// -----------------------------------------------------------------------------
// Mock MCP Executor
// -----------------------------------------------------------------------------

// mockMCPExecutor is a test implementation of MCPExecutor.
// It captures calls and returns configurable results.
type mockMCPExecutor struct {
	mu sync.Mutex

	// CallToolResult is returned by CallTool
	CallToolResult *mcp.CallToolResult
	CallToolErr    error

	// CallToolStreamChan is returned by CallToolStream
	CallToolStreamChan chan StreamEvent
	CallToolStreamErr  error

	// LastCallToolParams captures the last CallTool call
	LastServerName string
	LastParams     *mcp.CallToolParams
	CallCount      int
}

func newMockMCPExecutor() *mockMCPExecutor {
	return &mockMCPExecutor{}
}

func (m *mockMCPExecutor) CallTool(_ context.Context, serverName string, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastServerName = serverName
	m.LastParams = params
	m.CallCount++
	return m.CallToolResult, m.CallToolErr
}

func (m *mockMCPExecutor) CallToolStream(_ context.Context, serverName string, params *mcp.CallToolParams) (<-chan StreamEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastServerName = serverName
	m.LastParams = params
	m.CallCount++
	if m.CallToolStreamErr != nil {
		return nil, m.CallToolStreamErr
	}
	return m.CallToolStreamChan, nil
}

// -----------------------------------------------------------------------------
// Mock Provider Executor
// -----------------------------------------------------------------------------

// mockProviderExecutor is a test implementation of ProviderExecutor.
type mockProviderExecutor struct {
	mu sync.Mutex

	// CallToolResult is returned by CallTool
	CallToolResult any
	CallToolErr    error

	// CallToolStreamChan is returned by CallToolStream
	CallToolStreamChan chan StreamEvent
	CallToolStreamErr  error

	// Captures
	LastProviderID string
	LastToolID     string
	LastArgs       map[string]any
	CallCount      int
}

func newMockProviderExecutor() *mockProviderExecutor {
	return &mockProviderExecutor{}
}

func (m *mockProviderExecutor) CallTool(_ context.Context, providerID, toolID string, args map[string]any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastProviderID = providerID
	m.LastToolID = toolID
	m.LastArgs = args
	m.CallCount++
	return m.CallToolResult, m.CallToolErr
}

func (m *mockProviderExecutor) CallToolStream(_ context.Context, providerID, toolID string, args map[string]any) (<-chan StreamEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastProviderID = providerID
	m.LastToolID = toolID
	m.LastArgs = args
	m.CallCount++
	if m.CallToolStreamErr != nil {
		return nil, m.CallToolStreamErr
	}
	return m.CallToolStreamChan, nil
}

// -----------------------------------------------------------------------------
// Mock Local Registry
// -----------------------------------------------------------------------------

// mockLocalRegistry is a map-based LocalRegistry implementation.
type mockLocalRegistry struct {
	handlers map[string]LocalHandler
}

func newMockLocalRegistry() *mockLocalRegistry {
	return &mockLocalRegistry{
		handlers: make(map[string]LocalHandler),
	}
}

func (m *mockLocalRegistry) Get(name string) (LocalHandler, bool) {
	h, ok := m.handlers[name]
	return h, ok
}

func (m *mockLocalRegistry) Register(name string, handler LocalHandler) {
	m.handlers[name] = handler
}

// -----------------------------------------------------------------------------
// Mock Index
// -----------------------------------------------------------------------------

// mockIndex is a test implementation of index.Index.
type mockIndex struct {
	mu sync.RWMutex

	// Tools maps tool ID to tool
	Tools map[string]model.Tool

	// Backends maps tool ID to list of backends
	Backends map[string][]model.ToolBackend

	// DefaultBackends maps tool ID to default backend (returned by GetTool)
	DefaultBackends map[string]model.ToolBackend

	// Errors to return
	GetToolErr        error
	GetAllBackendsErr error
}

func newMockIndex() *mockIndex {
	return &mockIndex{
		Tools:           make(map[string]model.Tool),
		Backends:        make(map[string][]model.ToolBackend),
		DefaultBackends: make(map[string]model.ToolBackend),
	}
}

func mustRegisterTool(t *testing.T, idx *mockIndex, tool model.Tool, backend model.ToolBackend) {
	t.Helper()
	if err := idx.RegisterTool(tool, backend); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
}

func (m *mockIndex) RegisterTool(tool model.Tool, backend model.ToolBackend) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := tool.ToolID()
	m.Tools[id] = tool
	m.Backends[id] = append(m.Backends[id], backend)
	if _, ok := m.DefaultBackends[id]; !ok {
		m.DefaultBackends[id] = backend
	}
	return nil
}

func (m *mockIndex) RegisterTools(regs []index.ToolRegistration) error {
	for _, reg := range regs {
		if err := m.RegisterTool(reg.Tool, reg.Backend); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockIndex) RegisterToolsFromMCP(serverName string, tools []model.Tool) error {
	backend := model.ToolBackend{
		Kind: model.BackendKindMCP,
		MCP:  &model.MCPBackend{ServerName: serverName},
	}
	for _, tool := range tools {
		if err := m.RegisterTool(tool, backend); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockIndex) UnregisterBackend(_ string, _ model.BackendKind, _ string) error {
	return nil
}

func (m *mockIndex) GetTool(id string) (model.Tool, model.ToolBackend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetToolErr != nil {
		return model.Tool{}, model.ToolBackend{}, m.GetToolErr
	}

	tool, ok := m.Tools[id]
	if !ok {
		return model.Tool{}, model.ToolBackend{}, index.ErrNotFound
	}

	backend := m.DefaultBackends[id]
	return tool, backend, nil
}

func (m *mockIndex) GetAllBackends(id string) ([]model.ToolBackend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetAllBackendsErr != nil {
		return nil, m.GetAllBackendsErr
	}

	backends, ok := m.Backends[id]
	if !ok {
		return nil, index.ErrNotFound
	}
	return backends, nil
}

func (m *mockIndex) Search(_ string, _ int) ([]index.Summary, error) {
	return nil, nil
}

func (m *mockIndex) SearchPage(query string, limit int, _ string) ([]index.Summary, string, error) {
	results, err := m.Search(query, limit)
	return results, "", err
}

func (m *mockIndex) ListNamespaces() ([]string, error) {
	return nil, nil
}

func (m *mockIndex) ListNamespacesPage(_ int, _ string) ([]string, string, error) {
	results, err := m.ListNamespaces()
	return results, "", err
}

// -----------------------------------------------------------------------------
// Mock Validator
// -----------------------------------------------------------------------------

// mockValidator is a test implementation of model.SchemaValidator.
type mockValidator struct {
	ValidateErr       error
	ValidateInputErr  error
	ValidateOutputErr error

	// Track calls
	ValidateCalls      int
	ValidateInputCalls int
	OutputCalls        int
}

func newMockValidator() *mockValidator {
	return &mockValidator{}
}

func (m *mockValidator) Validate(_ any, _ any) error {
	m.ValidateCalls++
	return m.ValidateErr
}

func (m *mockValidator) ValidateInput(_ *model.Tool, _ any) error {
	m.ValidateInputCalls++
	return m.ValidateInputErr
}

func (m *mockValidator) ValidateOutput(_ *model.Tool, _ any) error {
	m.OutputCalls++
	return m.ValidateOutputErr
}

// -----------------------------------------------------------------------------
// Test Helper Functions
// -----------------------------------------------------------------------------

// testTool creates a test tool with minimal required fields.
func testTool(name string) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name: name,
			InputSchema: map[string]any{
				"type": "object",
			},
		},
	}
}

// testToolWithNamespace creates a test tool with a namespace.
func testToolWithNamespace(namespace, name string) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name: name,
			InputSchema: map[string]any{
				"type": "object",
			},
		},
		Namespace: namespace,
	}
}

// testToolWithOutputSchema creates a test tool with both input and output schemas.
func testToolWithOutputSchema(name string) model.Tool {
	return model.Tool{
		Tool: mcp.Tool{
			Name: name,
			InputSchema: map[string]any{
				"type": "object",
			},
			OutputSchema: map[string]any{
				"type": "object",
			},
		},
	}
}

// testMCPBackend creates an MCP backend.
func testMCPBackend(serverName string) model.ToolBackend {
	return model.ToolBackend{
		Kind: model.BackendKindMCP,
		MCP:  &model.MCPBackend{ServerName: serverName},
	}
}

// testProviderBackend creates a provider backend.
func testProviderBackend(providerID, toolID string) model.ToolBackend {
	return model.ToolBackend{
		Kind: model.BackendKindProvider,
		Provider: &model.ProviderBackend{
			ProviderID: providerID,
			ToolID:     toolID,
		},
	}
}

// testLocalBackend creates a local backend.
func testLocalBackend(name string) model.ToolBackend {
	return model.ToolBackend{
		Kind:  model.BackendKindLocal,
		Local: &model.LocalBackend{Name: name},
	}
}

// testMCPResult creates a simple MCP CallToolResult with text content.
func testMCPResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// testMCPResultStructured creates an MCP CallToolResult with structured content.
func testMCPResultStructured(structured any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		StructuredContent: structured,
	}
}

// testMCPResultJSON creates an MCP CallToolResult with JSON text content.
func testMCPResultJSON(v any) *mcp.CallToolResult {
	data, _ := json.Marshal(v)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}
}

// -----------------------------------------------------------------------------
// Common Test Errors
// -----------------------------------------------------------------------------

var (
	errTest = errors.New("test error")
)
