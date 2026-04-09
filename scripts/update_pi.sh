#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Pulls the latest changes from git and selectively rebuilds and redeploys only the components that changed.
# Uses: scripts/build_frontend.sh, scripts/build_backend.sh, scripts/deploy_frontend.sh, scripts/deploy_backend.sh, scripts/deploy_pico_assets.sh, scripts/flash_pico_uf2.sh, scripts/pico_push_manual_style.sh, scripts/read_host_wifi_creds.sh, deploy/mosquitto/iot-hub.conf
# Used by: none (run manually as a normal user on the Pi)
set -euo pipefail

run_root() {
  if [[ "$EUID" -eq 0 ]]; then
    "$@"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for privileged deploy commands"
    exit 1
  fi
  sudo "$@"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

deploy_flash_helper() {
  run_root install -m 0755 "$SCRIPTS_DIR/flash_pico_uf2.sh" /usr/local/bin/iot-hub-flash-uf2
  run_root install -m 0755 "$SCRIPTS_DIR/pico_push_manual_style.sh" /usr/local/bin/iot-hub-pico-push
  run_root install -m 0755 "$SCRIPTS_DIR/read_host_wifi_creds.sh" /usr/local/bin/iot-hub-read-wifi-creds
  run_root bash -c "cat > /etc/sudoers.d/iot-hub-flash-pico <<'EOF'
iotled ALL=(root) NOPASSWD: /usr/local/bin/iot-hub-flash-uf2 *
iotled ALL=(root) NOPASSWD: /usr/local/bin/iot-hub-read-wifi-creds *
EOF"
  run_root chmod 440 /etc/sudoers.d/iot-hub-flash-pico
}

harden_serial_access() {
  for grp in dialout plugdev uucp; do
    if getent group "$grp" >/dev/null 2>&1; then
      run_root usermod -a -G "$grp" iotled || true
    fi
  done
  for svc in ModemManager.service brltty.service serial-getty@ttyACM0.service; do
    if run_root systemctl list-unit-files "$svc" --no-legend >/dev/null 2>&1; then
      run_root systemctl disable --now "$svc" >/dev/null 2>&1 || true
    fi
  done
}

cd "$ROOT_DIR"

harden_serial_access

OLD_COMMIT="$(git rev-parse HEAD)"
echo "Current commit: $OLD_COMMIT"

git pull

NEW_COMMIT="$(git rev-parse HEAD)"

if [[ "$OLD_COMMIT" == "$NEW_COMMIT" ]]; then
  echo "Already up to date. Checking deployment/runtime prerequisites."
  CHANGED=""
else
  echo "Updated to commit: $NEW_COMMIT"
  CHANGED="$(git diff --name-only "$OLD_COMMIT" "$NEW_COMMIT")"
fi

REBUILD_FRONTEND=0
REBUILD_BACKEND=0
REDEPLOY_PICO=0
REDEPLOY_HELPERS=0
REDEPLOY_SYSTEMD=0
REDEPLOY_NGINX=0
REDEPLOY_MOSQUITTO=0

if echo "$CHANGED" | grep -q '^frontend/'; then
  REBUILD_FRONTEND=1
fi

if echo "$CHANGED" | grep -qE '^(backend/|go\.mod|go\.sum)'; then
  REBUILD_BACKEND=1
  REDEPLOY_HELPERS=1
  REDEPLOY_SYSTEMD=1
fi

if echo "$CHANGED" | grep -q '^pico/'; then
  REDEPLOY_PICO=1
fi

if echo "$CHANGED" | grep -qE '^scripts/(flash_pico_uf2\.sh|pico_push_manual_style\.sh|read_host_wifi_creds\.sh)$'; then
  REDEPLOY_HELPERS=1
fi

if [[ ! -x /usr/local/bin/iot-hub-flash-uf2 ]]; then
  REDEPLOY_HELPERS=1
fi

if [[ ! -x /usr/local/bin/iot-hub-pico-push ]]; then
  REDEPLOY_HELPERS=1
fi

if [[ ! -x /usr/local/bin/iot-hub-read-wifi-creds ]]; then
  REDEPLOY_HELPERS=1
fi

if [[ ! -f /etc/sudoers.d/iot-hub-flash-pico ]]; then
  REDEPLOY_HELPERS=1
fi

if echo "$CHANGED" | grep -q '^deploy/systemd/'; then
  REDEPLOY_SYSTEMD=1
fi

if echo "$CHANGED" | grep -q '^deploy/nginx/'; then
  REDEPLOY_NGINX=1
fi

if echo "$CHANGED" | grep -q '^deploy/mosquitto/'; then
  REDEPLOY_MOSQUITTO=1
fi

if [[ ! -f /etc/mosquitto/conf.d/iot-hub.conf ]]; then
  REDEPLOY_MOSQUITTO=1
fi

if [[ "$REBUILD_FRONTEND" -eq 0 && "$REBUILD_BACKEND" -eq 0 && "$REDEPLOY_PICO" -eq 0 && "$REDEPLOY_HELPERS" -eq 0 && "$REDEPLOY_SYSTEMD" -eq 0 && "$REDEPLOY_NGINX" -eq 0 && "$REDEPLOY_MOSQUITTO" -eq 0 ]]; then
  echo "No frontend, backend, pico, helper, or deploy config changes detected. Nothing to rebuild."
  exit 0
fi

if [[ "$REBUILD_FRONTEND" -eq 1 ]]; then
  echo "--- Rebuilding frontend ---"
  "$SCRIPTS_DIR/build_frontend.sh"
  "$SCRIPTS_DIR/deploy_frontend.sh"
fi

if [[ "$REBUILD_BACKEND" -eq 1 ]]; then
  echo "--- Rebuilding backend ---"
  "$SCRIPTS_DIR/build_backend.sh"
  "$SCRIPTS_DIR/deploy_backend.sh"
fi

if [[ "$REDEPLOY_PICO" -eq 1 ]]; then
  echo "--- Redeploying Pico assets ---"
  "$SCRIPTS_DIR/deploy_pico_assets.sh"
fi

if [[ "$REDEPLOY_HELPERS" -eq 1 ]]; then
  echo "--- Redeploying provisioning helper scripts ---"
  deploy_flash_helper
fi

if [[ "$REDEPLOY_SYSTEMD" -eq 1 ]]; then
  echo "--- Redeploying systemd unit ---"
  run_root cp "$ROOT_DIR/deploy/systemd/iot-hub.service" /etc/systemd/system/iot-hub.service
  run_root systemctl daemon-reload
  run_root systemctl restart iot-hub
  echo "iot-hub service restarted."
fi

if [[ "$REDEPLOY_NGINX" -eq 1 ]]; then
  echo "--- Redeploying nginx config ---"
  run_root cp "$ROOT_DIR/deploy/nginx/iot-hub.conf" /etc/nginx/sites-available/iot-hub.conf
  run_root ln -sf /etc/nginx/sites-available/iot-hub.conf /etc/nginx/sites-enabled/iot-hub.conf
  run_root systemctl reload nginx
  echo "nginx config reloaded."
fi

if [[ "$REDEPLOY_MOSQUITTO" -eq 1 ]]; then
  echo "--- Redeploying mosquitto config ---"
  run_root cp "$ROOT_DIR/deploy/mosquitto/iot-hub.conf" /etc/mosquitto/conf.d/iot-hub.conf
  run_root systemctl restart mosquitto
  echo "mosquitto restarted."
fi

echo "Update complete."
