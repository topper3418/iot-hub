#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys the compiled backend binary and schema, then restarts iot-hub if it is already running.
# Uses: dist/iot-hub-backend, backend/schema.sql
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

install -m 0755 "$ROOT_DIR/dist/iot-hub-backend" /usr/local/bin/iot-hub-backend
cp "$ROOT_DIR/backend/schema.sql" /opt/iot-hub/backend/schema.sql

# Restart the service only if it is already active (i.e. not on first install).
if systemctl is-active --quiet iot-hub; then
  systemctl daemon-reload
  systemctl restart iot-hub
  echo "Backend deployed and iot-hub service restarted."
else
  echo "Backend binary and schema deployed. Service not yet active — install_pi.sh will start it."
fi
