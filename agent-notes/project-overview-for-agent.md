# Header
Directory: agent-notes/
Modified: 2026-04-08
Description: Standalone project briefing for AI agents with no chat history context.
Uses: README.md, backend/schema.sql, backend/internal/app/server.go, backend/internal/db/db.go, backend/internal/mqtt/client.go, backend/internal/mqtt/payload.go, backend/internal/model/types.go, frontend/src/pages/DeviceListPage.jsx, frontend/src/pages/DeviceControlPage.jsx, pico/main.py, pico/device_config.py, scripts/build.sh, scripts/install_pi.sh, scripts/teardown_pi.sh, scripts/generate_pico_config.sh
Used by: AI coding agents, maintainers onboarding to the repository

# Project Overview For Agents
This repository is a home automation control system currently focused on LED strip devices.

Core runtime components:
- Raspberry Pi hosts the Go backend service.
- SQLite stores device metadata and room assignments.
- Mosquitto is the MQTT broker.
- Frontend is React with Ant Design and is served by the backend.
- Pico W runs MicroPython firmware for LED strips.

Current data model:
- devices table is generic and stores shared device fields (mac, name, room, kind, last seen, raw status).
- led_strips table stores LED-specific fields (power, brightness, color, pixel pin, pixel count).
- Current operational focus is LED strips, but schema separation is intentionally future-compatible.

MQTT contract:
- Status topic: devices/status/{mac}
- Command topic: devices/cmd/{mac}
- QoS is 1 for status subscribe and command publish.
- Pico publishes full status periodically (every 8 seconds).

Backend architecture summary:
- MQTT package handles MQTT client, status parsing, and command payload building.
- DB package handles persistence and query logic.
- App package handles HTTP API wiring and orchestration across MQTT and DB.

Pico configuration:
- Runtime config is split into pico/device_config.py.
- scripts/generate_pico_config.sh generates Pico config values by combining shell discovery and user prompts.
- User selects pixel pin; pixel count is fixed at 255.

Frontend summary:
- Device list page supports rename, room assignment, and LED controls.
- Device details page supports power, brightness, color, and pixel pin commands.

Deployment flow:
- scripts/build.sh builds frontend assets and backend binary.
- scripts/install_pi.sh installs dependencies and deploys systemd/nginx config.
- scripts/teardown_pi.sh removes deployed artifacts.

Primary expectation for future changes:
- Preserve generic-vs-type-specific delineation.
- Keep LED behavior working by default while enabling future device kinds to be added without large rewrites.
