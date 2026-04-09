#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: First-time installation of the IoT hub on a Raspberry Pi. Installs dependencies, creates user/dirs, builds, deploys, and starts services.
# Uses: scripts/build_frontend.sh, scripts/build_backend.sh, scripts/deploy_frontend.sh, scripts/deploy_backend.sh, deploy/systemd/iot-hub.service, deploy/nginx/iot-hub.conf
# Used by: none (run manually as root on the Pi)
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root (sudo scripts/install_pi.sh)"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

apt-get update
apt-get install -y nginx mosquitto mosquitto-clients sqlite3 golang-go nodejs npm

id -u iotled >/dev/null 2>&1 || useradd --system --create-home --shell /usr/sbin/nologin iotled

mkdir -p /opt/iot-hub/backend /opt/iot-hub/frontend /var/lib/iot-hub /var/log/iot-hub
chown -R iotled:iotled /opt/iot-hub /var/lib/iot-hub /var/log/iot-hub

"$SCRIPTS_DIR/build_frontend.sh"
"$SCRIPTS_DIR/build_backend.sh"

"$SCRIPTS_DIR/deploy_frontend.sh"
"$SCRIPTS_DIR/deploy_backend.sh"

cp "$ROOT_DIR/deploy/systemd/iot-hub.service" /etc/systemd/system/iot-hub.service
cp "$ROOT_DIR/deploy/nginx/iot-hub.conf" /etc/nginx/sites-available/iot-hub.conf
ln -sf /etc/nginx/sites-available/iot-hub.conf /etc/nginx/sites-enabled/iot-hub.conf
rm -f /etc/nginx/sites-enabled/default

systemctl daemon-reload
systemctl enable mosquitto
systemctl enable iot-hub
systemctl restart mosquitto
systemctl restart iot-hub
systemctl restart nginx

echo "Optional: generate Pico config with ./scripts/generate_pico_config.sh"
echo "Install complete. Open http://<raspberry-pi-ip>/"
