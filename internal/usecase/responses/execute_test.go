package responses

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func TestExecuteStreamReturnsComboAliasModel(t *testing.T) {
	registry := chatcompletion.NewConnectionRegistry(map[string][]chatcompletion.ConnectionEntry{
		"cx": {{
			ID:         "cx-1",
			Name:       "cx-1",
			ProviderID: "cx",
			ProtocolConnections: chatcompletion.ProtocolConnections{
				Responses: responsesStreamConnection{
					body: `data: {"type":"response.created","response":{"id":"resp_1","object":"response","created_at":1712345678,"status":"in_progress","model":"gpt-5.4","output":[]}}` + "\n\n" + "data: [DONE]\n\n",
				},
			},
		}},
	})

	output, err := ExecuteStream(context.Background(), comboResponsesTestCatalog(), comboResponsesTestCombos(), &registry, Input{
		Request: openaiwire.ResponsesRequest{
			Model:  "combo/fast",
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

type responsesStreamConnection struct {
	body string
}

func (c responsesStreamConnection) Responses(context.Context, openaiwire.ResponsesRequest, routing.Target) (openaiwire.ResponsesResponse, error) {
	return openaiwire.ResponsesResponse{}, nil
}

func (c responsesStreamConnection) ResponsesStream(context.Context, openaiwire.ResponsesRequest, routing.Target) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(c.body)), nil
}

func comboResponsesTestCatalog() provider.Catalog {
	return provider.Catalog{Providers: []provider.Provider{{
		ID:           "cx",
		Name:         "Codex",
		DefaultModel: "cx/gpt-5.4",
	}}}
}

func comboResponsesTestCombos() []modelcombo.Combo {
	return []modelcombo.Combo{{
		Alias:   "combo/fast",
		Enabled: true,
		Targets: []modelcombo.Target{{
			ComboAlias: "combo/fast",
			ProviderID: "cx",
			ModelID:    "cx/gpt-5.4",
			Enabled:    true,
			Priority:   0,
		}},
	}}
}
