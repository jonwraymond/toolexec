package code

import (
	"context"
	"sync"

	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/toolfoundation/model"
	"github.com/jonwraymond/toolexec/run"
)

// mockIndex implements index.Index for testing.
type mockIndex struct {
	mu sync.Mutex

	// Configurable returns
	searchResult     []index.Summary
	searchErr        error
	namespacesResult []string
	getToolResult    model.Tool
	getToolBackend   model.ToolBackend
	getToolErr       error

	// Call tracking
	searchCalls     []searchCall
	getToolCalls    []string
	namespacesCalls int
}

type searchCall struct {
	query string
	limit int
}

func (m *mockIndex) Search(query string, limit int) ([]index.Summary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchCalls = append(m.searchCalls, searchCall{query, limit})
	return m.searchResult, m.searchErr
}

func (m *mockIndex) SearchPage(query string, limit int, _ string) ([]index.Summary, string, error) {
	results, err := m.Search(query, limit)
	return results, "", err
}

func (m *mockIndex) ListNamespaces() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.namespacesCalls++
	return m.namespacesResult, nil
}

func (m *mockIndex) ListNamespacesPage(limit int, _ string) ([]string, string, error) {
	results, err := m.ListNamespaces()
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, "", err
}

func (m *mockIndex) GetTool(id string) (model.Tool, model.ToolBackend, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getToolCalls = append(m.getToolCalls, id)
	return m.getToolResult, m.getToolBackend, m.getToolErr
}

func (m *mockIndex) GetAllBackends(_ string) ([]model.ToolBackend, error) {
	return nil, nil
}

func (m *mockIndex) RegisterTool(_ model.Tool, _ model.ToolBackend) error {
	return nil
}

func (m *mockIndex) RegisterTools(_ []index.ToolRegistration) error {
	return nil
}

func (m *mockIndex) RegisterToolsFromMCP(_ string, _ []model.Tool) error {
	return nil
}

func (m *mockIndex) UnregisterBackend(_ string, _ model.BackendKind, _ string) error {
	return nil
}

// mockStore implements tooldoc.Store for testing.
type mockStore struct {
	mu sync.Mutex

	// Configurable returns
	describeResult tooldoc.ToolDoc
	describeErr    error
	examplesResult []tooldoc.ToolExample
	examplesErr    error

	// Call tracking
	describeCalls []describeCall
	examplesCalls []examplesCall
}

type describeCall struct {
	id    string
	level tooldoc.DetailLevel
}

type examplesCall struct {
	id          string
	maxExamples int
}

func (m *mockStore) DescribeTool(id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.describeCalls = append(m.describeCalls, describeCall{id, level})
	return m.describeResult, m.describeErr
}

func (m *mockStore) ListExamples(id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.examplesCalls = append(m.examplesCalls, examplesCall{id, maxExamples})
	return m.examplesResult, m.examplesErr
}

// mockRunner implements run.Runner for testing.
type mockRunner struct {
	mu sync.Mutex

	// Configurable returns
	runResult    run.RunResult
	runErr       error
	chainResult  run.RunResult
	chainSteps   []run.StepResult
	chainErr     error
	streamEvents []run.StreamEvent
	streamErr    error

	// Call tracking
	runCalls   []runCall
	chainCalls [][]run.ChainStep
}

type runCall struct {
	ctx    context.Context
	toolID string
	args   map[string]any
}

func (m *mockRunner) Run(ctx context.Context, toolID string, args map[string]any) (run.RunResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runCalls = append(m.runCalls, runCall{ctx, toolID, args})
	return m.runResult, m.runErr
}

func (m *mockRunner) RunStream(_ context.Context, _ string, _ map[string]any) (<-chan run.StreamEvent, error) {
	ch := make(chan run.StreamEvent, len(m.streamEvents)+1)
	for _, e := range m.streamEvents {
		ch <- e
	}
	close(ch)
	return ch, m.streamErr
}

func (m *mockRunner) RunChain(_ context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chainCalls = append(m.chainCalls, steps)
	return m.chainResult, m.chainSteps, m.chainErr
}

// mockEngine implements Engine for testing.
type mockEngine struct {
	mu sync.Mutex

	// Configurable returns
	executeResult ExecuteResult
	executeErr    error

	// Call tracking
	executeCalls []executeCall
}

type executeCall struct {
	ctx    context.Context
	params ExecuteParams
	tools  Tools
}

func (m *mockEngine) Execute(ctx context.Context, params ExecuteParams, tools Tools) (ExecuteResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalls = append(m.executeCalls, executeCall{ctx, params, tools})
	return m.executeResult, m.executeErr
}

// mockLogger implements Logger for testing.
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *mockLogger) Logf(_ string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, "")
}
