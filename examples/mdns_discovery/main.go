// mDNS discovery example: advertise this node and/or browse for _qumbed._udp peers on the LAN.
//
// Run on a single machine (multiple terminals) or on multiple machines on the same LAN:
//
//   Terminal 1 (publish + browse):
//     go run ./examples/mdns_discovery/main.go -name alice -port 6121
//
//   Terminal 2 (publish + browse):
//     go run ./examples/mdns_discovery/main.go -name bob -port 6122
//
//   Terminal 3 (browse only, no publish):
//     go run ./examples/mdns_discovery/main.go -browse-only
//
// You should see alice and bob discover each other; the browse-only process sees both.
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/SWAI-Ltd/Qumbed/internal/crypto"
	"github.com/SWAI-Ltd/Qumbed/internal/discovery"
)

func main() {
	nodeName := flag.String("name", "mdns-example-node", "node name advertised via mDNS")
	port := flag.Int("port", 6121, "port to advertise (and listen on if running as node)")
	browseOnly := flag.Bool("browse-only", false, "only browse for peers, do not publish this node")
	flag.Parse()

	var disc *discovery.Discovery
	var err error

	if *browseOnly {
		log.Println("mDNS: browse-only mode (watching for _qumbed._udp services)")
		disc, err = discovery.NewBrowser(func(peer discovery.Peer) {
			printPeer("discovered", peer)
		})
	} else {
		keys, kerr := crypto.GenerateKeyPair()
		if kerr != nil {
			log.Fatal(kerr)
		}
		log.Printf("mDNS: publishing as %q on port %d (public key: %s)", *nodeName, *port, hex.EncodeToString(keys.Public[:8])+"...")
		topics := []string{"sensors/temp", "alerts"}
		disc, err = discovery.New(*nodeName, *port, topics, keys.Public[:], func(peer discovery.Peer) {
			printPeer("discovered", peer)
		})
	}
	if err != nil {
		log.Fatal(err)
	}
	defer disc.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Println("Peers will be listed below. Ctrl+C to exit.")
	<-ctx.Done()
	log.Println("shutting down...")
	time.Sleep(200 * time.Millisecond)
}

func printPeer(label string, p discovery.Peer) {
	pubStr := ""
	if len(p.PublicKey) == 32 {
		pubStr = hex.EncodeToString(p.PublicKey[:8]) + "..."
	} else {
		pubStr = "(none)"
	}
	topicsStr := ""
	if len(p.Topics) > 0 {
		topicsStr = fmt.Sprintf(" topics=%v", p.Topics)
	}
	log.Printf("[%s] %s addr=%s port=%d pubkey=%s%s", label, p.Name, p.Addr, p.Port, pubStr, topicsStr)
}
