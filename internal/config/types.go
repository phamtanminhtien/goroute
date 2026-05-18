package config

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Config struct {
	Server               ServerConfig               `json:"server"`
	LLMLogging           LLMLoggingConfig           `json:"llmLogging"`
	RTK                  RTKConfig                  `json:"rtk"`
	OpenAICompatibleAuth OpenAICompatibleAuthConfig `json:"openAICompatibleAuth"`
}

type ServerConfig struct {
	Listen    string `json:"listen"`
	AuthToken string `json:"auth_token"`
	WebUIDir  string `json:"web_ui_dir,omitempty"`
}

type LLMLoggingConfig struct {
	Flow       bool
	ThirdParty bool
	present    bool
}

type OpenAICompatibleAuthConfig struct {
	Enabled bool `json:"enabled"`
}

type llmLoggingJSON struct {
	Flow       bool `json:"flow,omitempty"`
	ThirdParty bool `json:"thirdParty,omitempty"`
}

func (c Config) MarshalJSON() ([]byte, error) {
	type rawConfig struct {
		Server               ServerConfig               `json:"server"`
		LLMLogging           LLMLoggingConfig           `json:"llmLogging"`
		RTK                  RTKConfig                  `json:"rtk"`
		OpenAICompatibleAuth OpenAICompatibleAuthConfig `json:"openAICompatibleAuth"`
	}

	return json.Marshal(rawConfig{
		Server:               c.Server,
		LLMLogging:           c.LLMLogging,
		RTK:                  c.RTK,
		OpenAICompatibleAuth: c.OpenAICompatibleAuth,
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type rawConfig struct {
		Server               ServerConfig               `json:"server"`
		OpenAICompatibleAuth OpenAICompatibleAuthConfig `json:"openAICompatibleAuth"`
	}

	var decoded rawConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*c = Config{
		Server:               decoded.Server,
		OpenAICompatibleAuth: decoded.OpenAICompatibleAuth,
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if payload, ok := raw["llmLogging"]; ok {
		if err := c.LLMLogging.UnmarshalJSON(payload); err != nil {
			return fmt.Errorf("decode config.llmLogging: %w", err)
		}
	}
	if payload, ok := raw["rtk"]; ok {
		if err := c.RTK.UnmarshalJSON(payload); err != nil {
			return fmt.Errorf("decode config.rtk: %w", err)
		}
	}

	return nil
}

func (c LLMLoggingConfig) MarshalJSON() ([]byte, error) {
	if !c.Flow && !c.ThirdParty {
		return []byte("false"), nil
	}

	return json.Marshal(llmLoggingJSON{
		Flow:       c.Flow,
		ThirdParty: c.ThirdParty,
	})
}

func (c *LLMLoggingConfig) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("value is required")
	}

	switch trimmed[0] {
	case 'f', 't':
		var disabled bool
		if err := json.Unmarshal(trimmed, &disabled); err != nil {
			return err
		}
		if disabled {
			return fmt.Errorf("boolean true is not supported")
		}

		*c = LLMLoggingConfig{
			Flow:       false,
			ThirdParty: false,
			present:    true,
		}
		return nil
	case '{':
		var decoded llmLoggingJSON
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return err
		}

		*c = LLMLoggingConfig{
			Flow:       decoded.Flow,
			ThirdParty: decoded.ThirdParty,
			present:    true,
		}
		return nil
	default:
		return fmt.Errorf("expected false or object")
	}
}

func (c LLMLoggingConfig) IsPresent() bool {
	return c.present
}

func NewLLMLoggingConfig(flow bool, thirdParty bool) LLMLoggingConfig {
	return LLMLoggingConfig{
		Flow:       flow,
		ThirdParty: thirdParty,
		present:    true,
	}
}
