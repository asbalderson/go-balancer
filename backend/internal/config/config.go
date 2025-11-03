package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port int
	Name string
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load the config file at: %v. %v", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	config := &Config{}

	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %v", err)
	}
	log.Printf("Loaded the json: $%+v", config)
	return config, nil
}
