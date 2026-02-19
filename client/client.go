// Package client provides the Qumbed developer SDK: a simple API to publish and
// subscribe with channel-based message delivery and context.Context for timeouts.
package client

import (
	"context"
	"errors"
	"sync"

	"github.com/SWAI-Ltd/Qumbed/internal/crypto"
	"github.com/SWAI-Ltd/Qumbed/internal/mesh"
	"github.com/SWAI-Ltd/Qumbed/internal/proto"
)

const (
	// DefaultMessageBuffer is the buffer size for the Messages() channel.
	DefaultMessageBuffer = 64
)

// ErrClosed is returned when using a client after Close.
var ErrClosed = errors.New("client closed")

// ReceivedMessage is a message delivered to the subscriber.
type ReceivedMessage struct {
	Topic   string
	Payload []byte
}

// Config configures the Qumbed client.
type Config struct {
	// Addr is the local QUIC listen address (e.g. ":0" for any port).
	Addr string
	// NodeID is a human-readable identifier for this node (e.g. "sensor-1").
	NodeID string
	// RelayAddr is the relay address (e.g. "localhost:6121"). Required for pub/sub via relay.
	RelayAddr string
	// DisableDiscovery disables mDNS (set true in containers or relay-only mode).
	DisableDiscovery bool
	// MessageBuffer sets the capacity of Messages() channel; 0 uses DefaultMessageBuffer.
	MessageBuffer int
}

// Client is the developer-facing Qumbed client. Use Publish/Subscribe and read from Messages().
type Client struct {
	node   *mesh.Node
	msgs   chan ReceivedMessage
	closed bool
	mu     sync.Mutex
}

// New creates a new Qumbed client. Call Subscribe to receive messages; read them from Messages().
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		cfg.Addr = ":0"
	}
	buf := cfg.MessageBuffer
	if buf <= 0 {
		buf = DefaultMessageBuffer
	}
	msgs := make(chan ReceivedMessage, buf)
	node, err := mesh.NewNode(ctx, mesh.Config{
		Addr:              cfg.Addr,
		NodeID:             cfg.NodeID,
		RelayAddr:         cfg.RelayAddr,
		DisableDiscovery:  cfg.DisableDiscovery,
		OnMessage: func(topic string, payload []byte) {
			select {
			case msgs <- ReceivedMessage{Topic: topic, Payload: payload}:
			default:
				// channel full; drop or could log
			}
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{node: node, msgs: msgs}, nil
}

// Publish sends a message to a topic. Payload must match schemaID (e.g. proto.SchemaTemperature).
// recipientPub is the subscriber's public key for E2EE; use nil to publish to self (demo only).
func (c *Client) Publish(ctx context.Context, topic, schemaID string, payload []byte, recipientPub *[crypto.PublicKeySize]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	return c.node.Publish(ctx, topic, schemaID, payload, recipientPub)
}

// Subscribe registers for a topic and starts receiving messages on Messages().
// Must be called with a non-empty RelayAddr in Config.
func (c *Client) Subscribe(ctx context.Context, topic, schemaID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	return c.node.Subscribe(ctx, topic, schemaID)
}

// Messages returns the channel of received messages. Read until the client is closed.
func (c *Client) Messages() <-chan ReceivedMessage {
	return c.msgs
}

// PublicKey returns this client's public key (hex for sharing with publishers).
func (c *Client) PublicKey() *[crypto.PublicKeySize]byte {
	return c.node.PublicKey()
}

// Addr returns the local QUIC listen address.
func (c *Client) Addr() string {
	return c.node.Addr()
}

// Close shuts down the client and closes the Messages() channel.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	err := c.node.Close()
	close(c.msgs)
	return err
}

// Schema constants for convenience (re-export from proto).
var (
	SchemaTemperature = proto.SchemaTemperature
	SchemaHumidity    = proto.SchemaHumidity
	SchemaCommand     = proto.SchemaCommand
)
