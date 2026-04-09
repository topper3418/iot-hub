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

UF2_SOURCE=""
while IFS= read -r candidate; do
  base="$(basename "$candidate")"
  lower_base="$(echo "$base" | tr '[:upper:]' '[:lower:]')"
  if [[ "$lower_base" == *"pico_w"* ]] || [[ "$lower_base" == *"picow"* ]]; then
    UF2_SOURCE="$candidate"
    break
  fi
done < <(find "$ROOT_DIR/pico" -maxdepth 1 -type f -name '*.uf2' | sort)

if [[ -z "$UF2_SOURCE" ]]; then
  if find "$ROOT_DIR/pico" -maxdepth 1 -type f -name '*.uf2' | grep -q .; then
    echo "ERROR: UF2 found, but none looks like Pico W firmware (expected filename containing PICO_W or PICOW)." >&2
    echo "This causes 'ImportError: no module named network' on Pico W projects." >&2
    echo "Place a Pico W MicroPython UF2 in $ROOT_DIR/pico and rerun this script." >&2
    exit 1
  fi
  echo "No UF2 found in $ROOT_DIR/pico; provisioning in BOOTSEL mode will fail until one is added."
else
  run_root cp "$UF2_SOURCE" /opt/iot-hub/pico-rp2.uf2
  echo "UF2 copied to /opt/iot-hub/pico-rp2.uf2 from $(basename "$UF2_SOURCE")"
fi

echo "Pico assets deployed to /opt/iot-hub/pico/"
