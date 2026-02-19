package mesh

import (
	"context"
	"log/slog"
	"sync"

	"github.com/SWAI-Ltd/Qumbed/internal/crypto"
	"github.com/SWAI-Ltd/Qumbed/internal/discovery"
	"github.com/SWAI-Ltd/Qumbed/internal/proto"
	"github.com/SWAI-Ltd/Qumbed/internal/transport"
)

// Node is a Qumbed mesh node: peer + optional relay
type Node struct {
	keys       *crypto.KeyPair
	server     *transport.Server
	relay      *Relay
	disc       *discovery.Discovery
	peers      sync.Map // addr -> discovery.Peer
	subs       sync.Map // topic -> map[addr]subInfo
	schema     map[string]struct{}
	onMsg      func(topic string, payload []byte)
	nodeID     string
	relayAddr  string
	relayConn  *transport.Conn
	relayDone  chan struct{}
}

// Config for Node
type Config struct {
	Addr         string
	NodeID       string
	RelayAddr    string // optional relay for cross-network
	OnMessage    func(topic string, payload []byte)
	DisableDiscovery bool // set true to skip mDNS (e.g. in containers)
}

// NewNode creates a new mesh node
func NewNode(ctx context.Context, cfg Config) (*Node, error) {
	keys, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	n := &Node{
		keys:   keys,
		nodeID: cfg.NodeID,
		relayAddr: cfg.RelayAddr,
		onMsg:  cfg.OnMessage,
		schema: make(map[string]struct{}),
	}
	for _, s := range proto.KnownSchemas() {
		n.schema[s] = struct{}{}
	}

	// Start QUIC server
	n.server, err = transport.ListenQUIC(ctx, cfg.Addr)
	if err != nil {
		return nil, err
	}
	n.server.Handler = n.handleConn

	// Start mDNS discovery (optional)
	if !cfg.DisableDiscovery {
		port := 6121
		if addr := n.server.LocalAddr(); addr != "" {
			_, p, _ := discovery.ParseAddr(addr)
			if p > 0 {
				port = p
			}
		}
		n.disc, err = discovery.New(cfg.NodeID, port, nil, keys.Public[:], n.onPeerDiscovered)
		if err != nil {
			n.server.Listener.Close()
			return nil, err
		}
	}

	// Connect to relay if configured
	if cfg.RelayAddr != "" {
		n.relay = NewRelay(cfg.RelayAddr)
	}
	return n, nil
}

func (n *Node) onPeerDiscovered(peer discovery.Peer) {
	addr := discovery.AddrForQUIC(peer)
	n.peers.Store(addr, peer)
	slog.Debug("peer discovered", "addr", addr)
}

func (n *Node) handleConn(c *transport.Conn) {
	defer c.Close()
	var f proto.Frame
	for {
		if err := c.RecvFrame(&f); err != nil {
			return
		}
		n.handleFrame(c, &f)
	}
}

func (n *Node) handleFrame(c *transport.Conn, f *proto.Frame) {
	switch f.Type {
	case proto.FrameTypeSubscribe:
		if s := f.Subscribe; s != nil {
			n.handleSubscribe(c, s)
		}
	case proto.FrameTypePublish:
		if p := f.Publish; p != nil {
			n.handlePublish(c, p)
		}
	case proto.FrameTypeMessage:
		if m := f.Message; m != nil {
			n.handleMessage(c, m)
		}
	case proto.FrameTypeUnsubscribe:
		if u := f.Unsubscribe; u != nil {
			n.handleUnsubscribe(c, u)
		}
	}
}

func (n *Node) handleSubscribe(c *transport.Conn, s *proto.SubscribeFrame) {
	if _, ok := n.schema[s.SchemaID]; !ok && s.SchemaID != "" {
		c.SendFrame(&proto.Frame{Type: proto.FrameTypeError, Error: &proto.ErrorFrame{
			Code: "SCHEMA_UNKNOWN", Message: "unknown schema: " + s.SchemaID,
		}})
		return
	}
	key := c.RemoteAddr()
	v, _ := n.subs.LoadOrStore(s.Topic, &sync.Map{})
	m := v.(*sync.Map)
	m.Store(key, s.PublicKey)
	c.SendFrame(&proto.Frame{Type: proto.FrameTypeAck, Ack: &proto.AckFrame{OK: true}})
}

func (n *Node) handleUnsubscribe(c *transport.Conn, u *proto.UnsubscribeFrame) {
	if v, ok := n.subs.Load(u.Topic); ok {
		m := v.(*sync.Map)
		m.Delete(c.RemoteAddr())
	}
}

