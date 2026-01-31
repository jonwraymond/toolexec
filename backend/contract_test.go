package backend

import (
	"context"
	"testing"

	"github.com/jonwraymond/toolfoundation/model"
)

type streamingBackend struct{}

func (s *streamingBackend) Kind() string  { return "streaming" }
func (s *streamingBackend) Name() string  { return "streaming" }
func (s *streamingBackend) Enabled() bool { return true }
func (s *streamingBackend) ListTools(_ context.Context) ([]model.Tool, error) {
	return nil, nil
}
func (s *streamingBackend) Execute(_ context.Context, _ string, _ map[string]any) (any, error) {
	return nil, nil
}
func (s *streamingBackend) Start(_ context.Context) error { return nil }
func (s *streamingBackend) Stop() error                   { return nil }
func (s *streamingBackend) ExecuteStream(_ context.Context, _ string, _ map[string]any) (<-chan any, error) {
	ch := make(chan any)
	close(ch)
	return ch, nil
}

type configurableBackend struct {
	streamingBackend
}

func (c *configurableBackend) Configure(_ []byte) error { return nil }

func TestBackendContracts(t *testing.T) {
	var _ StreamingBackend = (*streamingBackend)(nil)
	var _ ConfigurableBackend = (*configurableBackend)(nil)

	b := &streamingBackend{}
	ch, err := b.ExecuteStream(context.Background(), "tool", nil)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	if ch == nil {
		t.Fatalf("ExecuteStream should return non-nil channel when err is nil")
	}
}
