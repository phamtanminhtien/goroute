package providerregistry

import (
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func TestCatalogPreservesRegistrationOrder(t *testing.T) {
	registry, err := New(
		Registration{
			Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
			BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
				return nil, nil
			},
		},
		Registration{
			Descriptor: provider.Provider{ID: "openai", Name: "OpenAI"},
			BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
				return nil, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	catalog := registry.Catalog()
	if len(catalog.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(catalog.Providers))
	}
	if catalog.Providers[0].ID != "cx" || catalog.Providers[1].ID != "openai" {
		t.Fatalf("unexpected provider order: %#v", catalog.Providers)
	}
}

func TestBuildConnectionRejectsUnknownProvider(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.BuildConnection(config.ConnectionConfig{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), `unsupported provider "openai"`) {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestValidateConnectionUsesProviderValidator(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
		ValidateConnection: func(config.ConnectionConfig) []string {
			return []string{"missing access_token or api_key"}
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	problems := registry.ValidateConnection(config.ConnectionConfig{ProviderID: "cx"})
	if len(problems) != 1 || problems[0] != "missing access_token or api_key" {
		t.Fatalf("unexpected problems: %#v", problems)
	}
}
