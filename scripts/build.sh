#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

pushd "$ROOT_DIR/frontend" >/dev/null
npm install
npm run build
popd >/dev/null

pushd "$ROOT_DIR/backend" >/dev/null
go mod tidy
go build -o "$ROOT_DIR/dist/iot-hub-backend" ./cmd/server
popd >/dev/null

echo "Build complete. Binary: $ROOT_DIR/dist/iot-hub-backend"
