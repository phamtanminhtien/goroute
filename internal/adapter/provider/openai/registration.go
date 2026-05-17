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
			Models: []provider.Model{
				{
					ID:                       "openai/gpt-5.4",
					Name:                     "GPT-5.4",
					Description:              "",
					InputPricePerMillionUSD:  2.5,
					OutputPricePerMillionUSD: 15,
				},
				{
					ID:                       "openai/gpt-5.4-mini",
					Name:                     "GPT-5.4 Mini",
					Description:              "",
					InputPricePerMillionUSD:  0.75,
					OutputPricePerMillionUSD: 4.5,
				},
				{
					ID:                       "openai/gpt-5.4-nano",
					Name:                     "GPT-5.4 Nano",
					Description:              "",
					InputPricePerMillionUSD:  0.20,
					OutputPricePerMillionUSD: 1.25,
				},
				{
					ID:                       "openai/gpt-4.1",
					Name:                     "GPT-4.1",
					Description:              "",
					InputPricePerMillionUSD:  2,
					OutputPricePerMillionUSD: 8,
				},
				{
					ID:                       "openai/gpt-4.1-mini",
					Name:                     "GPT-4.1 Mini",
					Description:              "",
					InputPricePerMillionUSD:  0.4,
					OutputPricePerMillionUSD: 1.6,
				},
				{
					ID:                       "openai/gpt-4.1-nano",
					Name:                     "GPT-4.1 Nano",
					Description:              "",
					InputPricePerMillionUSD:  0.1,
					OutputPricePerMillionUSD: 0.4,
				},
				{
					ID:                       "openai/o4-mini",
					Name:                     "o4-mini",
					Description:              "",
					InputPricePerMillionUSD:  1.1,
					OutputPricePerMillionUSD: 4.4,
				},
			},
		},
		BuildConnection: func(connectionConfig connection.Record) (chatcompletion.ProtocolConnections, error) {
			client := NewClient(nil, connectionConfig)
			return chatcompletion.ProtocolConnections{
				ChatCompletions: client,
				Responses:       client,
			}, nil
		},
		ValidateConnection: func(connectionConfig connection.Record) []string {
			if strings.TrimSpace(connectionConfig.APIKey) == "" && strings.TrimSpace(connectionConfig.AccessToken) == "" {
				return []string{"missing api_key or access_token"}
			}

			return nil
		},
	}
}
