# Release assets

Pre-built **binaries** and **Docker image** for the Qumbed broker and CLI.

## Broker binary (relay)

Built for **Linux**, **macOS**, and **Windows** (amd64 and arm64).

- **relay** — the message broker (relay server)

Download the archive for your OS from the [Releases](https://github.com/qumbed/qumbed/releases) page, then:

```bash
# Linux/macOS
tar -xzf qumbed-<version>-<os>-<arch>.tar.gz
./relay -addr :6121

# Windows
# Unzip qumbed-<version>-windows-<arch>.zip, then:
relay.exe -addr :6121
```

## CLI tool (qumbed-check)

For testing topics and connections:

- **qumbed-check** — subscribe to a topic and validate incoming messages

```bash
# After extracting the CLI archive
./qumbed-check -topic test -relay localhost:6121
```

## Docker — run the broker in 5 seconds

Use the published image (after your first release):

```bash
docker pull ghcr.io/qumbed/qumbed:latest
docker run -p 6121:6121 ghcr.io/qumbed/qumbed:latest
```

Or with an explicit version:

```bash
docker pull ghcr.io/qumbed/qumbed:v1.0.0
docker run -p 6121:6121 ghcr.io/qumbed/qumbed:v1.0.0
```

The broker listens on **port 6121**. To use it from the host:

```bash
# Terminal 1: broker
docker run -p 6121:6121 ghcr.io/qumbed/qumbed:latest

# Terminal 2: CLI (using release binary or go run)
./qumbed-check -topic test -relay localhost:6121
```

## Building releases locally

1. Install [GoReleaser](https://goreleaser.com/install/).
2. **Start Docker Desktop** (required for building and pushing the broker image).
3. Create a Git tag and run:

```bash
export GITHUB_TOKEN=ghp_xxxx
./release/release.sh
```

Or manually:

```bash
git tag v1.0.0-alpha   # use semver, e.g. v1.0.0-alpha or v1.0.0
goreleaser release --clean
```

The script checks that Docker is running and sets up a multi-platform buildx builder so the Docker image builds correctly. If you don't need the Docker image this run, use:

```bash
goreleaser release --clean --skip=docker
```

For a dry run (no publish):

```bash
goreleaser release --snapshot
```

Artifacts are written to `dist/`.

## Image registry

The default image is **ghcr.io/qumbed/qumbed**. To push elsewhere (e.g. Docker Hub), set in `.goreleaser.yaml` under `dockers_v2`:

```yaml
images:
  - "docker.io/youruser/qumbed"
```

Then run `goreleaser release` as above.
