package discovery

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/betamos/zeroconf"
)

const (
	ServiceType = "_qumbed._udp"
	Domain      = "local."
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

	peer := Peer{Name: e.Name, Addr: addr, Port: int(e.Port), Topics: []string{}}
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
