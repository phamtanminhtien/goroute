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

	for i, connection := range cfg.Connections {
		if connection.ID == "" {
			return fmt.Errorf("config.connections[%d].id is required", i)
		}
		if connection.ProviderID == "" {
			return fmt.Errorf("config.connections[%d].provider_id is required", i)
		}
		if connection.Name == "" {
			return fmt.Errorf("config.connections[%d].name is required", i)
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
