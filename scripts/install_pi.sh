#!/usr/bin/env bash
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root (sudo scripts/install_pi.sh)"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

apt-get update
apt-get install -y nginx mosquitto mosquitto-clients sqlite3 golang-go nodejs npm

id -u iotled >/dev/null 2>&1 || useradd --system --create-home --shell /usr/sbin/nologin iotled

mkdir -p /opt/iot-hub/backend /opt/iot-hub/frontend /var/lib/iot-hub /var/log/iot-hub
chown -R iotled:iotled /opt/iot-hub /var/lib/iot-hub /var/log/iot-hub

# Build locally and place artifacts.
"$ROOT_DIR/scripts/build.sh"

install -m 0755 "$ROOT_DIR/dist/iot-hub-backend" /usr/local/bin/iot-hub-backend
rsync -a --delete "$ROOT_DIR/frontend/dist/" /opt/iot-hub/frontend/dist/
cp "$ROOT_DIR/backend/schema.sql" /opt/iot-hub/backend/schema.sql

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
