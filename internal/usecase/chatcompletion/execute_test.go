package chatcompletion

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestExecuteStreamReturnsComboAliasModel(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {newConnectionEntry("cx", 1, recordingConnection{
			streamBody: `data: {"id":"chatcmpl_1","object":"chat.completion.chunk","created":1712345678,"model":"gpt-5.4","choices":[{"index":0,"delta":{"content":"hi"}}]}` + "\n\n" + "data: [DONE]\n\n",
		}, nil)},
	})

	output, err := ExecuteStream(context.Background(), comboTestCatalog(), comboTestCombos(), &registry, Input{
		Request: openaiwire.ChatCompletionsRequest{
			Model: "combo/fast",
			Messages: []openaiwire.ChatMessage{{
				Role:    openaiwire.ChatRoleUser,
				Content: openaiwire.TextContent("hello"),
			}},
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteStream returned error: %v", err)
	}
	defer output.Body.Close()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if !strings.Contains(string(body), `"model":"combo/fast"`) {
		t.Fatalf("expected stream model to be combo alias, got %s", string(body))
	}
	if strings.Contains(string(body), `"model":"gpt-5.4"`) {
		t.Fatalf("expected provider model to be hidden from stream, got %s", string(body))
	}
}

func comboTestCatalog() provider.Catalog {
	return provider.Catalog{Providers: []provider.Provider{{
		ID:           "cx",
		Name:         "Codex",
		DefaultModel: "cx/gpt-5.4",
	}}}
}

func comboTestCombos() []modelcombo.Combo {
	return []modelcombo.Combo{{
		Alias: "combo/fast",
		Targets: []modelcombo.Target{{
			ComboAlias: "combo/fast",
			ProviderID: "cx",
			ModelID:    "cx/gpt-5.4",
			Enabled:    true,
			Priority:   0,
		}},
	}}
}
