package codex

import (
	"context"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestClientUsesOpenAICompatibleProvider(t *testing.T) {
	client := NewClient(config.ProviderConfig{Type: "codex", Name: "codex-user"})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderType: "codex", RequestedModel: "gpt-5.4"})
	if err == nil {
		t.Fatal("expected credential error")
	}
	if !strings.Contains(err.Error(), "no usable credential") {
		t.Fatalf("expected OpenAI-compatible credential error, got %v", err)
	}
}
