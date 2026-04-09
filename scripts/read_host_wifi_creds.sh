#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Root helper that reads active host WiFi SSID and password from NetworkManager or wpa_supplicant.
# Uses: none
# Used by: backend/internal/provision/provision.go, scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "This helper must run as root" >&2
  exit 1
fi

SSID_HINT="${1:-}"
SSID=""
PASSWORD=""

if command -v nmcli >/dev/null 2>&1; then
  ACTIVE_NAME="$(nmcli -t -f NAME,TYPE connection show --active 2>/dev/null | awk -F: '$2=="802-11-wireless"{print $1; exit}')"
  if [[ -n "$ACTIVE_NAME" ]]; then
    SSID="$(nmcli -g 802-11-wireless.ssid connection show "$ACTIVE_NAME" 2>/dev/null | head -n 1 | xargs || true)"
    PASSWORD="$(nmcli -s -g 802-11-wireless-security.psk connection show "$ACTIVE_NAME" 2>/dev/null | head -n 1 | xargs || true)"
  fi
fi

if [[ -z "$SSID" && -r /etc/wpa_supplicant/wpa_supplicant.conf ]]; then
  if [[ -n "$SSID_HINT" ]]; then
    SSID="$SSID_HINT"
    PASSWORD="$(awk -v ssid="$SSID_HINT" '
      $0 ~ "network=\\{" {in_network=1; cur_ssid=""; cur_psk=""; next}
      in_network && $0 ~ /^\\}/ {
        if (cur_ssid == ssid && cur_psk != "") {print cur_psk; exit}
        in_network=0; next
      }
      in_network && $0 ~ /^[[:space:]]*ssid=/ {gsub(/.*ssid=\"|\".*/, "", $0); cur_ssid=$0; next}
      in_network && $0 ~ /^[[:space:]]*psk=/ {gsub(/.*psk=\"|\".*/, "", $0); cur_psk=$0; next}
    ' /etc/wpa_supplicant/wpa_supplicant.conf || true)"
  else
    SSID="$(awk '/^[[:space:]]*ssid=/{gsub(/.*ssid=\"|\".*/, "", $0); print $0; exit}' /etc/wpa_supplicant/wpa_supplicant.conf || true)"
    PASSWORD="$(awk '/^[[:space:]]*psk=/{gsub(/.*psk=\"|\".*/, "", $0); print $0; exit}' /etc/wpa_supplicant/wpa_supplicant.conf || true)"
  fi
fi

echo "SSID=$SSID"
echo "PASSWORD=$PASSWORD"
