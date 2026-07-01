#!/usr/bin/env bash
# Cross-compile the `interest` CLI for both Mac architectures.
# Output goes to ./dist ready to upload to a GitHub Release.
set -euo pipefail

VERSION="${1:-dev}"
cd "$(dirname "$0")/cli"
mkdir -p ../dist

echo "Building interest ${VERSION}..."

GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${VERSION}" -o ../dist/interest_darwin_arm64 .
echo "  ✓ interest_darwin_arm64"

GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o ../dist/interest_darwin_amd64 .
echo "  ✓ interest_darwin_amd64"

echo "Done. Binaries in ./dist"
