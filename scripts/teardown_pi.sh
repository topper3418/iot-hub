#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Removes the deployed service, nginx config, and installed artifacts from the Raspberry Pi.
# Uses: none
# Used by: none (run manually as root on the Pi)
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root (sudo scripts/teardown_pi.sh)"
  exit 1
fi

systemctl disable --now iot-hub || true
systemctl disable --now mosquitto || true
systemctl restart nginx || true

rm -f /etc/systemd/system/iot-hub.service
rm -f /etc/nginx/sites-enabled/iot-hub.conf
rm -f /etc/nginx/sites-available/iot-hub.conf
rm -f /usr/local/bin/iot-hub-backend

rm -rf /opt/iot-hub /var/lib/iot-hub /var/log/iot-hub

if id -u iotled >/dev/null 2>&1; then
  userdel iotled || true
fi

systemctl daemon-reload
systemctl restart nginx || true

echo "Teardown complete."
