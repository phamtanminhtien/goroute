package codex

import (
	"strings"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func Registration() providerregistry.Registration {
	return providerregistry.Registration{
		Descriptor: provider.Provider{
			ID:           "cx",
			Name:         "Codex",
			AuthType:     "oauth",
			Category:     "oauth",
			DefaultModel: "cx/gpt-5.4",
			Models: []provider.Model{{
				ID:          "cx/gpt-5.4",
				Name:        "GPT-5.4",
				Description: "",
			}},
		},
		BuildConnection: func(connectionConfig config.ConnectionConfig) (chatcompletion.Connection, error) {
			return NewClient(connectionConfig), nil
		},
		ValidateConnection: func(connectionConfig config.ConnectionConfig) []string {
			if strings.TrimSpace(connectionConfig.AccessToken) == "" && strings.TrimSpace(connectionConfig.APIKey) == "" {
				return []string{"missing access_token or api_key"}
			}

			return nil
		},
	}
}
