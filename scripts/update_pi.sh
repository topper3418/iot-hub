#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Pulls the latest changes from git and selectively rebuilds and redeploys only the components that changed.
# Uses: scripts/build_frontend.sh, scripts/build_backend.sh, scripts/deploy_frontend.sh, scripts/deploy_backend.sh
# Used by: none (run manually as root on the Pi)
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "Run as root (sudo scripts/update_pi.sh)"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

cd "$ROOT_DIR"

OLD_COMMIT="$(git rev-parse HEAD)"
echo "Current commit: $OLD_COMMIT"

git pull

NEW_COMMIT="$(git rev-parse HEAD)"

if [[ "$OLD_COMMIT" == "$NEW_COMMIT" ]]; then
  echo "Already up to date. Nothing to do."
  exit 0
fi

echo "Updated to commit: $NEW_COMMIT"

CHANGED="$(git diff --name-only "$OLD_COMMIT" "$NEW_COMMIT")"

REBUILD_FRONTEND=0
REBUILD_BACKEND=0
REDEPLOY_SYSTEMD=0
REDEPLOY_NGINX=0

if echo "$CHANGED" | grep -q '^frontend/'; then
  REBUILD_FRONTEND=1
fi

if echo "$CHANGED" | grep -qE '^(backend/|go\.mod|go\.sum)'; then
  REBUILD_BACKEND=1
fi

if echo "$CHANGED" | grep -q '^deploy/systemd/'; then
  REDEPLOY_SYSTEMD=1
fi

if echo "$CHANGED" | grep -q '^deploy/nginx/'; then
  REDEPLOY_NGINX=1
fi

if [[ "$REBUILD_FRONTEND" -eq 0 && "$REBUILD_BACKEND" -eq 0 && "$REDEPLOY_SYSTEMD" -eq 0 && "$REDEPLOY_NGINX" -eq 0 ]]; then
  echo "No frontend, backend, or deploy config changes detected. Nothing to rebuild."
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

if [[ "$REDEPLOY_SYSTEMD" -eq 1 ]]; then
  echo "--- Redeploying systemd unit ---"
  cp "$ROOT_DIR/deploy/systemd/iot-hub.service" /etc/systemd/system/iot-hub.service
  systemctl daemon-reload
  systemctl restart iot-hub
  echo "iot-hub service restarted."
fi

if [[ "$REDEPLOY_NGINX" -eq 1 ]]; then
  echo "--- Redeploying nginx config ---"
  cp "$ROOT_DIR/deploy/nginx/iot-hub.conf" /etc/nginx/sites-available/iot-hub.conf
  ln -sf /etc/nginx/sites-available/iot-hub.conf /etc/nginx/sites-enabled/iot-hub.conf
  systemctl reload nginx
  echo "nginx config reloaded."
fi

echo "Update complete."
