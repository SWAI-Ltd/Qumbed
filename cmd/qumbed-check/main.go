// qumbed-check is a validation CLI: it subscribes to a topic and confirms that
// incoming messages are valid according to the protocol (schema, format).
// Usage: go run ./cmd/qumbed-check -topic test -relay localhost:6121
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qumbed/qumbed/client"
	"github.com/qumbed/qumbed/internal/proto"
)

func main() {
	relay := flag.String("relay", "localhost:6121", "relay address")
	topic := flag.String("topic", "test", "topic to listen on")
	schema := flag.String("schema", "sensor.Temperature", "expected schema (sensor.Temperature, sensor.Humidity, control.Command)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	c, err := client.New(ctx, client.Config{
		Addr:             ":0",
		NodeID:            "qumbed-check",
		RelayAddr:         *relay,
		DisableDiscovery:  true,
		MessageBuffer:     32,
	})
	if err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer c.Close()

	schemaID := *schema
	if err := c.Subscribe(ctx, *topic, schemaID); err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}
	fmt.Printf("Listening on topic %q (schema %s). Send messages to validate.\n", *topic, schemaID)

	var okCount, failCount int
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\nDone. Valid: %d, Invalid: %d\n", okCount, failCount)
			return
		case m, ok := <-c.Messages():
			if !ok {
				return
			}
			valid, err := validatePayload(schemaID, m.Payload)
			if valid {
				okCount++
				fmt.Printf("[%s] OK  %s -> %s\n", time.Now().Format("15:04:05"), m.Topic, string(m.Payload))
			} else {
				failCount++
				fmt.Printf("[%s] FAIL %s: %v (payload: %q)\n", time.Now().Format("15:04:05"), m.Topic, err, m.Payload)
			}
		}
	}
}

func validatePayload(schemaID string, payload []byte) (valid bool, err error) {
	if err := proto.ValidatePayload(schemaID, payload); err != nil {
		return false, err
	}
	return true, nil
}
