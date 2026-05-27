package anthropiccompatible

import (
	"strings"

	upstreamanthropic "github.com/phamtanminhtien/goroute/internal/adapter/upstream/anthropic"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Registration(descriptor provider.Provider) providerregistry.Registration {
	descriptor.AdapterType = provider.AdapterTypeAnthropicCompatible
	return providerregistry.Registration{
		Descriptor: descriptor,
		BuildConnection: func(connectionConfig connection.Record, runtimeSettings config.ProviderRuntimeSettings) (chatcompletion.ProtocolConnections, error) {
			client := upstreamanthropic.NewClientWithBaseURLAndRuntimeSettings(connectionConfig, descriptor.BaseURL, runtimeSettings)
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
