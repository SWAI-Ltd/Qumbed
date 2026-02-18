// High Throughput shows how to use the client for many messages and multiple streams.
// QUIC provides multiple streams per connection; the SDK uses one stream per Subscribe session.
// For maximum throughput: batch payloads, reuse one client, and avoid blocking on Messages().
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
	"sync"
	"sync/atomic"
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
	n := flag.Int("n", 1000, "number of messages (pub mode)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	// Larger buffer so we don't drop messages under load
	cfg := client.Config{
		Addr:             ":0",
		NodeID:            "throughput-node",
		RelayAddr:         *relay,
		DisableDiscovery:  true,
		MessageBuffer:     2048,
	}
	c, err := client.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("PublicKey:", hex.EncodeToString(c.PublicKey()[:]))
	topic := "perf/bench"

	switch *mode {
	case "sub":
		if err := c.Subscribe(ctx, topic, client.SchemaTemperature); err != nil {
			log.Fatal(err)
		}
		var count int64
		start := time.Now()
		for m := range c.Messages() {
			atomic.AddInt64(&count, 1)
			_ = m
			if count == 1 {
				log.Println("first message received")
			}
		}
		elapsed := time.Since(start)
		log.Printf("received %d messages in %v", atomic.LoadInt64(&count), elapsed)
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
		start := time.Now()
		var wg sync.WaitGroup
		for i := 0; i < *n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				payload, _ := json.Marshal(proto.Temperature{
					Celsius:     float64(idx % 100),
					TimestampMs: time.Now().UnixMilli(),
					SensorID:    "throughput",
				})
				_ = c.Publish(ctx, topic, client.SchemaTemperature, payload, recipient)
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		log.Printf("published %d messages in %v (%.0f msg/s)", *n, elapsed, float64(*n)/elapsed.Seconds())
		<-ctx.Done()
	default:
		fmt.Println("usage: -mode sub | pub [-n 1000] [-recipient-key <hex>]")
	}
}
