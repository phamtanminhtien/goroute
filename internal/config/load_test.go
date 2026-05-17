package config

import (
	"path/filepath"
	"testing"
)

func TestLoadPathDefaultsLoggingAndRTKWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"}}`)

	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}

	if !cfg.LLMLogging.Flow || !cfg.LLMLogging.ThirdParty {
		t.Fatalf("expected llm logging defaults to be enabled, got %#v", cfg.LLMLogging)
	}
	if !cfg.RTK.Enabled {
		t.Fatalf("expected rtk default to be enabled, got %#v", cfg.RTK)
	}
}

func TestLoadPathParsesFalseLLMLogging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"},"llmLogging":false}`)

	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}

	if cfg.LLMLogging.Flow || cfg.LLMLogging.ThirdParty {
		t.Fatalf("expected llm logging to be disabled, got %#v", cfg.LLMLogging)
	}
}

func TestLoadPathParsesEmptyObjectLLMLoggingAsDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"},"llmLogging":{}}`)

	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}

	if cfg.LLMLogging.Flow || cfg.LLMLogging.ThirdParty {
		t.Fatalf("expected llm logging to be disabled, got %#v", cfg.LLMLogging)
	}
}

func TestLoadPathParsesPartialLLMLogging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"},"llmLogging":{"thirdParty":true}}`)

	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}

	if cfg.LLMLogging.Flow || !cfg.LLMLogging.ThirdParty {
		t.Fatalf("expected only third-party logging enabled, got %#v", cfg.LLMLogging)
	}
}

func TestLoadPathRejectsInvalidLLMLoggingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"},"llmLogging":[true]}`)

	_, err := LoadPath(path)
	if err == nil {
		t.Fatal("expected invalid llmLogging error")
	}
}

func TestSavePathCanonicalizesDisabledLLMLoggingToFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SavePath(path, Config{
		Server: ServerConfig{
			Listen:    ":2232",
			AuthToken: "secret",
			WebUIDir:  "web/dist",
		},
		LLMLogging: NewLLMLoggingConfig(false, false),
	}); err != nil {
		t.Fatalf("SavePath returned error: %v", err)
	}

	assertConfigFile(t, path, "{\n  \"server\": {\n    \"listen\": \":2232\",\n    \"auth_token\": \"secret\",\n    \"web_ui_dir\": \"web/dist\"\n  },\n  \"llmLogging\": false,\n  \"rtk\": {\n    \"enabled\": true\n  }\n}\n")
}

func TestLoadPathParsesRTKObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfigFile(t, path, `{"server":{"listen":":2232","auth_token":"secret","web_ui_dir":"web/dist"},"rtk":{"enabled":true}}`)

	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}

	if !cfg.RTK.Enabled {
		t.Fatalf("expected rtk to be enabled, got %#v", cfg.RTK)
	}
}
