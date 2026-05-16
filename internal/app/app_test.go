package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
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
		{ID: "codex-1", ProviderID: "cx", Name: "codex-user"},
		{ID: "openai-1", ProviderID: "openai", Name: "openai-user", APIKey: "token"},
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
