package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"balancer/internal/config"
	"balancer/internal/discovery"
	"balancer/internal/handlers"
)

func main() {
	paths := []string{
		"/etc/balancer/config.json",
		"config.json",
	}
	// consider a kubeconf config value for running locally.
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

	stopCh := make(chan struct{})

	factory, err := discovery.GetBackendFactory("")
	if err != nil {
		log.Fatal("Failed to create the backend factory")
	}
	backends := discovery.GetBackends(factory, cfg.BackendName)
	factory.Start(stopCh)
	// consider using cache.WaitForCacheSync(stopCh, endpointInformer.HasSynced) so you can capture bool out for errors
	factory.WaitForCacheSync(stopCh)

	handler := handlers.NewBalanceHandler(cfg.BackendName, cfg.BackendPort, cfg.LoadbalancerPort, cfg.LoadbalancerMethod, backends)
	mux := http.NewServeMux()
	handler.Register(mux)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.LoadbalancerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on %s", server.Addr)
		server.ListenAndServe()
	}()

	go func() {
		<-stopCh
		log.Println("Stopping server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	close(stopCh)
}
