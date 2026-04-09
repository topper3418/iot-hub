-- Directory: backend/
-- Modified: 2026-04-08
-- Description: SQLite schema initialization. Creates rooms, devices, and led_strips tables on startup.
-- Uses: none
-- Used by: backend/internal/db/db.go

CREATE TABLE IF NOT EXISTS rooms (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS devices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  mac TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  room_id INTEGER NOT NULL,
  kind TEXT NOT NULL DEFAULT 'generic',
  last_seen TEXT NOT NULL,
  status_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS led_strips (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  device_id INTEGER NOT NULL UNIQUE,
  power INTEGER NOT NULL DEFAULT 0,
  brightness INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL DEFAULT '#000000',
  pixel_pin INTEGER NOT NULL DEFAULT 0,
  pixel_count INTEGER NOT NULL DEFAULT 255,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

INSERT OR IGNORE INTO rooms (id, name) VALUES (1, 'Unassigned');
