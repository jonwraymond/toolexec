package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/tooldoc"
	"github.com/jonwraymond/toolexec/run"
)

// Errors for proxy gateway operations.
var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrTimeout          = errors.New("request timeout")
	ErrProtocol         = errors.New("protocol error")
)

// Config configures a proxy gateway.
type Config struct {
	// Connection is the underlying connection to use.
	Connection Connection

	// Codec is the message codec to use. If nil, JSON is used.
	Codec Codec
}

// Gateway implements ToolGateway by serializing requests over a connection.
// This is used when the gateway needs to communicate across process boundaries,
// such as when code runs in a Docker container.
type Gateway struct {
	conn      Connection
	codec     Codec
	requestID atomic.Uint64
	pending   sync.Map // map[string]chan Message
	closed    atomic.Bool
	closeMu   sync.Mutex
}

// New creates a new proxy gateway with the given configuration.
func New(cfg Config) *Gateway {
	codec := cfg.Codec
	if codec == nil {
		codec = &jsonCodec{}
	}

	return &Gateway{
		conn:  cfg.Connection,
		codec: codec,
	}
}

// SearchTools sends a search request over the connection.
func (g *Gateway) SearchTools(ctx context.Context, query string, limit int) ([]index.Summary, error) {
	if g.closed.Load() {
		return nil, ErrConnectionClosed
	}

	resp, err := g.request(ctx, MsgSearchTools, map[string]any{
		"query": query,
		"limit": limit,
	})
	if err != nil {
		return nil, err
	}

	// Decode response
	results, ok := resp.Payload["results"].([]any)
	if !ok {
		return nil, nil
	}

	summaries := make([]index.Summary, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			summary := index.Summary{
				ID:               getString(m, "id"),
				Name:             getString(m, "name"),
				Namespace:        getString(m, "namespace"),
				ShortDescription: getString(m, "shortDescription"),
			}
			if tags, ok := m["tags"].([]any); ok {
				for _, t := range tags {
					if s, ok := t.(string); ok {
						summary.Tags = append(summary.Tags, s)
					}
				}
			}
			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}

// ListNamespaces sends a list namespaces request over the connection.
func (g *Gateway) ListNamespaces(ctx context.Context) ([]string, error) {
	if g.closed.Load() {
		return nil, ErrConnectionClosed
	}

	resp, err := g.request(ctx, MsgListNamespaces, nil)
	if err != nil {
		return nil, err
	}

	// Decode response
	results, ok := resp.Payload["namespaces"].([]any)
	if !ok {
		return nil, nil
	}

	namespaces := make([]string, 0, len(results))
	for _, r := range results {
		if s, ok := r.(string); ok {
			namespaces = append(namespaces, s)
		}
	}

	return namespaces, nil
}

// DescribeTool sends a describe tool request over the connection.
func (g *Gateway) DescribeTool(ctx context.Context, id string, level tooldoc.DetailLevel) (tooldoc.ToolDoc, error) {
	if g.closed.Load() {
		return tooldoc.ToolDoc{}, ErrConnectionClosed
	}

	resp, err := g.request(ctx, MsgDescribeTool, map[string]any{
		"id":    id,
		"level": string(level),
	})
	if err != nil {
		return tooldoc.ToolDoc{}, err
	}

	// Decode response - simplified version
	doc := tooldoc.ToolDoc{
		Summary: getString(resp.Payload, "summary"),
		Notes:   getString(resp.Payload, "notes"),
	}

	return doc, nil
}

// ListToolExamples sends a list examples request over the connection.
func (g *Gateway) ListToolExamples(ctx context.Context, id string, maxExamples int) ([]tooldoc.ToolExample, error) {
	if g.closed.Load() {
		return nil, ErrConnectionClosed
	}

	resp, err := g.request(ctx, MsgListToolExamples, map[string]any{
		"id":  id,
		"max": maxExamples,
	})
	if err != nil {
		return nil, err
	}

	// Decode response - simplified version
	results, ok := resp.Payload["examples"].([]any)
	if !ok {
		return nil, nil
	}

	examples := make([]tooldoc.ToolExample, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			ex := tooldoc.ToolExample{
				ID:          getString(m, "id"),
				Title:       getString(m, "title"),
				Description: getString(m, "description"),
				ResultHint:  getString(m, "resultHint"),
			}
			if args, ok := m["args"].(map[string]any); ok {
				ex.Args = args
			}
			examples = append(examples, ex)
		}
	}

	return examples, nil
}

