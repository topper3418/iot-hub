#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: First-time installation of the IoT hub on a Raspberry Pi. Installs dependencies, creates user/dirs, builds, deploys, and starts services.
# Uses: scripts/build_frontend.sh, scripts/build_backend.sh, scripts/deploy_frontend.sh, scripts/deploy_backend.sh, scripts/deploy_pico_assets.sh, scripts/flash_pico_uf2.sh, deploy/systemd/iot-hub.service, deploy/nginx/iot-hub.conf
# Used by: none (run manually as a normal user on the Pi)
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for privileged setup commands"
    exit 1
  fi
  sudo "$@"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

deploy_flash_helper() {
  run_root install -m 0755 "$SCRIPTS_DIR/flash_pico_uf2.sh" /usr/local/bin/iot-hub-flash-uf2
  run_root bash -c "cat > /etc/sudoers.d/iot-hub-flash-pico <<'EOF'
iotled ALL=(root) NOPASSWD: /usr/local/bin/iot-hub-flash-uf2 *
EOF"
  run_root chmod 440 /etc/sudoers.d/iot-hub-flash-pico
}

run_root apt-get update
run_root apt-get install -y nginx mosquitto mosquitto-clients sqlite3 golang-go nodejs npm python3 python3-pip wireless-tools

if ! command -v mpremote >/dev/null 2>&1; then
  run_root pip3 install --break-system-packages mpremote
fi

if ! run_root id -u iotled >/dev/null 2>&1; then
  run_root useradd --system --create-home --shell /usr/sbin/nologin iotled
fi

run_root mkdir -p /opt/iot-hub/backend /opt/iot-hub/frontend /var/lib/iot-hub /var/log/iot-hub
run_root chown -R iotled:iotled /opt/iot-hub /var/lib/iot-hub /var/log/iot-hub

"$SCRIPTS_DIR/build_frontend.sh"
"$SCRIPTS_DIR/build_backend.sh"

"$SCRIPTS_DIR/deploy_frontend.sh"
"$SCRIPTS_DIR/deploy_backend.sh"
"$SCRIPTS_DIR/deploy_pico_assets.sh"
deploy_flash_helper

run_root cp "$ROOT_DIR/deploy/systemd/iot-hub.service" /etc/systemd/system/iot-hub.service
run_root cp "$ROOT_DIR/deploy/nginx/iot-hub.conf" /etc/nginx/sites-available/iot-hub.conf
run_root ln -sf /etc/nginx/sites-available/iot-hub.conf /etc/nginx/sites-enabled/iot-hub.conf
run_root rm -f /etc/nginx/sites-enabled/default

run_root systemctl daemon-reload
run_root systemctl enable mosquitto
run_root systemctl enable iot-hub
run_root systemctl restart mosquitto
run_root systemctl restart iot-hub
run_root systemctl restart nginx

echo "Optional: place MicroPython UF2 at /opt/iot-hub/pico-rp2.uf2 for BOOTSEL auto-flash"
echo "Install complete. Open http://<raspberry-pi-ip>/"
