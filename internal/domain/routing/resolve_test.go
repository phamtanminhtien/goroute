package routing

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func TestResolveModel(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
		},
	}

	target, err := ResolveModel(catalog, "cx/gpt-5.4")
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if target.Prefix != "cx" {
		t.Fatalf("expected prefix cx, got %q", target.Prefix)
	}
	if target.RequestedModel != "gpt-5.4" {
		t.Fatalf("expected requested model gpt-5.4, got %q", target.RequestedModel)
	}
	if target.ProviderID != "cx" {
		t.Fatalf("expected provider id cx, got %q", target.ProviderID)
	}
}

func TestResolvePlanUsesExactComboAliasBeforeProviderPrefix(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
			{ID: "combo", Name: "Combo Provider", DefaultModel: "combo/default"},
		},
	}
	combos := []modelcombo.Combo{{
		Alias: "combo/fast",
		Targets: []modelcombo.Target{{
			ProviderID: "cx",
			ModelID:    "cx/gpt-5.4",
			Enabled:    true,
		}},
	}}

	plan, err := ResolvePlan(catalog, combos, "combo/fast")
	if err != nil {
		t.Fatalf("ResolvePlan returned error: %v", err)
	}

	if plan.ComboAlias != "combo/fast" {
		t.Fatalf("expected combo alias, got %#v", plan)
	}
	if len(plan.Targets) != 1 || plan.Targets[0].ProviderID != "cx" {
		t.Fatalf("expected combo target to resolve to cx, got %#v", plan.Targets)
	}
	if plan.ResponseModel != "combo/fast" {
		t.Fatalf("expected response model to stay alias, got %q", plan.ResponseModel)
	}
}

func TestResolvePlanFallsBackToProviderPrefixWhenNoComboMatches(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
		},
	}

	plan, err := ResolvePlan(catalog, nil, "cx/gpt-5.4")
	if err != nil {
		t.Fatalf("ResolvePlan returned error: %v", err)
	}

	if plan.ComboAlias != "" {
		t.Fatalf("expected provider route, got combo %#v", plan)
	}
	if plan.ResponseModel != "cx/gpt-5.4" {
		t.Fatalf("expected provider response model, got %q", plan.ResponseModel)
	}
}

func TestResolvePlanRejectsEmptyComboTargets(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{{ID: "cx", Name: "Codex"}},
	}
	combos := []modelcombo.Combo{{Alias: "fast"}}

	_, err := ResolvePlan(catalog, combos, "fast")
	if err == nil {
		t.Fatal("expected empty combo target error")
	}
}

func TestResolvePlanSkipsDisabledComboTargets(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
			{ID: "openai", Name: "OpenAI", DefaultModel: "openai/gpt-4.1"},
		},
	}
	combos := []modelcombo.Combo{{
		Alias: "fast",
		Targets: []modelcombo.Target{
			{ProviderID: "cx", ModelID: "cx/gpt-5.4", Enabled: false},
			{ProviderID: "openai", ModelID: "openai/gpt-4.1", ConnectionID: "openai-2", Enabled: true},
		},
	}}

	plan, err := ResolvePlan(catalog, combos, "fast")
	if err != nil {
		t.Fatalf("ResolvePlan returned error: %v", err)
	}

	if len(plan.Targets) != 1 || plan.Targets[0].ProviderID != "openai" {
		t.Fatalf("expected only enabled target, got %#v", plan.Targets)
	}
	if plan.Targets[0].ConnectionID != "openai-2" {
		t.Fatalf("expected connection id to resolve, got %#v", plan.Targets[0])
	}
}

func TestResolveModelUsesProviderDefault(t *testing.T) {
	catalog := provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
		},
	}

	target, err := ResolveModel(catalog, "cx")
	if err != nil {
		t.Fatalf("ResolveModel returned error: %v", err)
	}

	if target.RequestedModel != "gpt-5.4" {
		t.Fatalf("expected default requested model gpt-5.4, got %q", target.RequestedModel)
	}
}
