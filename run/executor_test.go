package run

import "testing"

func TestMockMCPExecutor_ImplementsInterface(t *testing.T) {
	t.Helper()
	var _ MCPExecutor = newMockMCPExecutor()
}

func TestMockProviderExecutor_ImplementsInterface(t *testing.T) {
	t.Helper()
	var _ ProviderExecutor = newMockProviderExecutor()
}

func TestMockLocalRegistry_ImplementsInterface(t *testing.T) {
	t.Helper()
	var _ LocalRegistry = newMockLocalRegistry()
}
