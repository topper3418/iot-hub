// Directory: backend/internal/provision/
// Modified: 2026-04-08
// Description: Pico provisioning workflow for flashing MicroPython UF2 and uploading runtime files via mpremote.
// Uses: none
// Used by: backend/internal/app/server.go

package provision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const flashHelperPath = "/usr/local/bin/iot-hub-flash-uf2"

type Options struct {
	Status       Status
	PixelPin     int
	MainPyPath   string
	UF2Path      string
	WiFiSSID     string
	WiFiPassword string
	BrokerHost   string
	BrokerPort   int
	Progress     func(stage, detail string)
}

func Provision(opts Options) error {
	emit(opts.Progress, "validating", "Validating provisioning inputs")
	if opts.PixelPin < 0 || opts.PixelPin > 28 {
		return fmt.Errorf("pixelPin must be between 0 and 28")
	}
	if strings.TrimSpace(opts.MainPyPath) == "" {
		return errors.New("main.py path is required")
	}
	if _, err := os.Stat(opts.MainPyPath); err != nil {
		return fmt.Errorf("main.py not found at %s", opts.MainPyPath)
	}
	if strings.TrimSpace(opts.BrokerHost) == "" {
		return errors.New("broker host is required")
	}

	ssid := strings.TrimSpace(opts.WiFiSSID)
	if ssid == "" {
		emit(opts.Progress, "network", "Detecting WiFi SSID from Raspberry Pi")
		autoSSID, err := detectSSID()
		if err != nil {
			return err
		}
		ssid = autoSSID
	}

	password := strings.TrimSpace(opts.WiFiPassword)

	if err := ensureMPRemote(); err != nil {
		return err
	}

	serialPort := strings.TrimSpace(opts.Status.SerialPort)
	switch opts.Status.State {
	case PicoStateBootsel:
		emit(opts.Progress, "flashing", "BOOTSEL detected, flashing MicroPython UF2")
		if strings.TrimSpace(opts.UF2Path) == "" {
			return errors.New("UF2 path is required in BOOTSEL mode")
		}
		if _, err := os.Stat(opts.UF2Path); err != nil {
			return fmt.Errorf("UF2 file not found at %s", opts.UF2Path)
		}
		mount, err := findBootselMount()
		if err == nil {
			if err := copyFile(filepath.Join(mount, filepath.Base(opts.UF2Path)), opts.UF2Path); err != nil {
				return fmt.Errorf("failed to flash UF2: %w", err)
			}
		} else {
			emit(opts.Progress, "flashing", "BOOTSEL detected but not mounted, using privileged flash helper")
			if helperErr := flashUF2ViaHelper(opts.UF2Path); helperErr != nil {
				return fmt.Errorf("%v; helper fallback failed: %w", err, helperErr)
			}
		}
		emit(opts.Progress, "waiting_serial", "Waiting for Pico serial interface")
		serialPort, err = waitForSerial(25 * time.Second)
		if err != nil {
			return fmt.Errorf("flash completed, but serial port did not appear: %w", err)
		}
	case PicoStateMicropython:
		emit(opts.Progress, "serial", "MicroPython mode detected")
		if serialPort == "" {
			emit(opts.Progress, "waiting_serial", "Waiting for Pico serial interface")
			p, err := waitForSerial(8 * time.Second)
			if err != nil {
				return err
			}
			serialPort = p
		}
	default:
		return errors.New("no Pico detected; plug in while holding BOOTSEL or connect a MicroPython Pico")
	}

	emit(opts.Progress, "config", "Generating Pico runtime config")
	cfgPath, cleanup, err := writeDeviceConfig(ssid, password, opts.BrokerHost, opts.BrokerPort, opts.PixelPin)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emit(opts.Progress, "upload_main", "Uploading main.py")
	if err := run(ctx, "mpremote", "connect", serialPort, "fs", "cp", opts.MainPyPath, ":main.py"); err != nil {
		return fmt.Errorf("failed to upload main.py: %w", err)
	}
	emit(opts.Progress, "upload_config", "Uploading device_config.py")
	if err := run(ctx, "mpremote", "connect", serialPort, "fs", "cp", cfgPath, ":device_config.py"); err != nil {
		return fmt.Errorf("failed to upload device_config.py: %w", err)
	}
	emit(opts.Progress, "reset", "Resetting Pico")
	if err := run(ctx, "mpremote", "connect", serialPort, "reset"); err != nil {
		return fmt.Errorf("uploaded files, but reset failed: %w", err)
	}
	emit(opts.Progress, "done", "Provisioning completed")

	return nil
}

func emit(progress func(stage, detail string), stage, detail string) {
	if progress != nil {
		progress(stage, detail)
	}
}

func LocalIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ipv4 := ipNet.IP.To4()
			if ipv4 != nil {
				return ipv4.String()
			}
		}
	}
	return ""
}

func detectSSID() (string, error) {
	out, err := exec.Command("iwgetid", "-r").CombinedOutput()
	if err != nil {
		return "", errors.New("unable to detect WiFi SSID automatically; set IOTHUB_WIFI_SSID")
	}
	ssid := strings.TrimSpace(string(out))
	if ssid == "" {
		return "", errors.New("detected empty WiFi SSID; set IOTHUB_WIFI_SSID")
	}
	return ssid, nil
}

func ensureMPRemote() error {
	if _, err := exec.LookPath("mpremote"); err != nil {
		return errors.New("mpremote is not installed; install it with: pip3 install mpremote")
	}
	return nil
}

func findBootselMount() (string, error) {
	patterns := []string{
		"/media/*/RPI-RP2",
		"/run/media/*/RPI-RP2",
		"/mnt/RPI-RP2",
		"/Volumes/RPI-RP2",
	}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, m := range matches {
			info, err := os.Stat(m)
			if err == nil && info.IsDir() {
				return m, nil
			}
		}
	}
	return "", errors.New("BOOTSEL device found but not mounted (RPI-RP2 mountpoint not found)")
}

func flashUF2ViaHelper(uf2Path string) error {
	if _, err := os.Stat(flashHelperPath); err != nil {
		return fmt.Errorf("flash helper not found at %s", flashHelperPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, "sudo", "-n", flashHelperPath, uf2Path); err != nil {
		return fmt.Errorf("sudo helper execution failed: %w", err)
	}
	return nil
}

func waitForSerial(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ports, _ := filepath.Glob("/dev/ttyACM*")
		if len(ports) > 0 {
			return ports[0], nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", errors.New("timed out waiting for /dev/ttyACM*")
}

func writeDeviceConfig(ssid, password, broker string, port, pixelPin int) (string, func(), error) {
	content := fmt.Sprintf("# Generated by backend provisioning\nWIFI_SSID = %q\nWIFI_PASSWORD = %q\nMQTT_BROKER = %q\nMQTT_PORT = %d\nPIXEL_PIN = %d\nPIXEL_COUNT = 255\n", ssid, password, broker, port, pixelPin)

	f, err := os.CreateTemp("", "device_config_*.py")
	if err != nil {
		return "", nil, err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", nil, err
	}
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, text)
	}
	return nil
}

func copyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
