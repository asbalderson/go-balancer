package config

import (
	"os"
	"testing"
)

func TestLoadConfigFile_Success(t *testing.T) {
	cfg, err := LoadFromFile("testdata/valid_config.json")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got: %d", cfg.Port)
	}

	if cfg.ServiceName != "test" {
		t.Errorf("Expected service name 'test', got '%s'", cfg.ServiceName)
	}

}

func TestLoadFileConfig_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("/no/file.json")
	if err == nil {
		t.Error("Expected error for missing file, but we found the file?")
	}
}

func TestLoadFileConfig_InvalidJson(t *testing.T) {
	_, err := LoadFromFile("testdata/invalid_config_port_string.json")
	if err == nil {
		t.Error("Expected JSON type error, but we loaded anyway?")
	}
}

func TestLoadEnvConfig(t *testing.T) {
	os.Setenv("SERVICE_NAME", "myservice")
	os.Setenv("SERVICE_PORT", "1234")
	t.Cleanup(func() {
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_PORT")
	})

	cfg, err := LoadFromEnv()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Port != 1234 {
		t.Errorf("Expected port 1234, got: %d", cfg.Port)
	}

	if cfg.ServiceName != "myservice" {
		t.Errorf("Expected service name 'myservice', got '%s'", cfg.ServiceName)
	}
}

func TestLoadEnvConfig_NoName(t *testing.T) {
	os.Setenv("SERVICE_PORT", "1234")
	t.Cleanup(func() {
		os.Unsetenv("SERVICE_PORT")
	})

	_, err := LoadFromEnv()

	if err == nil {
		t.Fatalf("Loaded config when no SERVICE_NAME was not set")
	}
}

func TestLoadEnvConfig_NoPort(t *testing.T) {
	os.Setenv("SERVICE_NAME", "myservice")
	t.Cleanup(func() {
		os.Unsetenv("SERVICE_NAME")
	})

	_, err := LoadFromEnv()

	if err == nil {
		t.Fatalf("Loaded config when no SERVICE_PORT was not set")
	}
}

func TestLoadEnvConfig_BadPort(t *testing.T) {
	os.Setenv("SERVICE_NAME", "myservice")
	os.Setenv("SERVICE_PORT", "1234a")
	t.Cleanup(func() {
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_PORT")
	})

	_, err := LoadFromEnv()

	if err == nil {
		t.Fatalf("Loaded config when no SERVICE_PORT was not an int")
	}
}
