package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Load() (Config, error) {
	resolvedPath, err := resolvePath()
	if err != nil {
		return Config{}, err
	}

	bytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", resolvedPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", resolvedPath, err)
	}

	cfg = ApplyDefaults(cfg)

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func resolvePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".goroute", "config.json"), nil
}
