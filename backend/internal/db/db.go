// Directory: backend/internal/db/
// Modified: 2026-04-08
// Description: SQLite persistence layer. Handles device, LED strip, and room upsert and query operations.
// Uses: backend/internal/model/types.go, backend/schema.sql
// Used by: backend/internal/app/server.go

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"iot-hub/backend/internal/model"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("read schema.sql: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) UpsertDevice(mac, name, kind string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "led_strip"
	}

	stmt := `
	INSERT INTO devices (mac, name, room_id, kind, last_seen, status_json, created_at, updated_at)
	VALUES (?, ?, 1, ?, ?, ?, ?, ?)
	ON CONFLICT(mac) DO UPDATE SET
		kind = excluded.kind,
		last_seen = excluded.last_seen,
		status_json = excluded.status_json,
		updated_at = excluded.updated_at
	`

	if _, err := s.db.Exec(stmt, mac, name, kind, now, "", now, now); err != nil {
		return 0, fmt.Errorf("upsert device: %w", err)
	}

	var deviceID int64
	if err := s.db.QueryRow(`SELECT id FROM devices WHERE mac = ?`, mac).Scan(&deviceID); err != nil {
		return 0, fmt.Errorf("get device id: %w", err)
	}
	return deviceID, nil
}

func (s *Store) UpsertLEDStrip(deviceID int64, status model.LEDStatusUpdate) error {
	now := time.Now().UTC().Format(time.RFC3339)

	power := false
	if status.Power != nil {
		power = *status.Power
	}
	brightness := 0
	if status.Brightness != nil {
		brightness = *status.Brightness
	}
	color := "#000000"
	if status.Color != nil {
		color = *status.Color
	}
	pixelPin := 0
	if status.PixelPin != nil {
		pixelPin = *status.PixelPin
	}

	ledStmt := `
	INSERT INTO led_strips (device_id, power, brightness, color, pixel_pin, pixel_count, updated_at)
	VALUES (?, ?, ?, ?, ?, 255, ?)
	ON CONFLICT(device_id) DO UPDATE SET
		power = excluded.power,
		brightness = excluded.brightness,
		color = excluded.color,
		pixel_pin = excluded.pixel_pin,
		updated_at = excluded.updated_at
	`
	_, err := s.db.Exec(ledStmt, deviceID, boolToInt(power), brightness, color, pixelPin, now)
	return err
}

func (s *Store) UpsertLEDDevice(mac string, status model.LEDStatusUpdate) error {
	defaultName := fmt.Sprintf("Device %s", suffix(mac))
	deviceID, err := s.UpsertDevice(mac, defaultName, status.Kind)
	if err != nil {
		return err
	}
	return s.UpsertLEDStrip(deviceID, status)
}

func (s *Store) ListDevices() ([]model.Device, error) {
	rows, err := s.db.Query(`
	SELECT d.id, d.mac, d.name, d.room_id, r.name, d.kind, d.last_seen, d.status_json,
	       ls.power, ls.brightness, ls.color, ls.pixel_pin, ls.pixel_count
	FROM devices d
	JOIN rooms r ON r.id = d.room_id
	LEFT JOIN led_strips ls ON ls.device_id = d.id
	ORDER BY r.name, d.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := []model.Device{}
	for rows.Next() {
		var d model.Device
		var ledPower sql.NullInt64
		var ledBrightness sql.NullInt64
		var ledColor sql.NullString
		var ledPixelPin sql.NullInt64
		var ledPixelCount sql.NullInt64
		if err := rows.Scan(
			&d.ID,
			&d.MAC,
			&d.Name,
			&d.RoomID,
			&d.RoomName,
			&d.Kind,
			&d.LastSeen,
			&d.StatusJSON,
			&ledPower,
			&ledBrightness,
			&ledColor,
			&ledPixelPin,
			&ledPixelCount,
		); err != nil {
			return nil, err
		}
		if d.Kind == "led_strip" {
			power := ledPower.Valid && ledPower.Int64 == 1
			brightness := 0
			if ledBrightness.Valid {
				brightness = int(ledBrightness.Int64)
			}
			color := "#000000"
			if ledColor.Valid {
				color = ledColor.String
			}
			pixelPin := 0
			if ledPixelPin.Valid {
				pixelPin = int(ledPixelPin.Int64)
			}
			pixelCount := 255
			if ledPixelCount.Valid {
				pixelCount = int(ledPixelCount.Int64)
			}
			d.LEDStrip = &model.LEDStrip{
				Power:      power,
				Brightness: brightness,
				Color:      color,
				PixelPin:   pixelPin,
				PixelCount: pixelCount,
			}
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *Store) ListRooms() ([]model.Room, error) {
	rows, err := s.db.Query(`SELECT id, name FROM rooms ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := []model.Room{}
	for rows.Next() {
		var r model.Room
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}

func (s *Store) UpdateDevice(mac string, patch model.DevicePatch) error {
	if patch.Name != nil {
		if _, err := s.db.Exec(`UPDATE devices SET name = ?, updated_at = ? WHERE mac = ?`, strings.TrimSpace(*patch.Name), time.Now().UTC().Format(time.RFC3339), mac); err != nil {
			return err
		}
	}
	if patch.RoomID != nil {
		if _, err := s.db.Exec(`UPDATE devices SET room_id = ?, updated_at = ? WHERE mac = ?`, *patch.RoomID, time.Now().UTC().Format(time.RFC3339), mac); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ApplyLEDCommand(mac string, cmd model.LEDCommand) error {
	now := time.Now().UTC().Format(time.RFC3339)

	var deviceID int64
	var kind string
	if err := s.db.QueryRow(`SELECT id, kind FROM devices WHERE mac = ?`, mac).Scan(&deviceID, &kind); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if kind != "led_strip" {
		return nil
	}

	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO led_strips (device_id, power, brightness, color, pixel_pin, pixel_count, updated_at) VALUES (?, 0, 0, '#000000', 0, 255, ?)`,
		deviceID,
		now,
	); err != nil {
		return err
	}

	_, err := s.db.Exec(`
	UPDATE led_strips
	SET power = COALESCE(?, power),
	    brightness = COALESCE(?, brightness),
	    color = COALESCE(?, color),
	    pixel_pin = COALESCE(?, pixel_pin),
	    updated_at = ?
	WHERE device_id = ?
	`,
		nullableBoolInt(cmd.Power),
		nullableInt(cmd.Brightness),
		nullableString(cmd.Color),
		nullableInt(cmd.PixelPin),
		now,
		deviceID,
	)
	return err
}

func (s *Store) UpsertRoom(name string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO rooms (name) VALUES (?)`, strings.TrimSpace(name))
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableBoolInt(v *bool) any {
	if v == nil {
		return nil
	}
	return boolToInt(*v)
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return strings.TrimSpace(*v)
}

func suffix(mac string) string {
	mac = strings.ReplaceAll(strings.ToUpper(mac), ":", "")
	if len(mac) <= 6 {
		return mac
	}
	return mac[len(mac)-6:]
}
