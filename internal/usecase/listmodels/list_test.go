package listmodels

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
)

func TestExecuteIncludesDriverMetadataAndDefaultFallback(t *testing.T) {
	catalog := driver.Catalog{
		Drivers: []driver.Driver{
			{
				ID:           "cx",
				Name:         "Codex",
				Provider:     "codex",
				AuthType:     "oauth",
				DefaultModel: "cx/gpt-5.4",
				Models: []driver.Model{{
					ID:          "cx/gpt-5.4",
					Name:        "GPT-5.4",
					Description: "Primary Codex model",
				}},
			},
			{
				ID:           "fallback",
				Name:         "Fallback",
				Provider:     "openai",
				AuthType:     "api_key",
				DefaultModel: "fallback/gpt-4.1",
			},
		},
	}

	models := Execute(catalog)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Metadata["driver_id"] != "cx" {
		t.Fatalf("expected driver metadata, got %#v", models[0].Metadata)
	}
	if models[0].Metadata["is_default"] != "true" {
		t.Fatalf("expected default flag for first model, got %#v", models[0].Metadata)
	}
	if models[1].ID != "fallback/gpt-4.1" {
		t.Fatalf("expected fallback model from default model, got %#v", models[1])
	}
	if models[1].Metadata["provider_type"] != "openai" {
		t.Fatalf("expected provider metadata, got %#v", models[1].Metadata)
	}
}
