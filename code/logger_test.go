package code

import "testing"

func TestLogger_Interface(t *testing.T) {
	t.Helper()
	// Verify Logger interface has Logf method with correct signature
	var _ Logger = (*testLogger)(nil)
}

// testLogger is a test implementation of Logger
type testLogger struct {
}

func (l *testLogger) Logf(_ string, _ ...any) {
	// Implementation for testing
}
