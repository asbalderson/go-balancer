package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Success(T *testing.T) {
	content := `{"port": 8080, "name": "test"}`
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		T.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		T.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Port != 8080 {
		T.Errorf("Expected port 8080, got: %d", cfg.Port)
	}

	if cfg.Name != "test" {
		T.Errorf("Expected name 'test', got '%s'", cfg.Name)
	}

}

func TestLoadConfig_FileNotFound(T *testing.T) {
	_, err := LoadConfig("/no/file.json")
	if err == nil {
		T.Error("Exepected error for missing file, but we found the file?")
	}
}

func TestLoadConfig_InvalidJson(T *testing.T) {
	content := `{"port": "8080", "name": "test"}`
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		T.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		T.Error("Epected JSON type error, but we loaded anyway?")
	}
}
