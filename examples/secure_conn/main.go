// Secure Conn demonstrates how security works: QUIC TLS + E2EE.
// See README in this folder for loading your own certificates.
// Run: relay first, then sub, then pub with -recipient-key <sub_hex>.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
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
	recipientKey := flag.String("recipient-key", "", "subscriber public key (hex)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	c, err := client.New(ctx, client.Config{
		Addr:             ":0",
		NodeID:            "secure-node",
		RelayAddr:         *relay,
		DisableDiscovery:  true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	log.Println("PublicKey (share with publishers):", hex.EncodeToString(c.PublicKey()[:]))

	topic := "secure/demo"
	switch *mode {
	case "sub":
		if err := c.Subscribe(ctx, topic, client.SchemaTemperature); err != nil {
			log.Fatal(err)
		}
		log.Println("subscribed (E2EE); waiting for messages...")
		for m := range c.Messages() {
			var t proto.Temperature
			if json.Unmarshal(m.Payload, &t) == nil {
				log.Printf("decrypted: %.1f°C", t.Celsius)
			}
		}
	case "pub":
		var recipient *[crypto.PublicKeySize]byte
		if *recipientKey != "" {
			b, err := hex.DecodeString(*recipientKey)
			if err != nil || len(b) != crypto.PublicKeySize {
				log.Fatal("invalid -recipient-key")
			}
			recipient = new([crypto.PublicKeySize]byte)
			copy(recipient[:], b)
		} else {
			recipient = c.PublicKey()
		}
		payload, _ := json.Marshal(proto.Temperature{
			Celsius:     21.0,
			TimestampMs: time.Now().UnixMilli(),
			SensorID:    "secure-pub",
		})
		if err := c.Publish(ctx, topic, client.SchemaTemperature, payload, recipient); err != nil {
			log.Fatal(err)
		}
		log.Println("published (encrypted for recipient)")
		<-ctx.Done()
	}
}
