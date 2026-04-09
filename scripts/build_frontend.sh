#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Builds the React frontend assets into frontend/dist/.
# Uses: frontend/
# Used by: scripts/build.sh, scripts/update_pi.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

pushd "$ROOT_DIR/frontend" >/dev/null
npm install
npm run build
popd >/dev/null

echo "Frontend build complete. Assets: $ROOT_DIR/frontend/dist/"
