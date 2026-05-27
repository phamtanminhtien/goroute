package anthropiccompatible

import (
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func TestRegistrationBuildsAnthropicProtocolConnections(t *testing.T) {
	registration := Registration(provider.Provider{
		ID:           "customanthropic",
		Name:         "Custom Anthropic",
		AuthType:     provider.AuthTypeAPIKey,
		Category:     "custom",
		BaseURL:      "https://anthropic.example.com",
		DefaultModel: "customanthropic/claude-sonnet-4-5",
	})

	if registration.Descriptor.AdapterType != provider.AdapterTypeAnthropicCompatible {
		t.Fatalf("unexpected adapter type %q", registration.Descriptor.AdapterType)
	}

	connections, err := registration.BuildConnection(connection.Record{
		ID:         "customanthropic",
		ProviderID: "customanthropic",
		Name:       "Custom Anthropic",
		APIKey:     "token",
	}, config.DefaultProviderRuntimeSettings())
	if err != nil {
		t.Fatalf("BuildConnection returned error: %v", err)
	}
	if connections.ChatCompletions == nil || connections.Responses == nil || connections.Anthropic == nil {
		t.Fatalf("expected all Anthropic-compatible protocols, got %#v", connections)
	}
}

func TestRegistrationRequiresAPIKey(t *testing.T) {
	registration := Registration(provider.Provider{ID: "customanthropic"})

	problems := registration.ValidateConnection(connection.Record{ID: "customanthropic", ProviderID: "customanthropic"})
	if len(problems) != 1 || problems[0] != "missing api_key" {
		t.Fatalf("unexpected validation problems %#v", problems)
	}
}
