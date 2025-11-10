package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port        int    `json:"port"`
	ServiceName string `json:"name"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open the config file: %s. %w", path, err)
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
