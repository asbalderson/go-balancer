package config

import (
	"testing"
)

func TestLoadConfig_Success(T *testing.T) {
	cfg, err := Load("testdata/valid_config.json")
	if err != nil {
		T.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Port != 8080 {
		T.Errorf("Expected port 8080, got: %d", cfg.Port)
	}

	if cfg.ServiceName != "test" {
		T.Errorf("Expected service name 'test', got '%s'", cfg.ServiceName)
	}

}

func TestLoadConfig_FileNotFound(T *testing.T) {
	_, err := Load("/no/file.json")
	if err == nil {
		T.Error("Exepected error for missing file, but we found the file?")
	}
}

func TestLoadConfig_InvalidJson(T *testing.T) {
	_, err := Load("testdata/invalid_config_port_string.json")
	if err == nil {
		T.Error("Epected JSON type error, but we loaded anyway?")
	}
}
