package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SWAI-Ltd/Qumbed/internal/crypto"
	"github.com/SWAI-Ltd/Qumbed/internal/mesh"
	"github.com/SWAI-Ltd/Qumbed/internal/proto"
)

func main() {
	addr := flag.String("addr", ":0", "listen address")
	relayAddr := flag.String("relay", "localhost:6121", "relay address (empty to disable)")
	noDiscovery := flag.Bool("no-discovery", false, "disable mDNS (use when relay-only or in containers)")
	nodeID := flag.String("id", "node-1", "node id")
	mode := flag.String("mode", "sub", "sub | pub")
	topic := flag.String("topic", "sensors/temp", "topic")
	recipientKey := flag.String("recipient-key", "", "recipient public key (hex) for pub mode")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	var relay string
	if *relayAddr != "" {
		relay = *relayAddr
	}

	node, err := mesh.NewNode(ctx, mesh.Config{
		Addr:             *addr,
		NodeID:           *nodeID,
		RelayAddr:        relay,
		DisableDiscovery: *noDiscovery,
		OnMessage: func(t string, payload []byte) {
			slog.Info("message received", "topic", t, "payload", string(payload))
		},
	})
	if err != nil {
		slog.Error("failed to start node", "err", err)
		os.Exit(1)
	}
	defer node.Close()

	slog.Info("node started", "addr", node.Addr(), "id", *nodeID)
	fmt.Println("PublicKey:", hex.EncodeToString(node.PublicKey()[:]))

	switch *mode {
	case "sub":
		if err := node.Subscribe(ctx, *topic, proto.SchemaTemperature); err != nil {
			slog.Error("subscribe failed", "err", err)
		}
		slog.Info("subscribed", "topic", *topic)
		<-ctx.Done()
	case "pub":
		var pub *[crypto.PublicKeySize]byte
		if *recipientKey != "" {
			b, err := hex.DecodeString(*recipientKey)
			if err != nil || len(b) != crypto.PublicKeySize {
				slog.Error("invalid recipient-key", "err", err)
				os.Exit(1)
			}
			pub = new([crypto.PublicKeySize]byte)
			copy(pub[:], b)
		} else {
			pub = node.PublicKey() // self-publish for demo
		}
		payload, _ := json.Marshal(proto.Temperature{
			Celsius:    22.5,
			TimestampMs: time.Now().UnixMilli(),
			SensorID:   *nodeID,
		})
		if err := node.Publish(ctx, *topic, proto.SchemaTemperature, payload, pub); err != nil {
			slog.Error("publish failed", "err", err)
		} else {
			slog.Info("published", "topic", *topic)
		}
		<-ctx.Done()
	default:
		fmt.Println("usage: node -mode sub|pub [-relay localhost:6121] [-topic sensors/temp] [-recipient-key <hex>]")
	}
}
