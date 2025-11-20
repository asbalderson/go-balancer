package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"backend/internal/config"
	"backend/internal/handlers"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Printf("its broked, %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("We loaded the config from main: %v\n", cfg)

	handler := handlers.NewServiceHandler(cfg.ServiceName, time.Now().Format(time.RFC3339))

	mux := http.NewServeMux()
	handler.Register(mux)
	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("Starting server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())

}
