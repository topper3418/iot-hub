#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Deploys Pico firmware source assets used by backend provisioning.
# Uses: pico/
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for pico asset deployment"
    exit 1
  fi
  sudo "$@"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

run_root mkdir -p /opt/iot-hub/pico
run_root rsync -a --delete "$ROOT_DIR/pico/" /opt/iot-hub/pico/

UF2_SOURCE="$(find "$ROOT_DIR/pico" -maxdepth 1 -type f -name '*.uf2' | head -n 1)"
if [[ -n "$UF2_SOURCE" ]]; then
  run_root cp "$UF2_SOURCE" /opt/iot-hub/pico-rp2.uf2
  echo "UF2 copied to /opt/iot-hub/pico-rp2.uf2 from $(basename "$UF2_SOURCE")"
else
  echo "No UF2 found in $ROOT_DIR/pico; provisioning in BOOTSEL mode will fail until one is added."
fi

echo "Pico assets deployed to /opt/iot-hub/pico/"
