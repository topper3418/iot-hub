#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys built frontend assets to /opt/iot-hub/frontend/dist/. No service restart needed (Go FileServer reads from disk on each request).
# Uses: frontend/dist/ (produced by scripts/build_frontend.sh)
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

rsync -a --delete "$ROOT_DIR/frontend/dist/" /opt/iot-hub/frontend/dist/

echo "Frontend deployed to /opt/iot-hub/frontend/dist/"
