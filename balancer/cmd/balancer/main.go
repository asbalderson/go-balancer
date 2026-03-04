package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"balancer/internal/config"
	"balancer/internal/discovery"
	"balancer/internal/handlers"
	"pkg/logging"
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
			break
		}
	}

	if err != nil {
		cfg, err = config.LoadFromEnv()
	}

	if err != nil {
		logging.Error("Failed to load a config from any path or env, %v", err)
		os.Exit(1)
	}

	logging.Debug("We loaded the config from main: %v\n", cfg)

	stopCh := make(chan struct{})

	factory, err := discovery.GetBackendFactory("")
	if err != nil {
		logging.Error("Failed to create backend factory: %v", err)
		os.Exit(1)
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
		logging.Info("Starting server on %s", server.Addr)
		server.ListenAndServe()
	}()

	go func() {
		<-stopCh
		logging.Warning("Stopping server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	close(stopCh)
}
