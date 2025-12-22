package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"balancer/internal/config"
	"balancer/internal/handlers"
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
			log.Printf("Loaded config from %s", path)
			break
		}
	}

	if err != nil {
		cfg, err = config.LoadFromEnv()
	}

	if err != nil {
		log.Fatal("Failed to load a config from any path or env")
	}

	fmt.Printf("We loaded the config from main: %v\n", cfg)

	// could just pass in cfg and parse it on the other side, making an interface easier
	handler := handlers.NewBalanceHandler(cfg.BackendName, cfg.BackendPort, cfg.LoadbalancerPort, cfg.LoadbalancerMethod)

	mux := http.NewServeMux()
	handler.Register(mux)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.LoadbalancerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("Starting server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())

}
