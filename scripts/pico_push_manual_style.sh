#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Thonny-like manual upload helper: ensures umqtt dependency, pushes main.py/device_config.py, and resets Pico.
# Uses: none
# Used by: backend/internal/provision/provision.go, scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <preferred-serial-port|auto> <main.py> <device_config.py>" >&2
  exit 2
fi

PREFERRED_PORT="$1"
MAIN_PY="$2"
CFG_PY="$3"

if [[ ! -f "$MAIN_PY" ]]; then
  echo "main.py not found: $MAIN_PY" >&2
  exit 2
fi
if [[ ! -f "$CFG_PY" ]]; then
  echo "device_config.py not found: $CFG_PY" >&2
  exit 2
fi
if ! command -v mpremote >/dev/null 2>&1; then
  echo "mpremote is not installed" >&2
  exit 2
fi

candidates() {
  if [[ "$PREFERRED_PORT" != "auto" && -n "$PREFERRED_PORT" ]]; then
    echo "$PREFERRED_PORT"
  fi
  ls /dev/ttyACM* 2>/dev/null || true
}

LAST_ERR=""
for ATTEMPT in 1 2 3 4 5 6; do
  while IFS= read -r PORT; do
    [[ -z "$PORT" ]] && continue

    echo "[pico-push] attempt $ATTEMPT using $PORT" >&2

    if ! mpremote connect "$PORT" exec "import umqtt.simple" >/dev/null 2>&1; then
      echo "[pico-push] umqtt.simple missing on $PORT, installing via mip" >&2
      if ! OUT=$(mpremote connect "$PORT" mip install umqtt.simple 2>&1); then
        LAST_ERR="failed to install umqtt.simple: $OUT"
        continue
      fi
    fi

    if ! OUT=$(mpremote connect "$PORT" fs cp "$MAIN_PY" :main.py 2>&1); then
      LAST_ERR="$OUT"
      continue
    fi

    if ! OUT=$(mpremote connect "$PORT" fs cp "$CFG_PY" :device_config.py 2>&1); then
      LAST_ERR="$OUT"
      continue
    fi

    if ! OUT=$(mpremote connect "$PORT" reset 2>&1); then
      LAST_ERR="$OUT"
      continue
    fi

    echo "[pico-push] upload complete via $PORT" >&2
    exit 0

    LAST_ERR="$OUT"
  done < <(candidates)

  sleep 1
done

echo "[pico-push] failed after retries" >&2
if [[ -n "$LAST_ERR" ]]; then
  echo "$LAST_ERR" >&2
fi

if command -v lsof >/dev/null 2>&1; then
  for PORT in /dev/ttyACM*; do
    [[ -e "$PORT" ]] || continue
    echo "[pico-push] lsof $PORT" >&2
    lsof "$PORT" >&2 || true
  done
fi

exit 1
