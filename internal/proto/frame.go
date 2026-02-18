package proto

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

// Frame types
const (
	FrameTypePublish   = 1
	FrameTypeSubscribe = 2
	FrameTypeUnsubscribe = 3
	FrameTypeMessage   = 4
	FrameTypeAck       = 5
	FrameTypeError     = 6
	FrameTypeDiscovery = 7
)

// PublishFrame is sent when publishing to a topic
type PublishFrame struct {
	Topic           string `json:"topic"`
	Payload         []byte `json:"payload"`          // E2EE encrypted for recipient
	SchemaID        string `json:"schema_id"`
	RecipientKeyID  []byte `json:"recipient_key_id"`
	SenderPublicKey []byte `json:"sender_public_key"` // for relay to forward (zero-knowledge routing)
}

// SubscribeFrame registers interest in a topic
type SubscribeFrame struct {
	Topic     string `json:"topic"`
	SchemaID  string `json:"schema_id"`
	PublicKey []byte `json:"public_key"`
}

// UnsubscribeFrame
type UnsubscribeFrame struct {
	Topic string `json:"topic"`
}

// MessageFrame - relayed message (broker forwards without reading payload)
type MessageFrame struct {
	Topic            string `json:"topic"`
	EncryptedPayload []byte `json:"encrypted_payload"`
	SenderKeyID      []byte `json:"sender_key_id"`
	SenderPublicKey  []byte `json:"sender_public_key"` // needed for box.Open
}

// AckFrame
type AckFrame struct {
	MessageID string `json:"message_id"`
	OK       bool   `json:"ok"`
}

// ErrorFrame
type ErrorFrame struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DiscoveryFrame - P2P discovery
type DiscoveryFrame struct {
	NodeID    string   `json:"node_id"`
	Topics    []string `json:"topics"`
	PublicKey []byte   `json:"public_key"`
	Addr      string   `json:"addr"`
}

// Frame is the top-level wire message
type Frame struct {
	Type      int             `json:"t"`
	Publish   *PublishFrame   `json:"p,omitempty"`
	Subscribe *SubscribeFrame `json:"s,omitempty"`
	Unsubscribe *UnsubscribeFrame `json:"u,omitempty"`
	Message   *MessageFrame   `json:"m,omitempty"`
	Ack       *AckFrame       `json:"a,omitempty"`
	Error     *ErrorFrame     `json:"e,omitempty"`
	Discovery *DiscoveryFrame `json:"d,omitempty"`
}

// Encode writes a length-prefixed JSON frame to w
func (f *Frame) Encode(w io.Writer) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	// 4-byte big-endian length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Decode reads a length-prefixed JSON frame from r
func (f *Frame) Decode(r io.Reader) error {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > 1024*1024 { // 1MB max
		return io.ErrShortBuffer
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, f)
}
