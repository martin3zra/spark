#!/usr/bin/env bash
set -euo pipefail

# Build the frontend assets, then the web and CLI binaries (prod tag).
echo "==> building frontend"
npm ci
npm run build

echo "==> building web binary"
go build -tags prod -o bin/spark .

echo "==> building cli binary"
go build -tags prod -o bin/cli ./cmd/cli

echo "==> done: bin/spark, bin/cli"
