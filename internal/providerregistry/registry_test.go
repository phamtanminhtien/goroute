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

func TestGenerateOAuthURLUsesProviderGenerator(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
		GenerateOAuthURL: func(config.ConnectionConfig) (string, error) {
			return "https://auth.openai.com/oauth/authorize?provider=cx", nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	url, err := registry.GenerateOAuthURL(config.ConnectionConfig{ProviderID: "cx"})
	if err != nil {
		t.Fatalf("GenerateOAuthURL returned error: %v", err)
	}
	if url != "https://auth.openai.com/oauth/authorize?provider=cx" {
		t.Fatalf("unexpected oauth url: %q", url)
	}
}

func TestGenerateOAuthURLRejectsUnsupportedProvider(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "openai", Name: "OpenAI"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.GenerateOAuthURL(config.ConnectionConfig{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), "does not support oauth authorization url generation") {
		t.Fatalf("expected unsupported oauth generator error, got %v", err)
	}
}

func TestStartOAuthUsesProviderStarter(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
		StartOAuth: func(config.ConnectionConfig) (OAuthSession, error) {
			return OAuthSession{
				AuthorizationURL: "https://auth.openai.com/oauth/authorize?provider=cx",
				Pending:          map[string]string{"state": "test-state"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	session, err := registry.StartOAuth(config.ConnectionConfig{ProviderID: "cx"})
	if err != nil {
		t.Fatalf("StartOAuth returned error: %v", err)
	}
	if session.AuthorizationURL != "https://auth.openai.com/oauth/authorize?provider=cx" {
		t.Fatalf("unexpected oauth url: %q", session.AuthorizationURL)
	}
	if session.Pending["state"] != "test-state" {
		t.Fatalf("unexpected pending state: %#v", session.Pending)
	}
}

func TestCompleteOAuthUsesProviderCompleter(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
			return nil, nil
		},
		CompleteOAuth: func(config.ConnectionConfig, map[string]string, string) (OAuthResult, error) {
			return OAuthResult{
				AccessToken:  "access-123",
				RefreshToken: "refresh-456",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				Name:         "user@example.com",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := registry.CompleteOAuth(
		config.ConnectionConfig{ProviderID: "cx"},
		map[string]string{"state": "test-state"},
		"http://localhost:1455/auth/callback?code=abc&state=test-state",
	)
	if err != nil {
		t.Fatalf("CompleteOAuth returned error: %v", err)
	}
	if result.AccessToken != "access-123" || result.RefreshToken != "refresh-456" || result.TokenType != "Bearer" || result.ExpiresIn != 3600 || result.Name != "user@example.com" {
		t.Fatalf("unexpected oauth result: %#v", result)
	}
}
