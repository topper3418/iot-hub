// Directory: backend/internal/app/
// Modified: 2026-04-08
// Description: HTTP API routing, MQTT wiring, and orchestration layer. Bridges db, mqtt, and Pico provisioning.
// Uses: backend/internal/db/db.go, backend/internal/model/types.go, backend/internal/mqtt/client.go, backend/internal/mqtt/payload.go, backend/internal/provision/monitor.go, backend/internal/provision/provision.go
// Used by: backend/cmd/server/main.go

package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"iot-hub/backend/internal/db"
	"iot-hub/backend/internal/model"
	"iot-hub/backend/internal/mqtt"
	"iot-hub/backend/internal/provision"
)

type Application struct {
	store       *db.Store
	mqttClient  *mqtt.Client
	httpServer  *http.Server
	pico        *provision.Monitor
	provisionMu sync.Mutex
	provState   PicoProvisionState
	seenMu      sync.Mutex
	seenMACs    map[string]bool
	ackMu       sync.Mutex
	ackWaiters  map[string]chan string
}

type PicoProvisionState struct {
	Running    bool   `json:"running"`
	Stage      string `json:"stage"`
	Detail     string `json:"detail"`
	Error      string `json:"error,omitempty"`
	UpdatedAt  string `json:"updatedAt"`
	LastResult string `json:"lastResult"`
	Attempt    int    `json:"attempt"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

func New() (*Application, error) {
	dbPath := envOrDefault("IOTHUB_DB_PATH", "./data/iothub.db")
	broker := envOrDefault("IOTHUB_MQTT_BROKER", "tcp://127.0.0.1:1883")
	clientID := envOrDefault("IOTHUB_MQTT_CLIENT_ID", "iot-hub-backend")
	addr := envOrDefault("IOTHUB_HTTP_ADDR", ":8080")
	staticDir := envOrDefault("IOTHUB_STATIC_DIR", "../frontend/dist")

	store, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	app := &Application{store: store, seenMACs: map[string]bool{}, ackWaiters: map[string]chan string{}}
	log.Printf("startup config: db=%s mqtt=%s http=%s static=%s", dbPath, broker, addr, staticDir)

	mc, err := mqtt.NewClient(broker, clientID, func(mac string, payload []byte) {
		status := mqtt.ParseStatus(payload)
		if err := app.store.UpsertLEDDevice(mac, status); err != nil {
			log.Printf("upsert LED device failed for %s: %v", mac, err)
		}
		app.noteFirstSeen(mac)
		app.notifyProvisionAck(mac, payload)
	})
	if err != nil {
		store.Close()
		return nil, err
	}
	app.mqttClient = mc

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/devices", app.handleListDevices)
	mux.HandleFunc("PUT /api/devices/", app.handleUpdateDevice)
	mux.HandleFunc("POST /api/devices/", app.handleCommandDevice)
	mux.HandleFunc("GET /api/rooms", app.handleListRooms)
	mux.HandleFunc("POST /api/rooms", app.handleCreateRoom)
	mux.HandleFunc("GET /api/pico/status", app.handlePicoStatus)
	mux.HandleFunc("GET /api/pico/provision/state", app.handlePicoProvisionState)
	mux.HandleFunc("POST /api/pico/provision", app.handlePicoProvision)
	mux.HandleFunc("POST /api/pico/provision/reset", app.handlePicoProvisionReset)

	app.pico = provision.NewMonitor()
	app.provState = PicoProvisionState{Stage: "idle", Detail: "Waiting for configure request", UpdatedAt: time.Now().UTC().Format(time.RFC3339), LastResult: "none"}

	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fileServer := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			candidate := filepath.Join(staticDir, r.URL.Path)
			if _, err := os.Stat(candidate); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
	}

	app.httpServer = &http.Server{
		Addr:    addr,
		Handler: withJSONHeaders(mux),
	}

	return app, nil
}

func (a *Application) Start() error {
	a.pico.Start()
	go func() {
		log.Printf("http server listening on %s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()
	return nil
}

func (a *Application) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.httpServer.Shutdown(ctx)
	a.pico.Stop()
	a.mqttClient.Close()
	_ = a.store.Close()
}

func (a *Application) handleListDevices(w http.ResponseWriter, _ *http.Request) {
	devices, err := a.store.ListDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, devices)
}

func (a *Application) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/update") {
		http.NotFound(w, r)
		return
	}
	mac := pathPartBeforeSuffix(r.URL.Path, "/update")
	if mac == "" {
		http.Error(w, "missing mac", http.StatusBadRequest)
		return
	}

	var patch model.DevicePatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := a.store.UpdateDevice(mac, patch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("device metadata updated: mac=%s name_set=%t room_set=%t", mac, patch.Name != nil, patch.RoomID != nil)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (a *Application) handleCommandDevice(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/command") {
		http.NotFound(w, r)
		return
	}
	mac := pathPartBeforeSuffix(r.URL.Path, "/command")
	if mac == "" {
		http.Error(w, "missing mac", http.StatusBadRequest)
		return
	}

	var cmd model.LEDCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	payload, err := mqtt.BuildCommandPayload(cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.mqttClient.PublishCommand(mac, payload); err != nil {
		http.Error(w, fmt.Sprintf("mqtt publish failed: %v", err), http.StatusBadGateway)
		return
	}
	if err := a.store.ApplyLEDCommand(mac, cmd); err != nil {
		log.Printf("apply led command failed for %s: %v", mac, err)
	}
	log.Printf("command sent: mac=%s power=%t brightness=%t color=%t pixelPin=%t", mac, cmd.Power != nil, cmd.Brightness != nil, cmd.Color != nil, cmd.PixelPin != nil)
	writeJSON(w, map[string]string{"status": "sent"})
}

func (a *Application) handleListRooms(w http.ResponseWriter, _ *http.Request) {
	rooms, err := a.store.ListRooms()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, rooms)
}

func (a *Application) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "room name required", http.StatusBadRequest)
		return
	}
	if err := a.store.UpsertRoom(body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("room upserted: name=%s", strings.TrimSpace(body.Name))
	writeJSON(w, map[string]string{"status": "ok"})
}

func (a *Application) handlePicoStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.pico.GetStatus())
}

func (a *Application) handlePicoProvisionState(w http.ResponseWriter, _ *http.Request) {
	a.provisionMu.Lock()
	state := a.provState
	a.provisionMu.Unlock()
	writeJSON(w, state)
}

func (a *Application) handlePicoProvisionReset(w http.ResponseWriter, _ *http.Request) {
	a.provisionMu.Lock()
	if a.provState.Running {
		a.provisionMu.Unlock()
		http.Error(w, "cannot reset while provisioning is running", http.StatusConflict)
		return
	}
	a.provState.Stage = "idle"
	a.provState.Detail = "Waiting for configure request"
	a.provState.Error = ""
	a.provState.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.provState.FinishedAt = ""
	a.provisionMu.Unlock()
	writeJSON(w, map[string]string{"status": "reset"})
}

func (a *Application) handlePicoProvision(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PixelPin *int `json:"pixelPin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	pixelPin := 16
	if body.PixelPin != nil {
		pixelPin = *body.PixelPin
	}

	a.provisionMu.Lock()
	if a.provState.Running {
		a.provisionMu.Unlock()
		http.Error(w, "provisioning already in progress", http.StatusConflict)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	a.provState = PicoProvisionState{
		Running:    true,
		Stage:      "queued",
		Detail:     "Provisioning request accepted",
		UpdatedAt:  now,
		LastResult: a.provState.LastResult,
		Attempt:    a.provState.Attempt + 1,
		StartedAt:  now,
		FinishedAt: "",
	}
	a.provisionMu.Unlock()

	status := a.pico.GetStatus()
	log.Printf("provision request accepted: attempt=%d pixelPin=%d picoState=%s", a.currentAttempt(), pixelPin, status.State)
	brokerHost := brokerHostForPico(r)
	if brokerHost == "" {
		a.setProvisionError("unable to determine MQTT broker host")
		http.Error(w, "unable to determine MQTT broker host", http.StatusInternalServerError)
		return
	}

	mainPyPath := envOrDefault("IOTHUB_PICO_MAIN", "../pico/main.py")
	uf2Path := envOrDefault("IOTHUB_PICO_UF2", "/opt/iot-hub/pico-rp2.uf2")
	ssid := os.Getenv("IOTHUB_WIFI_SSID")
	password := os.Getenv("IOTHUB_WIFI_PASSWORD")
	provisionTag := newProvisionTag()
	a.registerProvisionAckWaiter(provisionTag)

	go a.runProvision(provision.Options{
		Status:       status,
		PixelPin:     pixelPin,
		MainPyPath:   mainPyPath,
		UF2Path:      uf2Path,
		WiFiSSID:     ssid,
		WiFiPassword: password,
		BrokerHost:   brokerHost,
		BrokerPort:   1883,
		ProvisionTag: provisionTag,
		Progress:     a.setProvisionProgress,
	}, provisionTag)

	writeJSON(w, map[string]string{"status": "started"})
}

func (a *Application) runProvision(opts provision.Options, provisionTag string) {
	err := provision.Provision(opts)
	if err != nil {
		a.unregisterProvisionAckWaiter(provisionTag)
		log.Printf("provision failed: attempt=%d err=%v", a.currentAttempt(), err)
		a.setProvisionError(err.Error())
		return
	}

	a.setProvisionProgress("verify", "Waiting for MQTT status confirmation from provisioned Pico")
	mac, err := a.waitForProvisionAck(provisionTag, 25*time.Second)
	if err != nil {
		a.unregisterProvisionAckWaiter(provisionTag)
		a.setProvisionError("files uploaded, but no MQTT confirmation received within 25s")
		log.Printf("provision upload succeeded but no mqtt confirmation: attempt=%d", a.currentAttempt())
		return
	}
	a.unregisterProvisionAckWaiter(provisionTag)
	log.Printf("provision confirmed by MQTT: attempt=%d mac=%s", a.currentAttempt(), mac)

	a.provisionMu.Lock()
	attempt := a.provState.Attempt
	startedAt := a.provState.StartedAt
	a.provState = PicoProvisionState{
		Running:    false,
		Stage:      "done",
		Detail:     fmt.Sprintf("Provisioning completed and device confirmed: %s", mac),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		LastResult: "success",
		Attempt:    attempt,
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}
	a.provisionMu.Unlock()
}

func (a *Application) setProvisionProgress(stage, detail string) {
	a.provisionMu.Lock()
	attempt := a.provState.Attempt
	startedAt := a.provState.StartedAt
	lastResult := a.provState.LastResult
	a.provState = PicoProvisionState{
		Running:    true,
		Stage:      stage,
		Detail:     detail,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		LastResult: lastResult,
		Attempt:    attempt,
		StartedAt:  startedAt,
		FinishedAt: "",
	}
	a.provisionMu.Unlock()
	log.Printf("provision progress: attempt=%d stage=%s detail=%s", attempt, stage, detail)
}

func (a *Application) setProvisionError(detail string) {
	a.provisionMu.Lock()
	attempt := a.provState.Attempt
	startedAt := a.provState.StartedAt
	a.provState = PicoProvisionState{
		Running:    false,
		Stage:      "error",
		Detail:     "Provisioning failed",
		Error:      detail,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		LastResult: "error",
		Attempt:    attempt,
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}
	a.provisionMu.Unlock()
	log.Printf("provision error: attempt=%d detail=%s", attempt, detail)
}

func (a *Application) noteFirstSeen(mac string) {
	a.seenMu.Lock()
	if a.seenMACs[mac] {
		a.seenMu.Unlock()
		return
	}
	a.seenMACs[mac] = true
	a.seenMu.Unlock()
	log.Printf("first status received from device mac=%s", mac)
}

func (a *Application) registerProvisionAckWaiter(tag string) {
	a.ackMu.Lock()
	a.ackWaiters[tag] = make(chan string, 1)
	a.ackMu.Unlock()
}

func (a *Application) unregisterProvisionAckWaiter(tag string) {
	a.ackMu.Lock()
	delete(a.ackWaiters, tag)
	a.ackMu.Unlock()
}

func (a *Application) notifyProvisionAck(mac string, payload []byte) {
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return
	}
	tag, _ := body["provisionTag"].(string)
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}
	a.ackMu.Lock()
	ch, ok := a.ackWaiters[tag]
	a.ackMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- mac:
	default:
	}
}

