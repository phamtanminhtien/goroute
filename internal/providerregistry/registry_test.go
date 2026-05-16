package providerregistry

import (
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func TestCatalogPreservesRegistrationOrder(t *testing.T) {
	registry, err := New(
		Registration{
			Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
			BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
				return nil, nil
			},
		},
		Registration{
			Descriptor: provider.Provider{ID: "openai", Name: "OpenAI"},
			BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
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
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.BuildConnection(connection.Record{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), `unsupported provider "openai"`) {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestValidateConnectionUsesProviderValidator(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
		ValidateConnection: func(connection.Record) []string {
			return []string{"missing access_token or api_key"}
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	problems := registry.ValidateConnection(connection.Record{ProviderID: "cx"})
	if len(problems) != 1 || problems[0] != "missing access_token or api_key" {
		t.Fatalf("unexpected problems: %#v", problems)
	}
}

func TestGenerateOAuthURLUsesProviderGenerator(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
		GenerateOAuthURL: func(connection.Record) (string, error) {
			return "https://auth.openai.com/oauth/authorize?provider=cx", nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	url, err := registry.GenerateOAuthURL(connection.Record{ProviderID: "cx"})
	if err != nil {
		t.Fatalf("GenerateOAuthURL returned error: %v", err)
	}
	if url != "https://auth.openai.com/oauth/authorize?provider=cx" {
		t.Fatalf("unexpected oauth url: %q", url)
	}
}

func TestGetAccessTokenUsesProviderResolver(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
		GetAccessToken: func(record connection.Record) (string, error) {
			return "access-for-" + record.ID, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	token, err := registry.GetAccessToken(connection.Record{ID: "codex-1", ProviderID: "cx"})
	if err != nil {
		t.Fatalf("GetAccessToken returned error: %v", err)
	}
	if token != "access-for-codex-1" {
		t.Fatalf("unexpected access token: %q", token)
	}
}

func TestGetAccessTokenRejectsUnsupportedProvider(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.GetAccessToken(connection.Record{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), `unsupported provider "openai"`) {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestGetAccessTokenRejectsProviderWithoutResolver(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.GetAccessToken(connection.Record{ProviderID: "cx"})
	if err == nil || !strings.Contains(err.Error(), "does not support access token resolution") {
		t.Fatalf("expected unsupported access token resolver error, got %v", err)
	}
}

func TestGenerateOAuthURLRejectsUnsupportedProvider(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "openai", Name: "OpenAI"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = registry.GenerateOAuthURL(connection.Record{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), "does not support oauth authorization url generation") {
		t.Fatalf("expected unsupported oauth generator error, got %v", err)
	}
}

func TestStartOAuthUsesProviderStarter(t *testing.T) {
	registry, err := New(Registration{
		Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
		StartOAuth: func(connection.Record) (OAuthSession, error) {
			return OAuthSession{
				AuthorizationURL: "https://auth.openai.com/oauth/authorize?provider=cx",
				Pending:          map[string]string{"state": "test-state"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	session, err := registry.StartOAuth(connection.Record{ProviderID: "cx"})
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
		BuildConnection: func(connection.Record) (chatcompletion.Connection, error) {
			return nil, nil
		},
		CompleteOAuth: func(connection.Record, map[string]string, string) (OAuthResult, error) {
			return OAuthResult{
				AccessToken:          "access-123",
				RefreshToken:         "refresh-456",
				TokenType:            "Bearer",
				ExpiresIn:            3600,
				AccessTokenExpiresAt: 1710003600,
				Name:                 "user@example.com",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	result, err := registry.CompleteOAuth(
		connection.Record{ProviderID: "cx"},
		map[string]string{"state": "test-state"},
		"http://localhost:1455/auth/callback?code=abc&state=test-state",
	)
	if err != nil {
		t.Fatalf("CompleteOAuth returned error: %v", err)
	}
	if result.AccessToken != "access-123" || result.RefreshToken != "refresh-456" || result.TokenType != "Bearer" || result.ExpiresIn != 3600 || result.AccessTokenExpiresAt != 1710003600 || result.Name != "user@example.com" {
		t.Fatalf("unexpected oauth result: %#v", result)
	}
}
