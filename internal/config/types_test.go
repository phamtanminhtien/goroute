package config

import (
	"encoding/json"
	"testing"
)

func TestServerConfigJSONShapeMatchesConfigFile(t *testing.T) {
	bytes, err := json.Marshal(ServerConfig{
		Listen:    ":2232",
		AuthToken: "change-me",
		WebUIDir:  "web/dist",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"listen":":2232","auth_token":"change-me","web_ui_dir":"web/dist"}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

func TestConfigMarshalUsesFalseForDisabledLLMLogging(t *testing.T) {
	bytes, err := json.Marshal(Config{
		Server: ServerConfig{
			Listen:    ":2232",
			AuthToken: "change-me",
			WebUIDir:  "web/dist",
		},
		LLMLogging: LLMLoggingConfig{},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"server":{"listen":":2232","auth_token":"change-me","web_ui_dir":"web/dist"},"llmLogging":false,"rtk":false,"openAICompatibleAuth":{"enabled":false}}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

func TestConfigMarshalUsesObjectForEnabledLLMLogging(t *testing.T) {
	bytes, err := json.Marshal(Config{
		Server: ServerConfig{
			Listen:    ":2232",
			AuthToken: "change-me",
			WebUIDir:  "web/dist",
		},
		LLMLogging: LLMLoggingConfig{
			Flow:       true,
			ThirdParty: true,
		},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"server":{"listen":":2232","auth_token":"change-me","web_ui_dir":"web/dist"},"llmLogging":{"flow":true,"thirdParty":true},"rtk":false,"openAICompatibleAuth":{"enabled":false}}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}

func TestConfigMarshalUsesObjectForEnabledRTK(t *testing.T) {
	bytes, err := json.Marshal(Config{
		Server: ServerConfig{
			Listen:    ":2232",
			AuthToken: "change-me",
			WebUIDir:  "web/dist",
		},
		LLMLogging: LLMLoggingConfig{
			Flow:       true,
			ThirdParty: true,
		},
		RTK: RTKConfig{
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	const want = `{"server":{"listen":":2232","auth_token":"change-me","web_ui_dir":"web/dist"},"llmLogging":{"flow":true,"thirdParty":true},"rtk":{"enabled":true},"openAICompatibleAuth":{"enabled":false}}`
	if string(bytes) != want {
		t.Fatalf("expected JSON shape %s, got %s", want, string(bytes))
	}
}
