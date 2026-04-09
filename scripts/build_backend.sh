#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Compiles the Go backend binary into dist/.
# Uses: backend/
# Used by: scripts/build.sh, scripts/update_pi.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p "$ROOT_DIR/dist"

pushd "$ROOT_DIR/backend" >/dev/null
go mod tidy
go build -o "$ROOT_DIR/dist/iot-hub-backend" ./cmd/server
popd >/dev/null

echo "Backend build complete. Binary: $ROOT_DIR/dist/iot-hub-backend"