// RunTool sends a run tool request over the connection.
func (g *Gateway) RunTool(ctx context.Context, id string, args map[string]any) (run.RunResult, error) {
	if g.closed.Load() {
		return run.RunResult{}, ErrConnectionClosed
	}

	resp, err := g.request(ctx, MsgRunTool, map[string]any{
		"id":   id,
		"args": args,
	})
	if err != nil {
		return run.RunResult{}, err
	}

	// Decode response - simplified version
	result := run.RunResult{
		Structured: resp.Payload["structured"],
	}

	return result, nil
}

// RunChain sends a run chain request over the connection.
func (g *Gateway) RunChain(ctx context.Context, steps []run.ChainStep) (run.RunResult, []run.StepResult, error) {
	if g.closed.Load() {
		return run.RunResult{}, nil, ErrConnectionClosed
	}

	if len(steps) == 0 {
		return run.RunResult{}, nil, nil
	}

	// Serialize steps
	stepsData := make([]map[string]any, len(steps))
	for i, step := range steps {
		stepsData[i] = map[string]any{
			"toolId":      step.ToolID,
			"args":        step.Args,
			"usePrevious": step.UsePrevious,
		}
	}

	resp, err := g.request(ctx, MsgRunChain, map[string]any{
		"steps": stepsData,
	})
	if err != nil {
		return run.RunResult{}, nil, err
	}

	// Decode response - simplified version
	result := run.RunResult{
		Structured: resp.Payload["structured"],
	}

	// Decode step results if present
	var stepResults []run.StepResult
	if results, ok := resp.Payload["stepResults"].([]any); ok {
		for _, r := range results {
			if m, ok := r.(map[string]any); ok {
				sr := run.StepResult{
					ToolID: getString(m, "toolId"),
					Result: run.RunResult{
						Structured: m["structured"],
					},
				}
				stepResults = append(stepResults, sr)
			}
		}
	}

	return result, stepResults, nil
}

// Close closes the underlying connection.
func (g *Gateway) Close() error {
	g.closeMu.Lock()
	defer g.closeMu.Unlock()

	if g.closed.Load() {
		return nil
	}

	g.closed.Store(true)
	return g.conn.Close()
}

// request sends a request and waits for the response.
func (g *Gateway) request(ctx context.Context, msgType MessageType, payload map[string]any) (Message, error) {
	id := fmt.Sprintf("%d", g.requestID.Add(1))

	msg := Message{
		Type:    msgType,
		ID:      id,
		Payload: payload,
	}

	// Create response channel
	respCh := make(chan Message, 1)
	g.pending.Store(id, respCh)
	defer g.pending.Delete(id)

	// Send request
	if err := g.conn.Send(ctx, msg); err != nil {
		return Message{}, err
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return Message{}, ctx.Err()
	case resp := <-respCh:
		if resp.Type == MsgError {
			errMsg := getString(resp.Payload, "error")
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return Message{}, errors.New(errMsg)
		}
		return resp, nil
	}
}

// DeliverResponse delivers a response to a pending request.
// This is called by the connection handler when a response is received.
func (g *Gateway) DeliverResponse(msg Message) error {
	ch, ok := g.pending.Load(msg.ID)
	if !ok {
		return fmt.Errorf("%w: no pending request for ID %s", ErrProtocol, msg.ID)
	}

	select {
	case ch.(chan Message) <- msg:
		return nil
	default:
		return fmt.Errorf("%w: response channel full for ID %s", ErrProtocol, msg.ID)
	}
}

// getString safely extracts a string from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// jsonCodec implements Codec using JSON encoding.
type jsonCodec struct{}

func (c *jsonCodec) Encode(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func (c *jsonCodec) Decode(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
