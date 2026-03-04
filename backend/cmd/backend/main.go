package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"backend/internal/config"
	"backend/internal/handlers"
	"pkg/logging"
)

func main() {
	paths := []string{
		"/etc/backend/config.json",
		"config.json",
	}
	var cfg *config.Config
	var err error

	for _, path := range paths {
		cfg, err = config.LoadFromFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		cfg, err = config.LoadFromEnv()
	}

	if err != nil {
		logging.Error("Failed to load a config from any path or env: %v", err)
		os.Exit(1)
	}
	handler := handlers.NewServiceHandler(cfg.ServiceName)

	mux := http.NewServeMux()
	handler.Register(mux)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	logging.Info("Starting server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())

}
