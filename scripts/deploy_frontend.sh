#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys built frontend assets to /opt/iot-hub/frontend/dist/. No service restart needed (Go FileServer reads from disk on each request).
# Uses: frontend/dist/ (produced by scripts/build_frontend.sh)
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for frontend deployment"
    exit 1
  fi
  sudo "$@"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

run_root rsync -a --delete "$ROOT_DIR/frontend/dist/" /opt/iot-hub/frontend/dist/

echo "Frontend deployed to /opt/iot-hub/frontend/dist/"
