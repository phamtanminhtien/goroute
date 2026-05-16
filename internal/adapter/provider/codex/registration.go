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
			AuthType:     provider.AuthTypeOAuth,
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
		GenerateOAuthURL: func(connectionConfig config.ConnectionConfig) (string, error) {
			return GenerateOAuthURL(connectionConfig)
		},
		StartOAuth: func(connectionConfig config.ConnectionConfig) (providerregistry.OAuthSession, error) {
			pending, err := StartOAuth(connectionConfig)
			if err != nil {
				return providerregistry.OAuthSession{}, err
			}

			return providerregistry.OAuthSession{
				AuthorizationURL: pending.AuthorizationURL,
				Pending: map[string]string{
					"code_verifier": pending.CodeVerifier,
					"state":         pending.State,
					"redirect_uri":  pending.RedirectURI,
				},
			}, nil
		},
		CompleteOAuth: func(connectionConfig config.ConnectionConfig, pending map[string]string, callbackURL string) (providerregistry.OAuthResult, error) {
			completed, err := CompleteCodexOAuthFromCallbackURL(PendingCodexOAuth{
				CodeVerifier: pending["code_verifier"],
				State:        pending["state"],
				RedirectURI:  pending["redirect_uri"],
			}, callbackURL)
			if err != nil {
				return providerregistry.OAuthResult{}, err
			}

			return providerregistry.OAuthResult{
				AccessToken:  completed.AccessToken,
				RefreshToken: completed.RefreshToken,
				TokenType:    completed.TokenType,
				ExpiresIn:    completed.ExpiresIn,
				Name:         completed.Email,
			}, nil
		},
		ValidateConnection: func(connectionConfig config.ConnectionConfig) []string {
			if strings.TrimSpace(connectionConfig.AccessToken) == "" && strings.TrimSpace(connectionConfig.APIKey) == "" {
				return []string{"missing access_token or api_key"}
			}

			return nil
		},
	}
}
