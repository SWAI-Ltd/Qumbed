#!/bin/bash
# Generate Go code from protobuf definitions
# Requires: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
set -e
cd "$(dirname "$0")/.."
mkdir -p internal/proto
protoc --go_out=internal/proto --go_opt=paths=source_relative \
  -I proto \
  proto/message.proto proto/schema.proto
