package runtime

import (
	"context"
	"errors"
	"testing"
)

// mockBackend is a minimal Backend implementation for testing
type mockBackend struct {
	kind       BackendKind
	executeErr error
	result     ExecuteResult
}

func (m *mockBackend) Kind() BackendKind {
	return m.kind
}

func (m *mockBackend) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return ExecuteResult{}, err
	}

	// Check context
	if ctx.Err() != nil {
		return ExecuteResult{}, ctx.Err()
	}

	if m.executeErr != nil {
		return ExecuteResult{}, m.executeErr
	}

	// Return configured result with backend info
	result := m.result
	result.Backend = BackendInfo{Kind: m.kind}
	return result, nil
}

// Test that mockBackend satisfies the interface
func TestMockBackendImplementsInterface(t *testing.T) {
	t.Helper()
	var _ Backend = (*mockBackend)(nil)
}

// Test mock backend contract compliance
func TestMockBackendContract(t *testing.T) {
	RunBackendContractTests(t, BackendContract{
		NewBackend: func() Backend {
			return &mockBackend{
				kind: BackendUnsafeHost,
				result: ExecuteResult{
					Value: "hello",
				},
			}
		},
		NewGateway: func() ToolGateway {
			return &mockToolGateway{}
		},
		ExpectedKind:       BackendUnsafeHost,
		SkipExecutionTests: true, // Mock backend doesn't execute real code
	})
}

// errBackend always returns an error
type errBackend struct {
	kind BackendKind
	err  error
}

func (e *errBackend) Kind() BackendKind {
	return e.kind
}

func (e *errBackend) Execute(_ context.Context, req ExecuteRequest) (ExecuteResult, error) {
	if err := req.Validate(); err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{}, e.err
}

func TestErrBackend(t *testing.T) {
	expectedErr := errors.New("test error")
	b := &errBackend{
		kind: BackendDocker,
		err:  expectedErr,
	}

	ctx := context.Background()
	req := ExecuteRequest{
		Code:    "test",
		Gateway: &mockToolGateway{},
	}

	_, err := b.Execute(ctx, req)
	if err != expectedErr {
		t.Errorf("errBackend.Execute() error = %v, want %v", err, expectedErr)
	}
}
