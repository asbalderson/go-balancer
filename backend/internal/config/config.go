package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port        int    `json:"port"`
	ServiceName string `json:"name"`
}

func LoadFromEnv() (*Config, error) {
	servicename, ok := os.LookupEnv("SERVICE_NAME")
	if !ok {
		return nil, fmt.Errorf("Could not load service name from environment `SERVICE_NAME`")
	}
	portStr, ok := os.LookupEnv("SERVICE_PORT")
	if !ok {
		return nil, fmt.Errorf("Could not load port from environment `SERVICE_PORT`")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert port to an int, %s", portStr)
	}

	cfg := Config{
		Port:        port,
		ServiceName: servicename,
	}

	return &cfg, nil
}

func LoadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the config file %s: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	config := &Config{}

	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %w", err)
	}
	log.Printf("Loaded the json: %+v", config)
	return config, nil
}
