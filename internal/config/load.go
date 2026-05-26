package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultAuthToken = "change-me"

func Load() (Config, error) {
	resolvedPath, err := ResolvePath()
	if err != nil {
		return Config{}, err
	}

	return LoadPath(resolvedPath)
}

func LoadPath(path string) (Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	cfg = ApplyDefaults(cfg)

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func LoadOrCreatePath(path string) (Config, bool, error) {
	cfg, err := LoadPath(path)
	if err == nil {
		return cfg, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, false, err
	}

	cfg = NewDefaultConfig()
	if err := SavePath(path, cfg); err != nil {
		return Config{}, false, err
	}

	return cfg, true, nil
}

func NewDefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Listen:    DefaultListenAddr,
			AuthToken: DefaultAuthToken,
			WebUIDir:  DefaultWebUIDir,
		},
		LLMLogging:           NewLLMLoggingConfig(false, false),
		RTK:                  NewRTKConfig(true),
		OpenAICompatibleAuth: OpenAICompatibleAuthConfig{Enabled: false},
		ProviderRuntime:      nil,
	}
}

func SavePath(path string, cfg Config) error {
	cfg = ApplyDefaults(cfg)
	if err := Validate(cfg); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir for %q: %w", path, err)
	}

	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config %q: %w", path, err)
	}
	bytes = append(bytes, '\n')

	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}

	return nil
}

func ResolvePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".goroute", "config.json"), nil
}
