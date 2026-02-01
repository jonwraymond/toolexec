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
	mu        sync.Mutex
	messages  []Message
	responder func(Message) Message
	closed    bool
	gateway   *Gateway
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

func TestGatewayListToolExamples(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type: MsgResponse,
			ID:   msg.ID,
			Payload: map[string]any{
				"examples": []any{
					map[string]any{
						"id":          "ex1",
						"title":       "Example 1",
						"description": "First example",
						"resultHint":  "Returns 42",
						"args":        map[string]any{"a": 1},
					},
					map[string]any{
						"id":    "ex2",
						"title": "Example 2",
					},
				},
			},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	examples, err := gw.ListToolExamples(ctx, "test:tool", 5)
	if err != nil {
		t.Fatalf("ListToolExamples() error = %v", err)
	}

	if len(examples) != 2 {
		t.Fatalf("ListToolExamples() returned %d examples, want 2", len(examples))
	}

	if examples[0].ID != "ex1" {
		t.Errorf("examples[0].ID = %q, want %q", examples[0].ID, "ex1")
	}
	if examples[0].Title != "Example 1" {
		t.Errorf("examples[0].Title = %q, want %q", examples[0].Title, "Example 1")
	}
	if examples[0].Description != "First example" {
		t.Errorf("examples[0].Description = %q, want %q", examples[0].Description, "First example")
	}
	if examples[0].ResultHint != "Returns 42" {
		t.Errorf("examples[0].ResultHint = %q, want %q", examples[0].ResultHint, "Returns 42")
	}
	if examples[0].Args["a"] != 1 {
		t.Errorf("examples[0].Args[a] = %v, want 1", examples[0].Args["a"])
	}
}

func TestGatewayListToolExamples_NoExamples(t *testing.T) {
	conn := newAutoRespondConnection(func(msg Message) Message {
		return Message{
			Type:    MsgResponse,
			ID:      msg.ID,
			Payload: map[string]any{},
		}
	})

	gw := New(Config{Connection: conn})
	conn.SetGateway(gw)

	ctx := context.Background()
	examples, err := gw.ListToolExamples(ctx, "test:tool", 5)
	if err != nil {
		t.Fatalf("ListToolExamples() error = %v", err)
	}

	if len(examples) != 0 {
		t.Errorf("ListToolExamples() returned %d examples, want 0", len(examples))
	}
}

func TestGatewayListToolExamples_ConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, err := gw.ListToolExamples(ctx, "test:tool", 5)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("ListToolExamples() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestGatewayRunTool_ConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, err := gw.RunTool(ctx, "test:tool", nil)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("RunTool() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestGatewayRunChain_ConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, _, err := gw.RunChain(ctx, []run.ChainStep{{ToolID: "tool"}})
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("RunChain() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestGatewayListNamespaces_ConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, err := gw.ListNamespaces(ctx)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("ListNamespaces() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestGatewayDescribeTool_ConnectionClosed(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	if err := gw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	ctx := context.Background()
	_, err := gw.DescribeTool(ctx, "test:tool", tooldoc.DetailFull)
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("DescribeTool() after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestJsonCodec_Encode(t *testing.T) {
	codec := &jsonCodec{}
	msg := Message{
		Type:    MsgSearchTools,
		ID:      "test-123",
		Payload: map[string]any{"key": "value"},
	}

	data, err := codec.Encode(msg)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("Encode() returned empty data")
	}
}

func TestJsonCodec_Decode(t *testing.T) {
	codec := &jsonCodec{}
	data := []byte(`{"type":"search_tools","id":"test-123","payload":{"key":"value"}}`)

	msg, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if msg.Type != MsgSearchTools {
		t.Errorf("Decode().Type = %q, want %q", msg.Type, MsgSearchTools)
	}
	if msg.ID != "test-123" {
		t.Errorf("Decode().ID = %q, want %q", msg.ID, "test-123")
	}
}

func TestJsonCodec_Decode_Invalid(t *testing.T) {
	codec := &jsonCodec{}
	data := []byte(`{invalid json}`)

	_, err := codec.Decode(data)
	if err == nil {
		t.Error("Decode() should return error for invalid JSON")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"exists":    "value",
		"notString": 42,
	}

	if got := getString(m, "exists"); got != "value" {
		t.Errorf("getString(exists) = %q, want %q", got, "value")
	}

	if got := getString(m, "notString"); got != "" {
		t.Errorf("getString(notString) = %q, want empty string", got)
	}

	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString(missing) = %q, want empty string", got)
	}
}

func TestGatewayDeliverResponse_UnknownID(t *testing.T) {
	conn := newMockConnection()
	gw := New(Config{Connection: conn})

	err := gw.DeliverResponse(Message{
		Type: MsgResponse,
		ID:   "unknown-id",
	})
	if err == nil {
		t.Error("DeliverResponse() should return error for unknown ID")
	}
}
