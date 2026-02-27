#!/usr/bin/env bash
# Run a full release (binaries + optional Docker image). Requires GITHUB_TOKEN set.
# Set SKIP_DOCKER=1 to publish only binaries (avoids 403 if you can't push to ghcr.io/swai-ltd/qumbed).
# To push Docker under your user: export GHCR_IMAGE=ghcr.io/YOUR_USERNAME/qumbed
set -e

cd "$(dirname "$0")/.."

if [ "${SKIP_DOCKER:-0}" != "1" ]; then
  if ! docker info >/dev/null 2>&1; then
    echo "Docker is not running. Start Docker Desktop, or run: SKIP_DOCKER=1 $0"
    exit 1
  fi
  if ! docker buildx inspect goreleaser >/dev/null 2>&1; then
    echo "Creating buildx builder 'goreleaser' for multi-platform Docker builds..."
    docker buildx create --name goreleaser --driver docker-container --use
    docker run --privileged --rm tonistiigi/binfmt --install all
  else
    docker buildx use goreleaser 2>/dev/null || true
  fi
  GITHUB_USER="${GITHUB_USER:-$(gh api user --jq .login 2>/dev/null)}"
  if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER (your GitHub username) for ghcr.io login, or run: SKIP_DOCKER=1 $0"
    exit 1
  fi
  echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin
fi

if [ -z "$GITHUB_TOKEN" ]; then
  echo "GITHUB_TOKEN is not set. Export it to publish to GitHub."
  exit 1
fi

GORELEASER_EXTRA=""
if [ "${SKIP_DOCKER:-0}" = "1" ]; then
  GORELEASER_EXTRA="--skip=docker"
  echo "Skipping Docker build/push (binaries only)."
fi

goreleaser release --clean $GORELEASER_EXTRA
