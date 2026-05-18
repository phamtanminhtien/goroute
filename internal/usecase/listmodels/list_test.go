package listmodels

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
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

func TestExecuteWithCombosIncludesComboMetadata(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{{
			ID:           "cx",
			Name:         "Codex",
			AuthType:     provider.AuthTypeOAuth,
			DefaultModel: "cx/gpt-5.4",
			Models:       []provider.Model{{ID: "cx/gpt-5.4", Name: "GPT-5.4"}},
		}},
	}
	combos := []modelcombo.Combo{{
		Alias:       "fast",
		Name:        "Fast",
		Description: "Fast combo",
		Targets: []modelcombo.Target{{
			ProviderID: "cx",
			ModelID:    "cx/gpt-5.4",
			Enabled:    true,
		}},
	}}

	models := ExecuteWithCombos(catalog, combos)
	if len(models) != 2 {
		t.Fatalf("expected provider model plus combo, got %d", len(models))
	}
	combo := models[1]
	if combo.ID != "fast" || combo.OwnedBy != "combo" {
		t.Fatalf("expected combo model, got %#v", combo)
	}
	if combo.Metadata["is_combo"] != "true" || combo.Metadata["target_count"] != "1" {
		t.Fatalf("expected combo metadata, got %#v", combo.Metadata)
	}
}
