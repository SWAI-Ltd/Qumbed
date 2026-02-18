#!/usr/bin/env bash
# Run a full release (binaries + Docker image). Requires Docker daemon running and GITHUB_TOKEN set.
set -e

cd "$(dirname "$0")/.."

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not running. Start Docker Desktop and run this script again."
  exit 1
fi

# Use a buildx builder that supports multi-platform (removes "unknown driver" warning)
if ! docker buildx inspect goreleaser >/dev/null 2>&1; then
  echo "Creating buildx builder 'goreleaser' for multi-platform Docker builds..."
  docker buildx create --name goreleaser --driver docker-container --use
  docker run --privileged --rm tonistiigi/binfmt --install all
else
  docker buildx use goreleaser 2>/dev/null || true
fi

if [ -z "$GITHUB_TOKEN" ]; then
  echo "GITHUB_TOKEN is not set. Export it to publish to GitHub and push the Docker image."
  exit 1
fi

# Log in to GitHub Container Registry so Docker can push ghcr.io/swai-ltd/qumbed
GITHUB_USER="${GITHUB_USER:-$(gh api user --jq .login 2>/dev/null)}"
if [ -z "$GITHUB_USER" ]; then
  echo "Set GITHUB_USER (your GitHub username) so we can log in to ghcr.io, or run: docker login ghcr.io -u YOUR_USERNAME"
  exit 1
fi
echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin

goreleaser release --clean
