package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/logging"
)

func TestBuildConnectionRegistryLogsDiagnostics(t *testing.T) {
	var logs bytes.Buffer
	logger := logging.NewWithWriter("prod", &logs)
	providers, err := buildProviderRegistry()
	if err != nil {
		t.Fatalf("buildProviderRegistry returned error: %v", err)
	}

	_, err = buildConnectionRegistryWithLogger([]connection.Record{
		{ID: "codex-1", ProviderID: "cx", Name: "codex-user", Enabled: true},
		{ID: "openai-1", ProviderID: "openai", Name: "openai-user", APIKey: "token", Enabled: true},
	}, providers, &logger)
	if err != nil {
		t.Fatalf("buildConnectionRegistryWithLogger returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `"connection_id":"codex-1"`) || !strings.Contains(output, `"status":"misconfigured"`) {
		t.Fatalf("expected misconfigured codex connection diagnostic, got %s", output)
	}
	if !strings.Contains(output, `"connection_id":"openai-1"`) || !strings.Contains(output, `"status":"ready"`) {
		t.Fatalf("expected ready openai connection diagnostic, got %s", output)
	}
}

func TestBuildConnectionEntriesSkipsDisabledConnections(t *testing.T) {
	providers, err := buildProviderRegistry()
	if err != nil {
		t.Fatalf("buildProviderRegistry returned error: %v", err)
	}

	entries, err := buildConnectionEntries([]connection.Record{{
		ID:         "openai-disabled",
		ProviderID: "openai",
		Name:       "disabled",
		APIKey:     "token",
		Enabled:    false,
	}}, providers, nil, nil)
	if err != nil {
		t.Fatalf("buildConnectionEntries returned error: %v", err)
	}
	if len(entries["openai"]) != 0 {
		t.Fatalf("expected disabled connection to be skipped, got %#v", entries)
	}
}

func TestBuildProviderRegistryWithCustomAnthropicCompatibleProvider(t *testing.T) {
	providers, err := buildProviderRegistryWithCustom([]provider.Record{{
		ID:           "customanthropic",
		Name:         "Custom Anthropic",
		AuthType:     provider.AuthTypeAPIKey,
		Category:     "custom",
		AdapterType:  provider.AdapterTypeAnthropicCompatible,
		BaseURL:      "https://anthropic.example.com",
		DefaultModel: "customanthropic/claude-sonnet-4-5",
	}})
	if err != nil {
		t.Fatalf("buildProviderRegistryWithCustom returned error: %v", err)
	}

	connections, err := providers.BuildConnection(connection.Record{
		ID:         "customanthropic",
		ProviderID: "customanthropic",
		Name:       "Custom Anthropic",
		APIKey:     "token",
	}, config.DefaultProviderRuntimeSettings())
	if err != nil {
		t.Fatalf("BuildConnection returned error: %v", err)
	}
	if connections.ChatCompletions == nil || connections.Responses == nil || connections.Anthropic == nil {
		t.Fatalf("expected Anthropic-compatible protocols, got %#v", connections)
	}
}

func TestNewStartsWithEmptySQLiteDatabase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".goroute", "config.json")
	if err := config.SavePath(configPath, config.Config{
		Server: config.ServerConfig{
			Listen:    ":2232",
			AuthToken: "secret",
		},
	}); err != nil {
		t.Fatalf("SavePath returned error: %v", err)
	}

	logger := logging.NewWithWriter("prod", &bytes.Buffer{})
	app, err := New(logger)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer app.repo.Close()

	if app.server == nil {
		t.Fatal("expected server to be initialized")
	}
	if app.repo == nil {
		t.Fatal("expected repository to be initialized")
	}
}

func TestNewCreatesMissingUserConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var logs bytes.Buffer
	logger := logging.NewWithWriter("prod", &logs)
	app, err := New(logger)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer app.repo.Close()

	configPath := filepath.Join(home, ".goroute", "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
	if !strings.Contains(logs.String(), `"message":"user_config_created"`) {
		t.Fatalf("expected config creation log, got %s", logs.String())
	}
}
