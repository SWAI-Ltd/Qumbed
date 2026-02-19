# Qumbed — A Next-Gen MQTT Alternative in Go

[![Build](https://github.com/SWAI-Ltd/Qumbed/actions/workflows/build.yml/badge.svg)](https://github.com/SWAI-Ltd/Qumbed/actions/workflows/build.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![ProVerif](https://img.shields.io/badge/ProVerif-verified-green.svg)](docs/proverif.md)

**Platform support:**  
[![OS](https://img.shields.io/badge/OS-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey)](https://github.com/SWAI-Ltd/Qumbed)
[![Arch](https://img.shields.io/badge/Arch-amd64%20%7C%20arm64-lightgrey)](https://github.com/SWAI-Ltd/Qumbed)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)

Qumbed is a modern pub/sub messaging protocol that addresses the main limitations of MQTT with a QUIC-based, P2P-capable, E2EE-first design.

## Features

| Feature | MQTT (Legacy) | Qumbed |
|---------|---------------|--------|
| **Transport** | TCP (head-of-line blocking) | **QUIC/UDP** (independent streams) |
| **Security** | Optional TLS | **Native E2EE** (NaCl/Curve25519) |
| **Topology** | Centralized broker | **Hybrid P2P + Relay** (mDNS discovery) |
| **Data** | Raw bytes | **Schema-aware** (typed topics) |

### 1. No Head-of-Line Blocking (QUIC)

Uses QUIC over UDP with multiple independent streams. A lost packet on one stream does not block others, improving real-time performance (e.g., drones, sensors).

### 2. Brokerless P2P + Relay Fallback

- **mDNS discovery**: Devices on the same LAN discover each other and communicate directly.
- **Relay fallback**: When peers are on different networks, an optional relay forwards messages without reading content.

### 3. End-to-End Encryption by Default

Messages are encrypted with the subscriber’s public key before they leave the publisher. The relay only sees topic and routing metadata, not payload.

### 4. Schema Enforcement

Topics use typed schemas (e.g. `sensor.Temperature`, `control.Command`). Publishers must send valid payloads; mismatches are rejected.

### 5. Formal verification (ProVerif)

The application-layer protocol (Subscribe → Publish → Relay → Message) is modeled and verified with [ProVerif](https://bblanche.gitlabpages.inria.fr/proverif/) in the symbolic model. ProVerif proves **payload secrecy** (the attacker cannot derive the plaintext) and **correspondence** (if the subscriber decrypts a value, a publisher must have sent it). See [docs/proverif.md](docs/proverif.md) for scope, how to run, and the model file `proverif/qumbed.pv`.

## Architecture

```
┌─────────────┐                    ┌─────────────┐
│  Publisher  │──── QUIC ─────────▶│   Relay     │
│  (encrypts  │                    │ (forwards   │
│   for sub)  │                    │  blind)     │
└─────────────┘                    └──────┬──────┘
       │                                  │
       │ mDNS                             │ QUIC
       ▼                                  ▼
┌─────────────┐                    ┌─────────────┐
│ Subscriber  │◀──── E2EE payload ─│ Subscriber  │
│  (decrypts) │                    │  (decrypts) │
└─────────────┘                    └─────────────┘
```

## Quick Start

### How do I connect?

Use the **client SDK** so you don’t write raw sockets. Five lines to get going:

```go
import "github.com/SWAI-Ltd/Qumbed/client"

ctx := context.Background()
c, err := client.New(ctx, client.Config{RelayAddr: "localhost:6121", DisableDiscovery: true})
if err != nil { log.Fatal(err) }
defer c.Close()
// c.Subscribe(ctx, "mytopic", client.SchemaTemperature) and read from c.Messages()
```

Then run a relay and a subscriber (see below).

### 1. Run the Relay

```bash
go run ./cmd/relay -addr :6121
```

### 2. Run a Subscriber

```bash
go run ./cmd/node -mode sub -relay localhost:6121 -no-discovery
# Prints: PublicKey: <hex>  # use this for publisher
```

Use `-no-discovery` when running relay-only (e.g. in containers) to skip mDNS.

### 3. Publish a Message

```bash
go run ./cmd/node -mode pub -relay localhost:6121 -recipient-key <subscriber_public_key_hex> -no-discovery
```

For a self-test (publish to self):

```bash
go run ./cmd/node -mode pub -relay localhost:6121 -no-discovery
```

---

## Developer FAQ (README checklist)

1. **How do I connect?** — Use the Go client: `import "github.com/SWAI-Ltd/Qumbed/client"` and `client.New(ctx, client.Config{RelayAddr: "host:6121", ...})`. See the 5-line snippet above and the [examples/](examples/) directory.
2. **How is it secure?** — **Transport:** QUIC uses TLS 1.3 (self-signed in dev; use your own certs in production). **Application:** E2EE by default with NaCl box (Curve25519). The relay never sees plaintext; it only routes by topic and key ID. See [examples/secure_conn/](examples/secure_conn/).
3. **What is the performance gain?** — QUIC avoids head-of-line blocking (one lost packet doesn’t stall other streams). For latency/throughput numbers, run the [high-throughput example](examples/high_throughput/) and compare against MQTT on your workload.
4. **How do I handle failures?** — There is no “Last Will” in v1. **Reconnection:** if the relay connection drops, create a new client (or reconnect) and call `Subscribe` again. Use `context.Context` for timeouts on `Publish`/`Subscribe`. Read from `c.Messages()` until the channel is closed when the client is closed.

## Project Structure

```
qumbed/
├── cmd/
│   ├── node/         # Mesh node (pub/sub CLI)
│   ├── relay/        # Zero-knowledge relay server
│   └── qumbed-check/ # Validation CLI (listen and confirm messages)
├── client/           # Developer SDK (import this in your app)
├── docs/
│   └── wire-protocol.md  # Wire spec, state machine, error codes
├── examples/
│   ├── simple_pubsub/    # Hello World pub/sub
│   ├── secure_conn/      # TLS + E2EE usage
│   └── high_throughput/  # Multi-stream / throughput
├── internal/
│   ├── crypto/       # E2EE (NaCl box, Curve25519)
│   ├── discovery/    # mDNS P2P discovery
│   ├── mesh/         # Node, relay, routing logic
│   ├── proto/        # Wire format & schema validation
│   └── transport/    # QUIC transport layer
└── proto/            # Protobuf definitions (for codegen in other languages)
```

## Schema Types

- `sensor.Temperature` — `{celsius, timestamp_ms, sensor_id}`
- `sensor.Humidity` — `{percent, timestamp_ms, sensor_id}`
- `control.Command` — `{action, params}`

## Dependencies

- [quic-go](https://github.com/quic-go/quic-go) — QUIC transport
- [betamos/zeroconf](https://github.com/betamos/zeroconf) — mDNS discovery
- `golang.org/x/crypto/nacl/box` — E2EE

## Documentation & tooling

- **Wire protocol:** See [docs/wire-protocol.md](docs/wire-protocol.md) for packet layout, state machine, and error codes.
- **Formal verification:** [docs/proverif.md](docs/proverif.md) describes the ProVerif model and how to run it; `./scripts/verify_protocol.sh` runs verification (requires `proverif`).
- **Protobuf:** The [proto/](proto/) folder holds `.proto` files so Python/C++/other clients can generate compatible code.
- **Validation:** Run `go run ./cmd/qumbed-check -topic <topic>` to listen and confirm messages against the protocol (see [Integration](#integration-test) below).

### Integration test

- **Mock server:** Run the relay locally to mimic the protocol: `go run ./cmd/relay -addr :6121`. It speaks the same wire format as production.
- **Validation CLI:** Run `qumbed-check` to subscribe and verify messages:
  ```bash
  go run ./cmd/qumbed-check -topic test -relay localhost:6121 -schema sensor.Temperature
  ```
  Then publish to `test`; the tool prints whether each message is valid for the given schema.

## License

This project is open source under the [Apache License 2.0](LICENSE).
