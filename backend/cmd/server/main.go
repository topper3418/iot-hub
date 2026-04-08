package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"iot-hub/backend/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := a.Start(); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Stop()
}
