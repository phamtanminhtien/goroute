package app

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
)

func TestBuildProviderRegistryLogsDiagnostics(t *testing.T) {
	var logs bytes.Buffer

	_, err := buildProviderRegistryWithLogger([]config.ProviderConfig{
		{ID: "codex-1", Type: "codex", Name: "codex-user"},
		{ID: "openai-1", Type: "openai", Name: "openai-user", APIKey: "token"},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("buildProviderRegistryWithLogger returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `provider_id="codex-1"`) || !strings.Contains(output, `status="misconfigured"`) {
		t.Fatalf("expected misconfigured codex provider diagnostic, got %s", output)
	}
	if !strings.Contains(output, `provider_id="openai-1"`) || !strings.Contains(output, `status="ready"`) {
		t.Fatalf("expected ready openai provider diagnostic, got %s", output)
	}
}
