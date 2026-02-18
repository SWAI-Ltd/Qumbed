# Secure Connection (TLS / E2EE)

Qumbed uses two layers of security:

1. **Transport:** QUIC mandates TLS 1.3. By default the relay and node use **self-signed certificates** (development only). For production you must supply your own certificates.

2. **Application (E2EE):** Payloads are encrypted with the subscriber's public key (NaCl box). The relay never sees plaintext.

## Loading your own certificates (production)

The QUIC transport in `internal/transport/quic.go` currently uses `generateTLSConfig()` for self-signed certs. To use your own certs:

1. **Relay:** Set `TLS_CERT_FILE` and `TLS_KEY_FILE` (or pass flags). The relay would need to be updated to accept a custom `*tls.Config` in the listener.

2. **Client:** For production, do **not** use `InsecureSkipVerify: true`. Instead:
   - Use a real CA and server certificate hostname.
   - Or pin the relay's public key (e.g. certificate fingerprint).

Example of loading certificates in Go (for use when we add configurable TLS):

```go
cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
if err != nil {
    log.Fatal(err)
}
cfg := &tls.Config{
    Certificates: []tls.Certificate{cert},
    NextProtos:   []string{"qumbed/1"},
    // MinVersion: tls.VersionTLS13,
}
// For client dial:
// cfg.ServerName = "relay.example.com"
// cfg.InsecureSkipVerify = false  // use real CA
```

## E2EE (always on)

- Each node has a Curve25519 key pair. Share the **public key** (hex) with publishers.
- Publishers call `client.Publish(ctx, topic, schemaID, payload, subscriberPublicKey)`.
- Only the subscriber can decrypt; the relay only routes by topic and key ID.

This example runs the same simple pub/sub but documents how security works. Run as in `examples/simple_pubsub`.
