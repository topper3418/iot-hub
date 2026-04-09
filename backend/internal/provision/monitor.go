// Directory: backend/internal/provision/
// Modified: 2026-04-08
// Description: Detects connected Pico states from Linux device nodes for BOOTSEL and serial modes.
// Uses: none
// Used by: backend/internal/app/server.go

package provision

import (
	"path/filepath"
	"sync"
	"time"
)

const (
	PicoStateNone        = "none"
	PicoStateBootsel     = "bootsel"
	PicoStateMicropython = "micropython"
)

type Status struct {
	State      string `json:"state"`
	Connected  bool   `json:"connected"`
	SerialPort string `json:"serialPort,omitempty"`
	UpdatedAt  string `json:"updatedAt"`
}

type Monitor struct {
	mu     sync.RWMutex
	status Status
	stop   chan struct{}
}

func NewMonitor() *Monitor {
	m := &Monitor{
		status: Status{
			State:     PicoStateNone,
			Connected: false,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		},
		stop: make(chan struct{}),
	}
	m.refresh()
	return m
}

func (m *Monitor) Start() {
	t := time.NewTicker(2 * time.Second)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-t.C:
				m.refresh()
			}
		}
	}()
}

func (m *Monitor) Stop() {
	select {
	case <-m.stop:
		return
	default:
		close(m.stop)
	}
}

func (m *Monitor) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Monitor) refresh() {
	next := detect()
	m.mu.Lock()
	m.status = next
	m.mu.Unlock()
}

func detect() Status {
	now := time.Now().UTC().Format(time.RFC3339)

	serial, _ := filepath.Glob("/dev/ttyACM*")
	if len(serial) > 0 {
		return Status{
			State:      PicoStateMicropython,
			Connected:  true,
			SerialPort: serial[0],
			UpdatedAt:  now,
		}
	}

	bootsel, _ := filepath.Glob("/dev/disk/by-label/RPI-RP2")
	if len(bootsel) > 0 {
		return Status{
			State:     PicoStateBootsel,
			Connected: true,
			UpdatedAt: now,
		}
	}

	return Status{
		State:     PicoStateNone,
		Connected: false,
		UpdatedAt: now,
	}
}
