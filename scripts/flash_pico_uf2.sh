#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Root helper that flashes a UF2 to a BOOTSEL-mounted Pico, mounting RPI-RP2 if necessary.
# Uses: none
# Used by: backend/internal/provision/provision.go, scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ "$EUID" -ne 0 ]]; then
  echo "This helper must run as root"
  exit 1
fi

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <path-to-uf2>"
  exit 1
fi

UF2_PATH="$1"
if [[ ! -f "$UF2_PATH" ]]; then
  echo "UF2 file not found: $UF2_PATH"
  exit 1
fi

LABEL_LINK="/dev/disk/by-label/RPI-RP2"
if [[ ! -e "$LABEL_LINK" ]]; then
  echo "BOOTSEL device label not found at $LABEL_LINK"
  exit 1
fi

DEVICE_PATH="$(readlink -f "$LABEL_LINK")"
if [[ -z "$DEVICE_PATH" || ! -b "$DEVICE_PATH" ]]; then
  echo "Resolved BOOTSEL device is not a block device: $DEVICE_PATH"
  exit 1
fi

MOUNTPOINT="$(lsblk -no MOUNTPOINT "$DEVICE_PATH" | head -n 1 | xargs)"
MOUNTED_BY_HELPER=0
TEMP_MOUNT="/tmp/iot-hub-rpi-rp2"

if [[ -z "$MOUNTPOINT" ]]; then
  mkdir -p "$TEMP_MOUNT"
  mount "$DEVICE_PATH" "$TEMP_MOUNT"
  MOUNTPOINT="$TEMP_MOUNT"
  MOUNTED_BY_HELPER=1
fi

cp "$UF2_PATH" "$MOUNTPOINT/$(basename "$UF2_PATH")"
sync

if [[ "$MOUNTED_BY_HELPER" -eq 1 ]]; then
  umount "$MOUNTPOINT"
fi

echo "UF2 flashed successfully to Pico BOOTSEL device"
