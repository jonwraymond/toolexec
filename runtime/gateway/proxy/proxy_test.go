package proxy

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
	"github.com/jonwraymond/toolexec/runtime"
)

// mockConnection implements Connection for testing
type mockConnection struct {
	mu        sync.Mutex
	messages  []Message
	responses map[string]Message
	sendErr   error
	recvErr   error
	closed    bool
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		responses: make(map[string]Message),
	}
}

func (c *mockConnection) Send(_ context.Context, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConnectionClosed
	}

	if c.sendErr != nil {
		return c.sendErr
	}

	c.messages = append(c.messages, msg)

	// If there's a response queued, deliver it
	if resp, ok := c.responses[msg.ID]; ok {
		// The gateway will call DeliverResponse
		_ = resp
	}

	return nil
}

func (c *mockConnection) Receive(_ context.Context) (Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return Message{}, ErrConnectionClosed
	}

	if c.recvErr != nil {
		return Message{}, c.recvErr
	}

	return Message{}, errors.New("no message")
}

func (c *mockConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConnection) SetResponse(id string, resp Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses[id] = resp
}

// autoRespondConnection automatically responds to requests
type autoRespondConnection struct {
	mu         sync.Mutex
	messages   []Message
	responder  func(Message) Message
	closed     bool
	gateway    *Gateway
}

func newAutoRespondConnection(responder func(Message) Message) *autoRespondConnection {
	return &autoRespondConnection{
		responder: responder,
	}
}

func (c *autoRespondConnection) SetGateway(g *Gateway) {
	c.gateway = g
}

func (c *autoRespondConnection) Send(_ context.Context, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrConnectionClosed
	}

	c.messages = append(c.messages, msg)

	// Auto-respond
	if c.responder != nil && c.gateway != nil {
		resp := c.responder(msg)
		go func() {
			_ = c.gateway.DeliverResponse(resp)
		}()
	}

	return nil
}

func (c *autoRespondConnection) Receive(_ context.Context) (Message, error) {
	return Message{}, errors.New("not implemented")
}

func (c *autoRespondConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// TestGatewayImplementsInterface verifies Gateway satisfies ToolGateway
func TestGatewayImplementsInterface(t *testing.T) {
	t.Helper()
	var _ runtime.ToolGateway = (*Gateway)(nil)
}

func TestGatewaySearchTools(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"results": []any{
					map[string]any{
						"id":               "test:tool",
						"name":             "tool",
						"namespace":        "test",
						"shortDescription": "A test tool",
						"tags":             []any{"tag1", "tag2"},
					},
				},
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	results, err := gw.SearchTools(ctx, "test", 10)
	if err != nil {
		t.Fatalf("SearchTools() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchTools() returned %d results, want 1", len(results))
	}
	if results[0].ID != "test:tool" {
		t.Errorf("SearchTools()[0].ID = %q, want %q", results[0].ID, "test:tool")
	}
}

func TestGatewayListNamespaces(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"namespaces": []any{"ns1", "ns2"},
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	results, err := gw.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("ListNamespaces() returned %d results, want 2", len(results))
	}
}

func TestGatewayDescribeTool(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"summary": "Test tool summary",
				"notes":   "Test notes",
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	doc, err := gw.DescribeTool(ctx, "test:tool", tooldoc.DetailFull)
	if err != nil {
		t.Fatalf("DescribeTool() error = %v", err)
	}

	if doc.Summary != "Test tool summary" {
		t.Errorf("DescribeTool().Summary = %q, want %q", doc.Summary, "Test tool summary")
	}
}

func TestGatewayRunTool(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"structured": "result",
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	result, err := gw.RunTool(ctx, "test:tool", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("RunTool() error = %v", err)
	}

	if result.Structured != "result" {
		t.Errorf("RunTool().Structured = %v, want %v", result.Structured, "result")
	}
}

func TestGatewayRunChain(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"structured": "chain_result",
				"stepResults": []any{
					map[string]any{
						"toolId":     "step1",
						"structured": "step1_result",
					},
				},
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	steps := []run.ChainStep{
		{ToolID: "step1"},
	}
	result, stepResults, err := gw.RunChain(ctx, steps)
	if err != nil {
		t.Fatalf("RunChain() error = %v", err)
	}

	if result.Structured != "chain_result" {
		t.Errorf("RunChain().Structured = %v, want %v", result.Structured, "chain_result")
	}
	if len(stepResults) != 1 {
		t.Errorf("RunChain() returned %d step results, want 1", len(stepResults))
	}
}

func TestGatewayRunChainEmpty(t *testing.T) {
	conn := newAutoRespondConnection(nil)
	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	_, stepResults, err := gw.RunChain(ctx, []run.ChainStep{})
	if err != nil {
		t.Fatalf("RunChain() with empty steps error = %v", err)
	}
	if len(stepResults) != 0 {
		t.Errorf("RunChain() with empty steps returned %d results", len(stepResults))
	}
}

func TestGatewayConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	// Close the gateway
	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, err := gw.SearchTools(ctx, "test", 10)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("SearchTools() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestGatewayErrorResponse(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgError,
			ID:   msg.ID,
			Payload: map[string]any{
				"error": "tool not found",
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	_, err := gw.DescribeTool(ctx, "nonexistent:tool", tooldoc.DetailSummary)
	if err == nil {
		t.Error("DescribeTool() should return error for error response")
	}
	if err.Error() != "tool not found" {
		t.Errorf("DescribeTool() error = %q, want %q", err.Error(), "tool not found")
	}
}
