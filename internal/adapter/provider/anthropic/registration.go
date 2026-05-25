package anthropic

import (
	"strings"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Registration() providerregistry.Registration {
	return providerregistry.Registration{
		Descriptor: provider.Provider{
			ID:           "anthropic",
			Name:         "Anthropic",
			AuthType:     provider.AuthTypeAPIKey,
			Category:     "api_key",
			DefaultModel: "anthropic/claude-sonnet-4-5",
			Models: []provider.Model{
				{ID: "anthropic/claude-sonnet-4-5", Name: "Claude Sonnet 4.5", Description: ""},
				{ID: "anthropic/claude-opus-4-1", Name: "Claude Opus 4.1", Description: ""},
				{ID: "anthropic/claude-haiku-4-5", Name: "Claude Haiku 4.5", Description: ""},
			},
		},
		BuildConnection: func(connectionConfig connection.Record, runtimeSettings config.ProviderRuntimeSettings) (chatcompletion.ProtocolConnections, error) {
			client := NewClientWithRuntimeSettings(connectionConfig, runtimeSettings)
			return chatcompletion.ProtocolConnections{
				ChatCompletions: client,
				Responses:       client,
				Anthropic:       client,
			}, nil
		},
		ValidateConnection: func(connectionConfig connection.Record) []string {
			if strings.TrimSpace(connectionConfig.APIKey) == "" {
				return []string{"missing api_key"}
			}
			return nil
		},
	}
}
