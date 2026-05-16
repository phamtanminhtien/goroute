package openai

import (
	"strings"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Registration() providerregistry.Registration {
	return providerregistry.Registration{
		Descriptor: provider.Provider{
			ID:           "openai",
			Name:         "OpenAI",
			AuthType:     provider.AuthTypeAPIKey,
			Category:     "api_key",
			DefaultModel: "openai/gpt-4.1",
			Models: []provider.Model{{
				ID:          "openai/gpt-4.1",
				Name:        "GPT-4.1",
				Description: "",
			}},
		},
		BuildConnection: func(connectionConfig connection.Record) (chatcompletion.Connection, error) {
			return NewClient(nil, connectionConfig), nil
		},
		ValidateConnection: func(connectionConfig connection.Record) []string {
			if strings.TrimSpace(connectionConfig.APIKey) == "" && strings.TrimSpace(connectionConfig.AccessToken) == "" {
				return []string{"missing api_key or access_token"}
			}

			return nil
		},
	}
}
