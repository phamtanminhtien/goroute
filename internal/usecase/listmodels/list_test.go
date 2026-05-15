package listmodels

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func TestExecuteIncludesProviderMetadataAndDefaultFallback(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{
				ID:           "cx",
				Name:         "Codex",
				AuthType:     provider.AuthTypeOAuth,
				DefaultModel: "cx/gpt-5.4",
				Models: []provider.Model{{
					ID:          "cx/gpt-5.4",
					Name:        "GPT-5.4",
					Description: "Primary Codex model",
				}},
			},
			{
				ID:           "fallback",
				Name:         "Fallback",
				AuthType:     provider.AuthTypeAPIKey,
				DefaultModel: "fallback/gpt-4.1",
			},
		},
	}

	models := Execute(catalog)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Metadata["provider_id"] != "cx" {
		t.Fatalf("expected provider metadata, got %#v", models[0].Metadata)
	}
	if models[0].Metadata["is_default"] != "true" {
		t.Fatalf("expected default flag for first model, got %#v", models[0].Metadata)
	}
	if models[1].ID != "fallback/gpt-4.1" {
		t.Fatalf("expected fallback model from default model, got %#v", models[1])
	}
	if models[1].Metadata["provider_name"] != "Fallback" {
		t.Fatalf("expected provider metadata, got %#v", models[1].Metadata)
	}
}
