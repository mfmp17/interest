#!/usr/bin/env bash
# Cross-compile the `fred.cash` CLI for both Mac architectures.
# Also emits legacy `interest` assets so cached/older installers keep working.
# Output goes to ./dist ready to upload to a GitHub Release.
set -euo pipefail

VERSION="${1:-dev}"
cd "$(dirname "$0")/cli"
mkdir -p ../dist

echo "Building fred.cash ${VERSION}..."

GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${VERSION}" -o ../dist/fred.cash_darwin_arm64 .
cp ../dist/fred.cash_darwin_arm64 ../dist/interest_darwin_arm64
echo "  ✓ fred.cash_darwin_arm64"
echo "  ✓ interest_darwin_arm64"

GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o ../dist/fred.cash_darwin_amd64 .
cp ../dist/fred.cash_darwin_amd64 ../dist/interest_darwin_amd64
echo "  ✓ fred.cash_darwin_amd64"
echo "  ✓ interest_darwin_amd64"

echo "Done. Binaries in ./dist"
