#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Removes the deployed service, nginx config, and installed artifacts from the Raspberry Pi.
# Uses: none
# Used by: none (run manually as a normal user on the Pi)
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for teardown"
    exit 1
  fi
  sudo "$@"
}

run_root systemctl disable --now iot-hub || true
run_root systemctl disable --now mosquitto || true
run_root systemctl restart nginx || true

run_root rm -f /etc/systemd/system/iot-hub.service
run_root rm -f /etc/nginx/sites-enabled/iot-hub.conf
run_root rm -f /etc/nginx/sites-available/iot-hub.conf
run_root rm -f /usr/local/bin/iot-hub-backend

run_root rm -rf /opt/iot-hub /var/lib/iot-hub /var/log/iot-hub

if run_root id -u iotled >/dev/null 2>&1; then
  run_root userdel iotled || true
fi

run_root systemctl daemon-reload
run_root systemctl restart nginx || true

echo "Teardown complete."
