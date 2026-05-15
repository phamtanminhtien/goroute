package app

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
)

func TestBuildConnectionRegistryLogsDiagnostics(t *testing.T) {
	var logs bytes.Buffer

	_, err := buildConnectionRegistryWithLogger([]config.ConnectionConfig{
		{ID: "codex-1", ProviderID: "cx", Name: "codex-user"},
		{ID: "openai-1", ProviderID: "openai", Name: "openai-user", APIKey: "token"},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("buildConnectionRegistryWithLogger returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `connection_id="codex-1"`) || !strings.Contains(output, `status="misconfigured"`) {
		t.Fatalf("expected misconfigured codex connection diagnostic, got %s", output)
	}
	if !strings.Contains(output, `connection_id="openai-1"`) || !strings.Contains(output, `status="ready"`) {
		t.Fatalf("expected ready openai connection diagnostic, got %s", output)
	}
}
