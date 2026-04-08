# Header
Directory: agent-notes/
Modified: 2026-04-08
Description: Design patterns and development rules that guide changes in this project.
Uses: agent-notes/agent-seed, agent-notes/project-overview-for-agent.md, backend/internal/db/db.go, backend/internal/mqtt/payload.go, backend/internal/model/types.go, backend/internal/app/server.go, frontend/src/pages/DeviceListPage.jsx, frontend/src/pages/DeviceControlPage.jsx
Used by: AI coding agents, contributors implementing features or refactors

# Design Patterns And Development Rules
## 1) Layer delineation
- Keep MQTT concerns in the mqtt package.
- Keep persistence concerns in the db package.
- Keep HTTP routing/orchestration in the app package.
- Avoid cross-layer leakage of parsing or SQL logic.

## 2) Generic base + type-specific extension
- Shared device identity and lifecycle data belongs in devices.
- Type-specific capabilities belong in dedicated tables (for now: led_strips).
- New device types should follow the same pattern with their own table and handlers.

## 3) Upsert orchestration pattern
- Use a small orchestrator function per device type.
- Example shape:
  1. upsert base device
  2. upsert type-specific state
- Do not place all type branches in a single giant function.

## 4) Naming conventions
- Use type-specific names where behavior is type-specific.
- LED-specific payloads/commands should use LED-prefixed model names.

## 5) Command handling rules
- Validate command bounds in mqtt payload helpers before publish.
- Persist command side effects for LED strips in led_strips.
- Keep QoS 1 for reliability.

## 6) Frontend rules
- Present LED controls as the primary workflow today.
- Keep model shape ready for non-LED kinds without hard-coding a single-table assumption.

## 7) Pico rules
- Keep firmware simple and resilient.
- Keep runtime credentials and broker/pin config in pico/device_config.py.
- Generate config via script; do not hard-code local environment values in main.py.
- Pixel count remains fixed at 255 unless requirements change.

## 8) Change safety checklist
- If schema or model changes, update db queries and frontend consumers in the same change set.
- Build backend after Go refactors.
- Build frontend after API shape changes.
- Preserve backward-compatible separation even when implementing LED-first behavior.
