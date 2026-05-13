package routing

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
)

func TestResolveModel(t *testing.T) {
	catalog := driver.Catalog{
		Drivers: []driver.Driver{
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
}

func TestResolveModelUsesDriverDefault(t *testing.T) {
	catalog := driver.Catalog{
		Drivers: []driver.Driver{
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
