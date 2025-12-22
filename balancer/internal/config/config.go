package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
)

const (
	StrategyRoundRobin string = "RoundRobin"
)

type Config struct {
	BackendName        string `json:"backendname"`
	BackendPort        int    `json:"backendport"`
	LoadbalancerPort   int    `json:"loadbalancerport"`
	LoadbalancerMethod string `json:"loadbalancermethod"`
}

func (c *Config) validate() error {
	strategies := []string{StrategyRoundRobin}
	valid := false
	for _, s := range strategies {
		if c.LoadbalancerMethod == s {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid strategy, set one of %v", strategies)
	}
	return nil
}

func LoadFromEnv() (*Config, error) {
	backendName, ok := os.LookupEnv("BACKEND_NAME")
	if !ok {
		return nil, fmt.Errorf("could not load backend name from environment `BACKEND_NAME`")
	}
	backendPortStr, ok := os.LookupEnv("BACKEND_PORT")
	if !ok {
		return nil, fmt.Errorf("could not load backend port from environment `BACKEND_PORT`")
	}

	backendPort, err := strconv.Atoi(backendPortStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert backend port to an int, %s", backendPortStr)
	}

	loadbalancerPortStr, ok := os.LookupEnv("LOADBALANCER_PORT")
	if !ok {
		return nil, fmt.Errorf("could not load loadbalancer port from environment `LOADBALANCER_PORT`")
	}

	loadbalancerPort, err := strconv.Atoi(loadbalancerPortStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert loadbalancer port to an int, %s", loadbalancerPortStr)
	}

	loadbalancerMethod, ok := os.LookupEnv("LOADBALANCER_METHOD")
	if !ok {
		return nil, fmt.Errorf("could not load loadbalancer method from environment `LOADBALANCER_METHOD`")
	}

	cfg := Config{
		BackendName:        backendName,
		BackendPort:        backendPort,
		LoadbalancerPort:   loadbalancerPort,
		LoadbalancerMethod: loadbalancerMethod,
	}

	err = cfg.validate()
	if err != nil {
		return nil, err
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

	cfg := &Config{}

	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %w", err)
	}
	log.Printf("Loaded the json: %+v", cfg)

	err = cfg.validate()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