func (a *Application) waitForProvisionAck(tag string, timeout time.Duration) (string, error) {
	a.ackMu.Lock()
	ch, ok := a.ackWaiters[tag]
	a.ackMu.Unlock()
	if !ok {
		return "", fmt.Errorf("ack waiter missing")
	}
	select {
	case mac := <-ch:
		return mac, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout")
	}
}

func (a *Application) currentAttempt() int {
	a.provisionMu.Lock()
	defer a.provisionMu.Unlock()
	return a.provState.Attempt
}

func newProvisionTag() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("prov-%d", time.Now().UnixNano())
	}
	return "prov-" + hex.EncodeToString(b)
}

func brokerHostForPico(r *http.Request) string {
	if v := strings.TrimSpace(os.Getenv("IOTHUB_PICO_MQTT_BROKER")); v != "" {
		return v
	}

	host := strings.TrimSpace(r.Host)
	if host != "" {
		parsedHost := host
		if h, _, err := net.SplitHostPort(host); err == nil {
			parsedHost = h
		}
		parsedHost = strings.Trim(parsedHost, "[]")
		if parsedHost != "" && parsedHost != "localhost" && parsedHost != "127.0.0.1" {
			return parsedHost
		}
	}

	if ip := provision.LocalIPv4(); ip != "" {
		return ip
	}
	return ""
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func pathPartBeforeSuffix(path, suffix string) string {
	trimmed := strings.TrimSuffix(path, suffix)
	trimmed = strings.TrimPrefix(trimmed, "/api/devices/")
	trimmed = strings.Trim(trimmed, "/")
	return strings.ToLower(trimmed)
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
