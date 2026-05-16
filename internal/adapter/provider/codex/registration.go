package codex

import (
	"context"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
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
		BuildConnection: func(connectionConfig connection.Record) (chatcompletion.Connection, error) {
			return NewClient(connectionConfig), nil
		},
		GetAccessToken: func(connectionConfig connection.Record) (string, error) {
			return GetAccessToken(connectionConfig)
		},
		GetUsage: func(ctx context.Context, connectionConfig connection.Record) (providerregistry.UsageInfo, error) {
			return NewClient(connectionConfig).Usage(ctx)
		},
		GenerateOAuthURL: func(connectionConfig connection.Record) (string, error) {
			return GenerateOAuthURL(connectionConfig)
		},
		StartOAuth: func(connectionConfig connection.Record) (providerregistry.OAuthSession, error) {
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
		CompleteOAuth: func(connectionConfig connection.Record, pending map[string]string, callbackURL string) (providerregistry.OAuthResult, error) {
			completed, err := CompleteCodexOAuthFromCallbackURL(PendingCodexOAuth{
				CodeVerifier: pending["code_verifier"],
				State:        pending["state"],
				RedirectURI:  pending["redirect_uri"],
			}, callbackURL)
			if err != nil {
				return providerregistry.OAuthResult{}, err
			}

			return providerregistry.OAuthResult{
				AccessToken:          completed.AccessToken,
				RefreshToken:         completed.RefreshToken,
				TokenType:            completed.TokenType,
				ExpiresIn:            completed.ExpiresIn,
				AccessTokenExpiresAt: calculateAccessTokenExpiresAt(completed.ExpiresIn),
				Name:                 completed.Email,
			}, nil
		},
		ValidateConnection: func(connectionConfig connection.Record) []string {
			if strings.TrimSpace(connectionConfig.AccessToken) == "" && strings.TrimSpace(connectionConfig.APIKey) == "" {
				return []string{"missing access_token or api_key"}
			}

			return nil
		},
	}
}
