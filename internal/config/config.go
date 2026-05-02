package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"noveltts/internal/model"
)

var (
	globalConfig *model.AppConfig
	configPath   string
	mu           sync.RWMutex
)

func defaultConfig() *model.AppConfig {
	return &model.AppConfig{
		Server: model.ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Providers: []model.ProviderConfig{},
		Defaults: model.DefaultsConfig{
			Provider: "",
			Speed:    1.0,
			Format:   "mp3",
		},
	}
}

func getConfigPath() string {
	if p := os.Getenv("NOVELTTS_CONFIG"); p != "" {
		return p
	}
	return filepath.Join("data", "config.json")
}

func resolveConfigPath() string {
	p := getConfigPath()
	if filepath.IsAbs(p) {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

func Load() (*model.AppConfig, error) {
	mu.Lock()
	defer mu.Unlock()

	configPath = resolveConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			if err := saveLocked(cfg); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			globalConfig = cfg
			log.Printf("[config] created default config at %s", configPath)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &model.AppConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	globalConfig = cfg
	log.Printf("[config] loaded from %s (%d providers)", configPath, len(cfg.Providers))
	return cfg, nil
}

func Get() *model.AppConfig {
	mu.RLock()
	defer mu.RUnlock()
	return globalConfig
}

func Save(cfg *model.AppConfig) error {
	mu.Lock()
	defer mu.Unlock()
	globalConfig = cfg
	return saveLocked(cfg)
}

func saveLocked(cfg *model.AppConfig) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func Reload() (*model.AppConfig, error) {
	return Load()
}
