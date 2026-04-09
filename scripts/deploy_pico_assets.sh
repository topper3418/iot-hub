#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys Pico firmware source assets used by backend provisioning.
# Uses: pico/
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p /opt/iot-hub/pico
rsync -a --delete "$ROOT_DIR/pico/" /opt/iot-hub/pico/

echo "Pico assets deployed to /opt/iot-hub/pico/"
