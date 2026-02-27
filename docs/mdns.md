# mDNS Discovery in Qumbed

This document describes how Qumbed uses **mDNS** (multicast DNS) for local-network peer discovery. It is intended for developers integrating with Qumbed or implementing clients in other languages.

---

## 1. Overview

### What is mDNS?

**mDNS** (multicast DNS) is a protocol that lets devices on the same link-local network discover services by name and type without a central directory. It is defined by:

- [RFC 6762](https://tools.ietf.org/html/rfc6762) — Multicast DNS  
- [RFC 6763](https://tools.ietf.org/html/rfc6763) — DNS-Based Service Discovery (DNS-SD)

Multicast queries and responses use the link-local multicast group (e.g. `224.0.0.251` for IPv4). Typical latency for discovery is under a second on a LAN.

### Why Qumbed uses mDNS

Qumbed supports two ways for nodes to communicate:

1. **P2P on the same LAN** — Nodes discover each other via mDNS and can connect directly over QUIC.
2. **Via a relay** — When peers are on different networks (or mDNS is disabled), nodes use an optional relay to forward messages.

mDNS provides **zero-configuration discovery**: devices on the same subnet can find each other without manual IP configuration or a central broker. This fits IoT, local sensors, and hybrid topologies where some traffic stays local and some goes through the relay.

---

## 2. Service type and domain

Qumbed advertises and browses a single service type on the local domain:

| Item        | Value           | Description                          |
|------------|-----------------|--------------------------------------|
| Service type | `_qumbed._udp` | UDP-based Qumbed service (DNS-SD)    |
| Domain     | `local.`        | Standard mDNS local domain           |

Full name: `_qumbed._udp.local.`

Each instance is advertised with an **instance name** (e.g. the node ID such as `sensor-1` or `alice`). The same type is used for both publishing this node and browsing for other nodes.

---

## 3. Publish and browse behavior

### Publish (advertise)

When mDNS discovery is **enabled**, each Qumbed node:

1. Starts a QUIC server on a chosen address (e.g. `:0` for any port).
2. Calls the discovery layer with the node’s **instance name**, **port**, optional **topics**, and **public key**.
3. Advertises an mDNS service:
   - **Type:** `_qumbed._udp`
   - **Name:** instance name (e.g. `NodeID` from config)
   - **Port:** the QUIC server’s port
   - **TXT records:** see below

The implementation uses a single zeroconf client that both **publishes** this service and **browses** for the same service type, so the node sees other Qumbed instances on the LAN.

### Browse (discover)

While running, the node listens for mDNS responses for `_qumbed._udp`. When another instance appears (or updates), the library raises an event. Qumbed then:

1. Parses the peer’s address (host:port), name, and optional TXT records.
2. Builds a `Peer` (name, addr, port, topics, public key).
3. Stores the peer in the mesh’s peer table and uses the address for QUIC when doing P2P.

Discovery is continuous: new peers appear when they join the LAN, and entries can be updated or removed when they leave or change.

### Browse-only mode

The package also supports **browse-only** discovery: a process can watch for `_qumbed._udp` services **without** advertising itself. This is useful for monitoring or tooling. In Go, use `discovery.NewBrowser(onPeer)` instead of `discovery.New(...)`.

---

## 4. TXT records

Qumbed encodes optional metadata in mDNS TXT records so that peers can learn each other’s public key and topics without an extra round-trip.

| Key      | Format        | Description                                  |
|----------|---------------|----------------------------------------------|
| `pubkey=` | 64 hex chars  | Subscriber’s public key (32 bytes, hex)      |
| `topics=` | Comma-separated | List of topic names this node cares about |

- **pubkey** — Used for E2EE: publishers encrypt to this key; the relay only forwards. Omitted if no key is provided.
- **topics** — Hint of which topics this node subscribes to or publishes; can be used for filtering or UI. Omitted if empty.

Parsing is lenient: unknown keys are ignored; invalid hex or missing fields leave those fields empty.

---

## 5. Integration in Qumbed

### Mesh node

When you create a mesh node with discovery **enabled** (default):

1. The node starts the QUIC server and gets a port.
2. It calls `discovery.New(nodeID, port, topics, publicKey, onPeerDiscovered)`.
3. Discovered peers are stored in the node’s peer map; the node can use them for P2P QUIC connections (when P2P delivery is implemented).

If you set `DisableDiscovery: true`, mDNS is not started and the node relies only on the relay (if configured).

### Client SDK

The Go client exposes discovery via mesh config:

```go
c, err := client.New(ctx, client.Config{
    Addr:              ":0",
    NodeID:            "my-node",
    RelayAddr:         "relay.example.com:6121",
    DisableDiscovery:  false,  // enable mDNS (default)
})
```

- **DisableDiscovery: false** — mDNS is used; peers on the LAN are discovered.
- **DisableDiscovery: true** — mDNS is off; use when running in containers, or relay-only, or when the LAN is untrusted (see [Security](#7-security)).

---

## 6. When to enable or disable mDNS

| Scenario                    | Recommendation        | Reason |
|----------------------------|------------------------|--------|
| Same-LAN devices, no relay | Enable (default)      | P2P discovery and direct QUIC. |
| Same LAN + relay           | Enable                 | Discover local peers; use relay for remote. |
| Containers / no multicast  | Disable                | mDNS often doesn’t work in typical container networks. |
| Relay-only deployment      | Disable                | No need for discovery; fixed relay address. |
| Untrusted LAN              | Disable                | Discovery is unauthenticated; see [Security](#7-security). |

Use the `-no-discovery` flag for the node CLI when you want mDNS disabled.

---

## 7. Security

Discovery over mDNS is **unauthenticated**. Any host on the LAN can:

- Advertise a fake `_qumbed._udp` service with an arbitrary name, address, and public key.
- Impersonate another node or attract traffic to a malicious endpoint.

Qumbed does **not** authenticate or authorize mDNS advertisements. Payloads remain protected by E2EE once a connection is established, but discovery itself is trusted only as much as the LAN.

**Recommendation:** In environments where the LAN is not trusted, set `DisableDiscovery: true` and use fixed relay addresses and proper transport security. See [SECURITY.md](../SECURITY.md) for the full threat model and mitigations.

---

## 8. Example and API (Go)

### Standalone example

The **mdns_discovery** example shows publish + browse without running a full node or relay:

```bash
# Terminal 1
go run ./examples/mdns_discovery/main.go -name alice

# Terminal 2
go run ./examples/mdns_discovery/main.go -name bob
```

Each process advertises itself and browses for `_qumbed._udp`; both should log that they discovered the other (e.g. name, address, port).

### Package API (internal)

The implementation lives in `internal/discovery/`:

- **`discovery.New(nodeName, port, topics, publicKey, onPeer)`** — Creates a discovery that publishes this node and browses for peers. `onPeer(Peer)` is called for each discovered or updated peer.
- **`discovery.NewBrowser(onPeer)`** — Creates a discovery that only browses (no publish).
- **`discovery.Peer`** — `Name`, `Addr`, `Port`, `Topics`, `PublicKey`.
- **`discovery.AddrForQUIC(peer)`** — Returns the address string suitable for QUIC (handles IPv6 brackets).
- **`Close()`** — Stops publishing and browsing and releases resources.

---

## 9. References

- [RFC 6762 — Multicast DNS](https://tools.ietf.org/html/rfc6762)
- [RFC 6763 — DNS-Based Service Discovery](https://tools.ietf.org/html/rfc6763)
- [betamos/zeroconf](https://github.com/betamos/zeroconf) — Go library used for mDNS publish and browse
- [SECURITY.md](../SECURITY.md) — Qumbed security model and mDNS caveats
- [Wire protocol](wire-protocol.md) — QUIC and frame format (Discovery frame type)
