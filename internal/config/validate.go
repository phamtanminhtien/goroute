package config

import (
	"fmt"
	"strings"
)

const DefaultListenAddr = ":2232"

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Server.AuthToken) == "" {
		return fmt.Errorf("config.server.auth_token is required")
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("config.providers must contain at least one provider")
	}

	for i, provider := range cfg.Providers {
		if provider.ID == "" {
			return fmt.Errorf("config.providers[%d].id is required", i)
		}
		if provider.Type == "" {
			return fmt.Errorf("config.providers[%d].type is required", i)
		}
		if provider.Name == "" {
			return fmt.Errorf("config.providers[%d].name is required", i)
		}
	}

	return nil
}

func ApplyDefaults(cfg Config) Config {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = DefaultListenAddr
	}

	return cfg
}
