package mesh

import (
	"context"
	"log/slog"
	"sync"

	"github.com/SWAI-Ltd/Qumbed/internal/proto"
	"github.com/SWAI-Ltd/Qumbed/internal/transport"
)

// RelayServer is a zero-knowledge broker: routes by topic, never sees payload
type RelayServer struct {
	server *transport.Server
	subs   sync.Map // topic -> map[connKey]subscriberInfo
}

type subInfo struct {
	schemaID  string
	publicKey []byte
	send      func(*proto.Frame) error
}

// RunRelay starts a relay server on addr
func RunRelay(ctx context.Context, addr string) (*RelayServer, error) {
	r := &RelayServer{}
	server, err := transport.ListenQUICWithHandler(ctx, addr, r.handleConn)
	if err != nil {
		return nil, err
	}
	r.server = server
	slog.Info("relay listening", "addr", server.LocalAddr())
	return r, nil
}

func (r *RelayServer) handleConn(c *transport.Conn) {
	key := c.RemoteAddr()
	defer func() {
		// Remove from all topic subscriptions
		r.subs.Range(func(topic, v interface{}) bool {
			m := v.(*sync.Map)
			m.Delete(key)
			return true
		})
		c.Close()
	}()

	var f proto.Frame
	for {
		if err := c.RecvFrame(&f); err != nil {
			return
		}
		switch f.Type {
		case proto.FrameTypeSubscribe:
			if s := f.Subscribe; s != nil {
				v, _ := r.subs.LoadOrStore(s.Topic, &sync.Map{})
				m := v.(*sync.Map)
				m.Store(key, &subInfo{
					schemaID:  s.SchemaID,
					publicKey: s.PublicKey,
					send:      c.SendFrame,
				})
				c.SendFrame(&proto.Frame{Type: proto.FrameTypeAck, Ack: &proto.AckFrame{OK: true}})
			}
		case proto.FrameTypeUnsubscribe:
			if u := f.Unsubscribe; u != nil {
				if v, ok := r.subs.Load(u.Topic); ok {
					v.(*sync.Map).Delete(key)
				}
			}
		case proto.FrameTypePublish:
			if p := f.Publish; p != nil {
				// Forward to all subscribers of this topic (zero-knowledge: payload stays encrypted)
				if v, ok := r.subs.Load(p.Topic); ok {
					msg := &proto.Frame{
						Type: proto.FrameTypeMessage,
						Message: &proto.MessageFrame{
							Topic:            p.Topic,
							EncryptedPayload: p.Payload,
							SenderKeyID:      p.RecipientKeyID,
							SenderPublicKey:  p.SenderPublicKey,
						},
					}
					count := 0
					v.(*sync.Map).Range(func(k, val interface{}) bool {
						si := val.(*subInfo)
						if si.send != nil {
							if err := si.send(msg); err != nil {
								slog.Error("relay: failed to forward to subscriber", "err", err, "sub", k)
							} else {
								count++
							}
						}
						return true
					})
					slog.Info("relay: forwarded", "topic", p.Topic, "subscribers", count)
				}
				c.SendFrame(&proto.Frame{Type: proto.FrameTypeAck, Ack: &proto.AckFrame{OK: true}})
			}
		}
	}
}
