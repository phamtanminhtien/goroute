package openaicompatible

import (
	"strings"

	upstreamopenai "github.com/phamtanminhtien/goroute/internal/adapter/upstream/openai"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Registration(descriptor provider.Provider) providerregistry.Registration {
	descriptor.AdapterType = provider.AdapterTypeOpenAICompatible
	return providerregistry.Registration{
		Descriptor: descriptor,
		BuildConnection: func(connectionConfig connection.Record) (chatcompletion.ProtocolConnections, error) {
			client := upstreamopenai.NewClientWithBaseURL(nil, connectionConfig, descriptor.BaseURL)
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
