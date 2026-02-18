// Simple Pub/Sub is the "Hello World" of the Qumbed protocol.
// Run the relay first: go run ./cmd/relay -addr :6121
// Then in one terminal: go run ./examples/simple_pubsub/main.go -mode sub
// In another:       go run ./examples/simple_pubsub/main.go -mode pub -recipient-key <hex_from_sub>
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qumbed/qumbed/client"
	"github.com/qumbed/qumbed/internal/crypto"
	"github.com/qumbed/qumbed/internal/proto"
)

func main() {
	relay := flag.String("relay", "localhost:6121", "relay address")
	mode := flag.String("mode", "sub", "sub or pub")
	topic := flag.String("topic", "sensors/temp", "topic")
	recipientKey := flag.String("recipient-key", "", "subscriber public key (hex) for pub mode")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	cfg := client.Config{
		Addr:              ":0",
		NodeID:            "simple-node",
		RelayAddr:         *relay,
		DisableDiscovery:  true,
		MessageBuffer:     32,
	}
	c, err := client.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("PublicKey:", hex.EncodeToString(c.PublicKey()[:]))

	switch *mode {
	case "sub":
		if err := c.Subscribe(ctx, *topic, client.SchemaTemperature); err != nil {
			log.Fatal(err)
		}
		log.Println("subscribed to", *topic, "- waiting for messages...")
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-c.Messages():
				if !ok {
					return
				}
				var t proto.Temperature
				if json.Unmarshal(m.Payload, &t) == nil {
					log.Printf("received %s: %.1f°C from %s", m.Topic, t.Celsius, t.SensorID)
				} else {
					log.Printf("received %s: %s", m.Topic, string(m.Payload))
				}
			}
		}
	case "pub":
		var recipient *[crypto.PublicKeySize]byte
		if *recipientKey != "" {
			b, err := hex.DecodeString(*recipientKey)
			if err != nil || len(b) != crypto.PublicKeySize {
				log.Fatal("invalid -recipient-key (need 32-byte hex)")
			}
			recipient = new([crypto.PublicKeySize]byte)
			copy(recipient[:], b)
		} else {
			recipient = c.PublicKey()
		}
		payload, _ := json.Marshal(proto.Temperature{
			Celsius:     22.0,
			TimestampMs: time.Now().UnixMilli(),
			SensorID:    "simple-pub",
		})
		if err := c.Publish(ctx, *topic, client.SchemaTemperature, payload, recipient); err != nil {
			log.Fatal(err)
		}
		log.Println("published to", *topic)
		<-ctx.Done()
	default:
		fmt.Println("usage: -mode sub | pub [-relay localhost:6121] [-topic sensors/temp] [-recipient-key <hex>]")
	}
}
