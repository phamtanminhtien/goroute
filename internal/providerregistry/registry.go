package providerregistry

import (
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type Registration struct {
	Descriptor         provider.Provider
	BuildConnection    func(config.ConnectionConfig) (chatcompletion.Connection, error)
	ValidateConnection func(config.ConnectionConfig) []string
	GenerateOAuthURL   func(config.ConnectionConfig) (string, error)
	StartOAuth         func(config.ConnectionConfig) (OAuthSession, error)
	CompleteOAuth      func(config.ConnectionConfig, map[string]string, string) (OAuthResult, error)
}

type OAuthSession struct {
	AuthorizationURL string
	Pending          map[string]string
}

type OAuthResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Name         string
}

type Registry struct {
	ordered []Registration
	byID    map[string]Registration
}

func New(registrations ...Registration) (Registry, error) {
	ordered := make([]Registration, 0, len(registrations))
	byID := make(map[string]Registration, len(registrations))

	for _, registration := range registrations {
		if registration.Descriptor.ID == "" {
			return Registry{}, fmt.Errorf("provider registration id is required")
		}
		if registration.BuildConnection == nil {
			return Registry{}, fmt.Errorf("provider %q build connection is required", registration.Descriptor.ID)
		}
		if _, exists := byID[registration.Descriptor.ID]; exists {
			return Registry{}, fmt.Errorf("provider %q already registered", registration.Descriptor.ID)
		}

		ordered = append(ordered, registration)
		byID[registration.Descriptor.ID] = registration
	}

	return Registry{
		ordered: ordered,
		byID:    byID,
	}, nil
}

func (r Registry) Catalog() provider.Catalog {
	providers := make([]provider.Provider, 0, len(r.ordered))
	for _, registration := range r.ordered {
		providers = append(providers, registration.Descriptor)
	}

	return provider.Catalog{Providers: providers}
}

func (r Registry) BuildConnection(connectionConfig config.ConnectionConfig) (chatcompletion.Connection, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}

	return registration.BuildConnection(connectionConfig)
}

func (r Registry) ValidateConnection(connectionConfig config.ConnectionConfig) []string {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return []string{"unsupported provider"}
	}
	if registration.ValidateConnection == nil {
		return nil
	}

	problems := registration.ValidateConnection(connectionConfig)
	return append([]string(nil), problems...)
}

func (r Registry) GenerateOAuthURL(connectionConfig config.ConnectionConfig) (string, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return "", fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.GenerateOAuthURL == nil {
		return "", fmt.Errorf("provider %q does not support oauth authorization url generation", connectionConfig.ProviderID)
	}

	return registration.GenerateOAuthURL(connectionConfig)
}

func (r Registry) StartOAuth(connectionConfig config.ConnectionConfig) (OAuthSession, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return OAuthSession{}, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.StartOAuth == nil {
		return OAuthSession{}, fmt.Errorf("provider %q does not support oauth start", connectionConfig.ProviderID)
	}

	return registration.StartOAuth(connectionConfig)
}

func (r Registry) CompleteOAuth(connectionConfig config.ConnectionConfig, pending map[string]string, callbackURL string) (OAuthResult, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return OAuthResult{}, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.CompleteOAuth == nil {
		return OAuthResult{}, fmt.Errorf("provider %q does not support oauth completion", connectionConfig.ProviderID)
	}

	return registration.CompleteOAuth(connectionConfig, pending, callbackURL)
}
