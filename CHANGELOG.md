# Changelog

All notable changes to this project are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

(No changes yet.)

---

## [v1.0.0-beta] — mDNS and relay release

### Added

- **mDNS discovery** — Nodes on the same LAN discover each other via multicast DNS (`_qumbed._udp`). Publish and browse with optional TXT records (topics, public key). Use `NewBrowser()` for browse-only discovery. See [docs/mdns.md](docs/mdns.md).
- **Relay mode** — Zero-knowledge relay server for cross-network or relay-only deployments. Run `relay -addr :6121` and point nodes at it with `-relay`; use `-no-discovery` when mDNS is not desired (e.g. in containers).
- **Node binary in releases** — Pre-built `node` binary in GitHub Releases (Linux, macOS, Windows; amd64/arm64) for pub/sub with mDNS and/or relay.
- **mDNS example** — `examples/mdns_discovery`: publish and browse, or browse-only, to see peers on the LAN.
- **mDNS documentation** — [docs/mdns.md](docs/mdns.md) describes service type, TXT records, and usage.

### Changed

- Release assets include **relay**, **node**, and **qumbed-check** binaries plus the Docker relay image.

---

## [v1.0.0-alpha]

Initial alpha release: QUIC transport, E2EE (NaCl box), relay server, and CLI tooling.
