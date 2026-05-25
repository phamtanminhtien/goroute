package config

import (
	"fmt"
	"strings"
)

const DefaultListenAddr = ":2232"
const DefaultWebUIDir = "web/dist"

const (
	DefaultDialTimeoutMs           = 10000
	DefaultTLSHandshakeTimeoutMs   = 10000
	DefaultResponseHeaderTimeoutMs = 30000
	DefaultTimeoutRetryCount       = 3
	DefaultRetryableCooldownMs     = 60000
)

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Server.AuthToken) == "" {
		return fmt.Errorf("config.server.auth_token is required")
	}
	for providerID, runtimeSettings := range cfg.ProviderRuntime {
		if strings.TrimSpace(providerID) == "" || strings.Contains(providerID, "/") {
			return fmt.Errorf("config.providerRuntimeSettings provider id is required and must not contain /")
		}
		if err := ValidateProviderRuntimeSettings(runtimeSettings); err != nil {
			return fmt.Errorf("config.providerRuntimeSettings.%s: %w", providerID, err)
		}
	}

	return nil
}

func ApplyDefaults(cfg Config) Config {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = DefaultListenAddr
	}
	if strings.TrimSpace(cfg.Server.WebUIDir) == "" {
		cfg.Server.WebUIDir = DefaultWebUIDir
	}
	if !cfg.LLMLogging.IsPresent() {
		cfg.LLMLogging = LLMLoggingConfig{
			Flow:       true,
			ThirdParty: true,
			present:    true,
		}
	}
	if !cfg.RTK.IsPresent() {
		cfg.RTK = RTKConfig{
			Enabled: true,
			present: true,
		}
	}

	return cfg
}

func DefaultProviderRuntimeSettings() ProviderRuntimeSettings {
	return ProviderRuntimeSettings{
		DialTimeoutMs:           DefaultDialTimeoutMs,
		TLSHandshakeTimeoutMs:   DefaultTLSHandshakeTimeoutMs,
		ResponseHeaderTimeoutMs: DefaultResponseHeaderTimeoutMs,
		TimeoutRetryCount:       DefaultTimeoutRetryCount,
		RetryableCooldownMs:     DefaultRetryableCooldownMs,
	}
}

func EffectiveProviderRuntimeSettings(cfg Config, providerID string) ProviderRuntimeSettings {
	if cfg.ProviderRuntime == nil {
		return DefaultProviderRuntimeSettings()
	}
	if settings, ok := cfg.ProviderRuntime[providerID]; ok {
		return settings
	}
	return DefaultProviderRuntimeSettings()
}

func ValidateProviderRuntimeSettings(settings ProviderRuntimeSettings) error {
	if settings.DialTimeoutMs <= 0 {
		return fmt.Errorf("dialTimeoutMs must be positive")
	}
	if settings.TLSHandshakeTimeoutMs <= 0 {
		return fmt.Errorf("tlsHandshakeTimeoutMs must be positive")
	}
	if settings.ResponseHeaderTimeoutMs <= 0 {
		return fmt.Errorf("responseHeaderTimeoutMs must be positive")
	}
	if settings.TimeoutRetryCount < 0 || settings.TimeoutRetryCount > 10 {
		return fmt.Errorf("timeoutRetryCount must be between 0 and 10")
	}
	if settings.RetryableCooldownMs <= 0 {
		return fmt.Errorf("retryableCooldownMs must be positive")
	}

	return nil
}
