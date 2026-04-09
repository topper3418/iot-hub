// Directory: backend/internal/app/
// Modified: 2026-04-08
// Description: HTTP API routing, MQTT wiring, and orchestration layer. Bridges the db and mqtt packages.
// Uses: backend/internal/db/db.go, backend/internal/model/types.go, backend/internal/mqtt/client.go, backend/internal/mqtt/payload.go, backend/internal/provision/monitor.go
// Used by: backend/cmd/server/main.go

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"iot-hub/backend/internal/db"
	"iot-hub/backend/internal/model"
	"iot-hub/backend/internal/mqtt"
	"iot-hub/backend/internal/provision"
)

type Application struct {
	store      *db.Store
	mqttClient *mqtt.Client
	httpServer *http.Server
	pico       *provision.Monitor
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

	app := &Application{store: store}

	mc, err := mqtt.NewClient(broker, clientID, func(mac string, payload []byte) {
		status := mqtt.ParseStatus(payload)
		if err := app.store.UpsertLEDDevice(mac, status); err != nil {
			log.Printf("upsert LED device failed for %s: %v", mac, err)
		}
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

	app.pico = provision.NewMonitor()

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
	writeJSON(w, map[string]string{"status": "ok"})
}

func (a *Application) handlePicoStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, a.pico.GetStatus())
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
