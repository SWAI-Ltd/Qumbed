# Security

This document describes what the Qumbed protocol is designed to protect against, how it does so, and—importantly—what it does **not** protect against. We aim to be explicit so you can make informed decisions.

---

## What We Protect Against

### Man-in-the-Middle (MITM) on the wire

**When TLS is properly configured**, the protocol is resistant to MITM attacks for two reasons:

1. **Transport:** QUIC mandates TLS 1.3. If the client verifies the relay’s certificate (real CA or pinned key) and does **not** use `InsecureSkipVerify`, an attacker cannot impersonate the relay or decrypt the QUIC stream. The default Go client currently uses `InsecureSkipVerify: true` for development; in production you must supply a proper `*tls.Config` with verification enabled (see [examples/secure_conn/README.md](examples/secure_conn/README.md)).

2. **Application:** Payloads are end-to-end encrypted with NaCl box (Curve25519). Even if an attacker could observe or relay traffic, they only see ciphertext. Only the subscriber with the matching private key can decrypt. The relay itself never sees plaintext.

So: **with proper TLS configuration, we resist MITM because of certificate-authenticated QUIC plus E2EE.** A passive or active attacker on the path cannot read message contents.

### Eavesdropping by the relay

The relay is a **zero-knowledge** forwarder. It routes frames by topic and key ID only. Message payloads are encrypted for the subscriber’s public key before they leave the publisher; the relay never has the subscriber’s private key. So we are designed to stop the relay (or anyone who compromises only the relay) from reading message contents.

### Schema / format abuse on the wire

Publishers must send payloads that validate against the declared schema (e.g. `sensor.Temperature`). The server rejects invalid or malformed payloads with `SCHEMA_INVALID`. This does not prevent malicious *content* (e.g. lies in a temperature value), but it stops arbitrary blob injection at the protocol level.

### 0-RTT (session resumption)

The relay and client use QUIC 0-RTT so **returning devices** can send data in the first packet without a full handshake (TLS session tickets are cached). That reduces connection latency from ~2 RTTs to effectively 0 RTT for reconnects. First-time connections still perform a full handshake and receive a session ticket for future 0-RTT.

---

## What We Do *Not* Protect Against

### Compromised physical device or process

If an attacker has access to a device or process that holds a node’s **private key**, they can decrypt all messages for that node and impersonate it. We do not protect against device compromise, malware, or key extraction. Key storage and process isolation are the deployer’s responsibility.

### 0-RTT replay

Data sent in the 0-RTT phase is encrypted but **not forward-secure** and can be replayed by an attacker who captures it. The protocol uses 0-RTT for Subscribe and Publish. Subscribe is effectively idempotent; Publish may be replayed (duplicate delivery). Applications that require strict once-only semantics for publishes should use idempotency keys or accept replay risk for the first flight.

### Compromised relay (metadata)

A malicious or compromised relay cannot read payloads, but it **can** observe metadata: who connects, which topics are subscribed and published, message timing, and size. It can also drop, reorder, or selectively not forward messages. We do not hide metadata or guarantee availability.

### No authentication or authorization

The relay does **not** authenticate clients. Anyone who can reach the relay can subscribe to any topic and publish to any topic. There is no access control, no credentials, and no notion of “allowed publishers” or “allowed subscribers.” If you need auth, you must add it (e.g. in front of the relay or in your application layer).

### Key distribution and identity

Public keys are exchanged **out-of-band** (e.g. you share the subscriber’s public key hex with the publisher). There is no PKI, no certificate chain, and no built-in binding between keys and real-world identities. Mistaken or malicious key substitution (e.g. wrong key in a config) is not detected by the protocol. You are responsible for correct key distribution and verification.

### mDNS / P2P discovery

Discovery over mDNS is **unauthenticated**. Any host on the LAN can advertise a fake `_qumbed._udp` service with an arbitrary address and public key. We do not protect against discovery spoofing or malicious peers in P2P mode. Use `DisableDiscovery: true` and fixed relay addresses when you cannot trust the LAN.

### Forward secrecy (application layer)

E2EE uses long-term Curve25519 keys with NaCl box. We do not provide forward secrecy at the application layer: if a subscriber’s private key is later compromised, all past messages encrypted to that key can be decrypted. Session or ephemeral key agreement is not part of the current design.

### Denial of service and abuse

The relay does not implement rate limiting, quotas, or abuse controls. A client can subscribe to many topics, publish at high volume, or open many connections. We do not protect against DoS or resource exhaustion; that must be handled by deployment (e.g. network controls, reverse proxy, or relay-side logic you add).

### Replay and ordering

The protocol does not define message IDs or sequence numbers for application-level replay protection. A passive attacker who captures encrypted frames could replay them; depending on your schema and app logic, that may or may not matter. We do not guarantee ordering or exactly-once delivery.

---

## Summary

| Threat / scenario                         | Protected? | Why / why not |
|------------------------------------------|------------|----------------|
| MITM reading or altering payloads        | Yes*       | *With proper TLS; E2EE hides payload from path and relay. |
| Relay reading payloads                   | Yes        | Zero-knowledge relay; payloads are E2EE. |
| Relay or path seeing metadata            | No         | Topics, key IDs, timing, and size are visible. |
| Compromised device / stolen private key  | No         | Key holder can decrypt and impersonate. |
| Unauthorized subscribe/publish           | No         | No authZ; anyone who can reach relay can use any topic. |
| Wrong or spoofed public key              | No         | Key distribution is out-of-band; no PKI. |
| mDNS discovery spoofing                  | No         | Discovery is unauthenticated. |
| Replay of encrypted messages             | No         | No application-level replay protection. |
| DoS / abuse                              | No         | No rate limiting or access control in protocol. |

Use Qumbed when your threat model fits: you want confidentiality of payloads against the network and the relay, and you accept that metadata is visible, keys are managed by you, and device/relay compromise or abuse are outside the protocol’s scope. For stricter requirements (authZ, forward secrecy, replay protection), you will need to add or combine additional mechanisms.
