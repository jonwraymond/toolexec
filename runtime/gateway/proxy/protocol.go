// Package proxy provides a gateway that implements ToolGateway
// by serializing requests over a connection (for cross-process/container communication).
package proxy

import "context"

// MessageType identifies the type of protocol message.
type MessageType string

const (
	// Request message types
	MsgSearchTools      MessageType = "search_tools"
	MsgListNamespaces   MessageType = "list_namespaces"
	MsgDescribeTool     MessageType = "describe_tool"
	MsgListToolExamples MessageType = "list_tool_examples"
	MsgRunTool          MessageType = "run_tool"
	MsgRunChain         MessageType = "run_chain"

	// Response message type
	MsgResponse MessageType = "response"
	MsgError    MessageType = "error"
)

// Message is the wire protocol envelope for gateway operations.
type Message struct {
	Type    MessageType    `json:"type"`
	ID      string         `json:"id"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Connection defines the interface for sending and receiving messages.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: Send/Receive must honor cancellation/deadlines.
// - Errors: return ErrConnectionClosed when closed.
type Connection interface {
	// Send sends a message and waits for acknowledgment.
	Send(ctx context.Context, msg Message) error

	// Receive waits for and returns the next message.
	Receive(ctx context.Context) (Message, error)

	// Close closes the connection.
	Close() error
}

// Codec defines the interface for message serialization.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Errors: Encode/Decode must return errors for invalid data.
type Codec interface {
	// Encode encodes a message to bytes.
	Encode(msg Message) ([]byte, error)

	// Decode decodes bytes to a message.
	Decode(data []byte) (Message, error)
}