func (n *Node) handlePublish(c *transport.Conn, p *proto.PublishFrame) {
	if err := proto.ValidatePayload(p.SchemaID, p.Payload); err != nil {
		c.SendFrame(&proto.Frame{Type: proto.FrameTypeError, Error: &proto.ErrorFrame{
			Code: "SCHEMA_INVALID", Message: err.Error(),
		}})
		return
	}
	// Forward encrypted payload to subscribers (zero-knowledge relay)
	msg := &proto.MessageFrame{
		Topic:            p.Topic,
		EncryptedPayload: p.Payload,
		SenderKeyID:      p.RecipientKeyID,
		SenderPublicKey:  p.SenderPublicKey,
	}
	if v, ok := n.subs.Load(p.Topic); ok {
		m := v.(*sync.Map)
		m.Range(func(key, _ interface{}) bool {
			// Would send to connection for key - simplified: ack only
			return true
		})
		_ = msg
	}
	c.SendFrame(&proto.Frame{Type: proto.FrameTypeAck, Ack: &proto.AckFrame{OK: true}})
}

func (n *Node) handleMessage(c *transport.Conn, m *proto.MessageFrame) {
	if len(m.SenderPublicKey) != crypto.PublicKeySize {
		return
	}
	var senderPub [crypto.PublicKeySize]byte
	copy(senderPub[:], m.SenderPublicKey)
	plain, ok := crypto.Open(m.EncryptedPayload, &senderPub, n.keys.Private)
	if !ok {
		return
	}
	if n.onMsg != nil {
		n.onMsg(m.Topic, plain)
	}
}

// Publish sends a message to a topic (E2EE to recipient)
func (n *Node) Publish(ctx context.Context, topic, schemaID string, payload []byte, recipientPub *[crypto.PublicKeySize]byte) error {
	if err := proto.ValidatePayload(schemaID, payload); err != nil {
		return err
	}
	enc, err := crypto.Seal(payload, recipientPub, n.keys.Private)
	if err != nil {
		return err
	}
	f := &proto.Frame{
		Type: proto.FrameTypePublish,
		Publish: &proto.PublishFrame{
			Topic:           topic,
			Payload:         enc,
			SchemaID:        schemaID,
			RecipientKeyID:  crypto.KeyID(recipientPub),
			SenderPublicKey: n.keys.Public[:],
		},
	}

	if n.relay != nil {
		conn, err := transport.DialQUIC(ctx, n.relayAddr)
		if err != nil {
			return err
		}
		defer conn.Close()
		return conn.SendFrame(f)
	}
	// P2P: would need to find peer and send
	return nil
}

// Subscribe registers for a topic and starts receiving (must keep relay conn open)
func (n *Node) Subscribe(ctx context.Context, topic, schemaID string) error {
	if n.relay == nil {
		return nil
	}
	conn, err := transport.DialQUIC(ctx, n.relayAddr)
	if err != nil {
		return err
	}
	f := &proto.Frame{
		Type: proto.FrameTypeSubscribe,
		Subscribe: &proto.SubscribeFrame{
			Topic:     topic,
			SchemaID:  schemaID,
			PublicKey: n.keys.Public[:],
		},
	}
	if err := conn.SendFrame(f); err != nil {
		conn.Close()
		return err
	}
	n.relayConn = conn
	n.relayDone = make(chan struct{})
	go n.relayRecvLoop(ctx, conn)
	return nil
}

func (n *Node) relayRecvLoop(ctx context.Context, c *transport.Conn) {
	defer close(n.relayDone)
	defer c.Close()
	var f proto.Frame
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := c.RecvFrame(&f); err != nil {
			slog.Debug("relayRecvLoop: recv ended", "err", err)
			return
		}
		if f.Type == proto.FrameTypeMessage && f.Message != nil {
			slog.Debug("relayRecvLoop: got message", "topic", f.Message.Topic)
			n.handleMessage(c, f.Message)
		}
	}
}

// PublicKey returns the node's public key for E2EE
func (n *Node) PublicKey() *[crypto.PublicKeySize]byte {
	return n.keys.Public
}

// Addr returns the local QUIC listen address
func (n *Node) Addr() string {
	return n.server.LocalAddr()
}

// Close shuts down the node
func (n *Node) Close() error {
	if n.relayConn != nil {
		n.relayConn.Close()
		if n.relayDone != nil {
			<-n.relayDone
		}
	}
	if n.disc != nil {
		n.disc.Close()
	}
	if n.server != nil && n.server.Listener != nil {
		_ = n.server.Listener.Close()
	}
	return nil
}
