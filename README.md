# Home LED Control System

This repository contains a complete home automation control stack:

- Raspberry Pi backend in Go using SQLite and Mosquitto MQTT.
- React + Ant Design frontend served by the backend.
- Pico W MicroPython LED strip client that publishes full status every 8 seconds.
- Pi deployment automation with systemd and nginx.

## Folder Layout

- `backend/` Go API server, SQLite schema, MQTT handling.
- `frontend/` React + Ant Design UI.
- `pico/` MicroPython firmware for Pico W.
- `deploy/` systemd + nginx files for Raspberry Pi.
- `scripts/` build/install/teardown scripts.

## Behavior Implemented

- Picos publish to `devices/status/{mac}` every 8 seconds at QoS 1.
- Backend subscribes to `devices/status/#` at QoS 1.
- First sighting of MAC auto-adds a generic device into SQLite with defaults.
- LED-specific fields are stored in `led_strips` linked to `devices`.
- Frontend supports device list, room management, and per-device control.
- Commands publish to `devices/cmd/{mac}` at QoS 1.

## Local Development

1. Start Mosquitto:
   - `mosquitto -v`
2. Start backend:
   - `cd backend && go run ./cmd/server`
3. Start frontend dev server:
   - `cd frontend && npm install && npm run dev`
4. Open `http://localhost:5173`.

## Build

Run:

- `./scripts/build.sh`

This produces:

- Backend binary at `dist/iot-hub-backend`
- Frontend static build at `frontend/dist`

## Raspberry Pi Install

1. Copy repository to Pi.
2. Run:
   - `sudo ./scripts/install_pi.sh`
3. Open:
   - `http://<pi-ip>/`

The installer does the following:

- Installs packages (`nginx`, `mosquitto`, `sqlite3`, `golang-go`, `nodejs`, `npm`).
- Creates `iotled` service user.
- Builds frontend and backend.
- Installs backend binary at `/usr/local/bin/iot-hub-backend`.
- Syncs frontend files to `/opt/iot-hub/frontend/dist`.
- Installs systemd and nginx config.
- Enables/restarts `mosquitto`, `iot-hub`, and `nginx`.

## Raspberry Pi Teardown

Run:

- `sudo ./scripts/teardown_pi.sh`

This removes service/config artifacts and app directories.

## Pico W Setup

1. Generate Pico config from shell (auto-detects broker IP and suggests SSID):
   - `./scripts/generate_pico_config.sh`
2. Copy both files to Pico W:
   - `pico/device_config.py`
   - `pico/main.py`
3. Reset Pico.

The Pico subscribes to `devices/cmd/{mac}` and publishes status on `devices/status/{mac}`.
