# Qumbed Wire Protocol Specification

This document describes the on-the-wire format and connection lifecycle for Qumbed. Use it to implement clients in any language without reading the Go source.

---

## 1. Transport

- **Protocol:** QUIC over UDP (ALPN: `qumbed/1`).
- **TLS:** Required for QUIC. Development uses self-signed certs; production should use proper certificates (see Secure Conn example).
- **Idle timeout:** 5 minutes (connection may be closed by the server after inactivity).

---

## 2. Packet Structure (Application Layer)

Every Qumbed message is a **length-prefixed JSON frame** on a QUIC stream.

### Header (4 bytes)

| Offset | Size | Endianness | Description |
|--------|------|------------|-------------|
| 0      | 4    | Big-endian | Payload length in bytes (max 1 MiB) |

### Payload (JSON body)

The payload is a single JSON object representing a **Frame**. The frame has a type discriminator and an optional type-specific payload.

**Top-level frame (abbreviated):**

```json
{
  "t": <frame_type>,
  "p": { ... },   // when type = Publish
  "s": { ... },   // when type = Subscribe
  "u": { ... },   // when type = Unsubscribe
  "m": { ... },   // when type = Message
  "a": { ... },   // when type = Ack
  "e": { ... },   // when type = Error
  "d": { ... }    // when type = Discovery
}
```

### Frame Types (decimal)

| Value | Name        | Direction        | Description |
|-------|-------------|------------------|-------------|
| 1     | Publish     | Client → Relay   | Publish encrypted payload to a topic |
| 2     | Subscribe   | Client → Relay   | Register interest in a topic (with public key) |
| 3     | Unsubscribe | Client → Relay   | Unregister from a topic |
| 4     | Message     | Relay → Client   | Delivered message (encrypted payload) |
| 5     | Ack         | Relay → Client   | Acknowledgment (success/failure) |
| 6     | Error       | Relay → Client   | Error response |
| 7     | Discovery   | P2P              | mDNS / discovery metadata |

### Field Layout by Frame Type

- **Publish (`p`):** `topic`, `payload` (base64/bytes), `schema_id`, `recipient_key_id`, `sender_public_key`
- **Subscribe (`s`):** `topic`, `schema_id`, `public_key`
- **Unsubscribe (`u`):** `topic`
- **Message (`m`):** `topic`, `encrypted_payload`, `sender_key_id`, `sender_public_key`
- **Ack (`a`):** `message_id`, `ok`
- **Error (`e`):** `code`, `message`
- **Discovery (`d`):** `node_id`, `topics`, `public_key`, `addr`

(Exact field names match the Go struct tags in `internal/proto/frame.go`.)

---

## 3. Connection State Machine

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                     IDLE (no QUIC conn)                  │
                    └───────────────────────────┬─────────────────────────────┘
                                                │ DialQUIC(relay)
                                                ▼
                    ┌─────────────────────────────────────────────────────────┐
                    │                   CONNECTED (QUIC session)              │
                    └───┬─────────────────────────────────────────────────┬───┘
                        │ OpenStream + Send Subscribe                     │ Send Publish
                        ▼                                                 ▼
                    ┌───────────────────────┐               ┌───────────────────────┐
                    │    SUBSCRIBED         │               │    PUBLISH            │
                    │ (recv Message frames) │               │ (send Publish frame)  │
                    └───────────┬───────────┘               └───────────┬───────────┘
                                │                                       │
                                │ Recv Message → onMsg(topic, payload)   │ Recv Ack / Error
                                │ (decrypt with private key)            │
                                ▼                                       ▼
                    ┌─────────────────────────────────────────────────────────────┐
                    │              CONNECTED (can Subscribe again / Publish again)│
                    └─────────────────────────────────────────────────────────────┘
                                                │
                        Stream/conn close       │ Idle timeout / Close
                        or Error frame          │
                                                ▼
                    ┌─────────────────────────────────────────────────────────┐
                    │                     CLOSED                               │
                    │ (reconnect: new DialQUIC + Subscribe if needed)         │
                    └─────────────────────────────────────────────────────────┘
```

- **Start:** Client establishes QUIC connection to relay (or peer for P2P).
- **Stay alive:** QUIC keeps the connection open; no explicit heartbeat in the app layer (QUIC handles keepalive).
- **End:** Stream or connection closed; client may reconnect and re-subscribe (no “Last Will” in v1—document reconnection instead).

---

## 4. Error Codes

These are returned in **Error** frames (`"e": { "code": "...", "message": "..." }`). Use them for programmatic handling.

| Code              | Description |
|-------------------|-------------|
| `SCHEMA_UNKNOWN`  | Subscribe used a schema_id the server does not recognize. |
| `SCHEMA_INVALID`  | Publish payload did not validate against the given schema (e.g. invalid JSON or missing required fields). |
| (future)          | `UNAUTHORIZED`, `RATE_LIMIT`, etc. can be added and documented here. |

---

## 5. Schema IDs (typed topics)

| Schema ID             | Purpose              | Payload shape (JSON) |
|-----------------------|----------------------|----------------------|
| `sensor.Temperature`  | Temperature readings | `celsius`, `timestamp_ms`, `sensor_id` |
| `sensor.Humidity`     | Humidity readings    | `percent`, `timestamp_ms`, `sensor_id` |
| `control.Command`     | Actuator commands   | `action`, `params` (map) |

Publishers must send valid JSON that matches the schema; otherwise the server responds with `SCHEMA_INVALID`.

---

## 6. End-to-End Encryption (E2EE)

- Key exchange: each subscriber has a **Curve25519** public key; publishers seal payloads with **NaCl box** (recipient’s public key, sender’s private key).
- The relay only sees: topic, schema_id, recipient_key_id, sender_public_key—**not** the plaintext payload.
- Subscriber decrypts with `box.Open(encrypted_payload, sender_public_key, my_private_key)`.

For full binary layout of keys and ciphertext, use the Go `internal/crypto` package or the `/proto` definitions as the reference.
