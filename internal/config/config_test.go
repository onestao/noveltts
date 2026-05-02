package config

import (
	"os"
	"path/filepath"
	"testing"

	"noveltts/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Defaults.Speed != 1.0 {
		t.Errorf("expected speed 1.0, got %f", cfg.Defaults.Speed)
	}
	if cfg.Defaults.Format != "mp3" {
		t.Errorf("expected format mp3, got %s", cfg.Defaults.Format)
	}
}

func TestLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "config.json")
	os.Setenv("NOVELTTS_CONFIG", configPath)
	defer os.Unsetenv("NOVELTTS_CONFIG")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	cfg.Defaults.Provider = "test"
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg2, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if cfg2.Defaults.Provider != "test" {
		t.Errorf("expected provider 'test', got '%s'", cfg2.Defaults.Provider)
	}
}

func TestGet(t *testing.T) {
	globalConfig = &model.AppConfig{
		Server: model.ServerConfig{Port: 9999},
	}
	cfg := Get()
	if cfg.Server.Port != 9999 {
		t.Errorf("expected 9999, got %d", cfg.Server.Port)
	}
}
