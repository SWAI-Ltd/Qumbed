package discovery

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/betamos/zeroconf"
)

const (
	ServiceType   = "_qumbed._udp"
	Domain        = "local."
	txtKeyPubkey  = "pubkey="
	txtKeyTopics  = "topics="
	publicKeyHexLen = 64 // 32 bytes
)

// Peer represents a discovered peer on the local network
type Peer struct {
	Name      string
	Addr      string
	Port      int
	Topics    []string
	PublicKey []byte
}

// Discovery handles mDNS service discovery for P2P mesh
type Discovery struct {
	client   *zeroconf.Client
	nodeName string
	port     int
	onPeer   func(Peer)
}

// New creates a new mDNS discovery, publishing this node and browsing for peers
func New(nodeName string, port int, topics []string, publicKey []byte, onPeer func(Peer)) (*Discovery, error) {
	svcType := zeroconf.NewType(ServiceType)
	port16 := uint16(port)
	if port > 65535 {
		port16 = 6121
	}
	self := zeroconf.NewService(svcType, nodeName, port16)
	self.Text = buildTxtRecords(topics, publicKey)

	client, err := zeroconf.New().
		Publish(self).
		Browse(func(e zeroconf.Event) {
			handleEvent(e, onPeer)
		}, svcType).
		Open()
	if err != nil {
		return nil, fmt.Errorf("zeroconf: %w", err)
	}

	return &Discovery{
		client:   client,
		nodeName: nodeName,
		port:     port,
		onPeer:   onPeer,
	}, nil
}

// NewBrowser creates a discovery that only browses for _qumbed._udp services (no publish).
// Use this to watch for peers without advertising this node.
func NewBrowser(onPeer func(Peer)) (*Discovery, error) {
	svcType := zeroconf.NewType(ServiceType)
	client, err := zeroconf.New().
		Browse(func(e zeroconf.Event) {
			handleEvent(e, onPeer)
		}, svcType).
		Open()
	if err != nil {
		return nil, fmt.Errorf("zeroconf: %w", err)
	}
	return &Discovery{client: client, onPeer: onPeer}, nil
}

func buildTxtRecords(topics []string, publicKey []byte) []string {
	var out []string
	if len(publicKey) >= 32 {
		out = append(out, txtKeyPubkey+hex.EncodeToString(publicKey[:32]))
	}
	if len(topics) > 0 {
		out = append(out, txtKeyTopics+strings.Join(topics, ","))
	}
	return out
}

func parseTxtRecords(text []string) (topics []string, publicKey []byte) {
	for _, s := range text {
		if strings.HasPrefix(s, txtKeyPubkey) {
			hexStr := strings.TrimPrefix(s, txtKeyPubkey)
			if len(hexStr) == publicKeyHexLen {
				if b, err := hex.DecodeString(hexStr); err == nil && len(b) == 32 {
					publicKey = b
				}
			}
		} else if strings.HasPrefix(s, txtKeyTopics) {
			raw := strings.TrimPrefix(s, txtKeyTopics)
			if raw != "" {
				topics = strings.Split(raw, ",")
				for i := range topics {
					topics[i] = strings.TrimSpace(topics[i])
				}
			}
		}
	}
	return topics, publicKey
}

func handleEvent(e zeroconf.Event, onPeer func(Peer)) {
	var addrs []string
	for _, a := range e.Addrs {
		if a.IsValid() {
			addrs = append(addrs, net.JoinHostPort(a.String(), strconv.Itoa(int(e.Port))))
		}
	}
	if len(addrs) == 0 {
		return
	}
	addr := addrs[0]
	for _, a := range addrs {
		if !strings.Contains(a, ":") || strings.Count(a, ":") < 2 {
			addr = a
			break
		}
	}

	topics, publicKey := parseTxtRecords(e.Text)
	peer := Peer{Name: e.Name, Addr: addr, Port: int(e.Port), Topics: topics, PublicKey: publicKey}
	if onPeer != nil {
		onPeer(peer)
	}
}

// Close stops discovery
func (d *Discovery) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// ParseAddr converts "host:port" to "host:port" for QUIC
func ParseAddr(s string) (host string, port int, err error) {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

// AddrForQUIC returns address suitable for quic.DialAddr (e.g. "192.168.1.5:6121")
func AddrForQUIC(peer Peer) string {
	if strings.Contains(peer.Addr, "]") {
		return "[" + peer.Addr + "]" // IPv6
	}
	return peer.Addr
}
