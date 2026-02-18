package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"time"

	"github.com/qumbed/qumbed/internal/proto"
	"github.com/quic-go/quic-go"
)

// Default idle timeout: 5 minutes (QUIC default is 30s, too short for pub/sub)
var defaultQuicConfig = &quic.Config{
	MaxIdleTimeout: 5 * time.Minute,
}

const (
	AddrLADDR = ":0"
	ProtoID   = "qumbed/1"
)

// Conn wraps a QUIC connection with frame read/write
type Conn struct {
	Stream quic.Stream
	Conn   quic.Connection
}

// NewConn wraps a QUIC stream and connection
func NewConnWithConn(stream quic.Stream, conn quic.Connection) *Conn {
	return &Conn{Stream: stream, Conn: conn}
}

// NewConn wraps a QUIC stream (connection may be nil for dial)
func NewConn(stream quic.Stream) *Conn {
	return &Conn{Stream: stream}
}

// RemoteAddr returns the peer address
func (c *Conn) RemoteAddr() string {
	if c.Conn != nil {
		return c.Conn.RemoteAddr().String()
	}
	return "unknown"
}

// SendFrame encodes and sends a frame
func (c *Conn) SendFrame(f *proto.Frame) error {
	return f.Encode(c.Stream)
}

// RecvFrame reads and decodes a frame
func (c *Conn) RecvFrame(f *proto.Frame) error {
	return f.Decode(c.Stream)
}

// Close closes the stream
func (c *Conn) Close() error {
	return c.Stream.Close()
}

// generateTLSConfig creates a self-signed cert for development
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{ProtoID},
	}, nil
}

// Server runs a QUIC listener
type Server struct {
	Listener *quic.Listener
	Handler  func(*Conn)
}

// ListenQUIC starts a QUIC server on addr. Pass handler to avoid races.
func ListenQUIC(ctx context.Context, addr string) (*Server, error) {
	return ListenQUICWithHandler(ctx, addr, nil)
}

// ListenQUICWithHandler starts a QUIC server with handler set before accepting.
func ListenQUICWithHandler(ctx context.Context, addr string, handler func(*Conn)) (*Server, error) {
	tlsCfg, err := generateTLSConfig()
	if err != nil {
		return nil, err
	}
	listener, err := quic.ListenAddr(addr, tlsCfg, defaultQuicConfig)
	if err != nil {
		return nil, err
	}
	s := &Server{Listener: listener, Handler: handler}
	go s.acceptLoop(ctx)
	return s, nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		sess, err := s.Listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go func() {
			stream, err := sess.AcceptStream(ctx)
			if err != nil {
				return
			}
			if s.Handler != nil {
				s.Handler(NewConnWithConn(stream, sess))
			} else {
				io.Copy(io.Discard, stream)
			}
		}()
	}
}

// DialQUIC connects to a QUIC server (skips cert verification for dev)
func DialQUIC(ctx context.Context, addr string) (*Conn, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{ProtoID},
	}
	sess, err := quic.DialAddr(ctx, addr, tlsCfg, defaultQuicConfig)
	if err != nil {
		return nil, err
	}
	stream, err := sess.OpenStreamSync(ctx)
	if err != nil {
		sess.CloseWithError(0, "")
		return nil, err
	}
	return NewConn(stream), nil
}

// LocalAddr returns the address of the QUIC listener
func (s *Server) LocalAddr() string {
	return s.Listener.Addr().String()
}
