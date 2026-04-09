#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys the compiled backend binary and schema, then restarts iot-hub if it is already running.
# Uses: dist/iot-hub-backend, backend/schema.sql
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for backend deployment"
    exit 1
  fi
  sudo "$@"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

run_root install -m 0755 "$ROOT_DIR/dist/iot-hub-backend" /usr/local/bin/iot-hub-backend
run_root cp "$ROOT_DIR/backend/schema.sql" /opt/iot-hub/backend/schema.sql

# Restart the service only if it is already active (i.e. not on first install).
if run_root systemctl is-active --quiet iot-hub; then
  run_root systemctl daemon-reload
  run_root systemctl restart iot-hub
  echo "Backend deployed and iot-hub service restarted."
else
  echo "Backend binary and schema deployed. Service not yet active — install_pi.sh will start it."
fi
