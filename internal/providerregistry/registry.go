package providerregistry

import (
	"context"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type Registration struct {
	Descriptor         provider.Provider
	BuildConnection    func(connection.Record) (chatcompletion.Connection, error)
	ValidateConnection func(connection.Record) []string
	GetAccessToken     func(connection.Record) (string, error)
	GetUsage           func(context.Context, connection.Record) (UsageInfo, error)
	GenerateOAuthURL   func(connection.Record) (string, error)
	StartOAuth         func(connection.Record) (OAuthSession, error)
	CompleteOAuth      func(connection.Record, map[string]string, string) (OAuthResult, error)
}

type UsageInfo struct {
	Plan               string                 `json:"plan,omitempty"`
	LimitReached       bool                   `json:"limitReached"`
	ReviewLimitReached bool                   `json:"reviewLimitReached"`
	Quotas             map[string]UsageWindow `json:"quotas,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

type UsageWindow struct {
	Used      int    `json:"used"`
	Total     int    `json:"total"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"resetAt,omitempty"`
	Unlimited bool   `json:"unlimited"`
}

type UsageUnavailableError struct {
	StatusCode int
}

func (e UsageUnavailableError) Error() string {
	if e.StatusCode <= 0 {
		return "usage api temporarily unavailable"
	}

	return fmt.Sprintf("usage api temporarily unavailable (%d)", e.StatusCode)
}

type OAuthSession struct {
	AuthorizationURL string
	Pending          map[string]string
}

type OAuthResult struct {
	AccessToken          string
	RefreshToken         string
	TokenType            string
	ExpiresIn            int
	AccessTokenExpiresAt int64
	Name                 string
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

func (r Registry) BuildConnection(connectionConfig connection.Record) (chatcompletion.Connection, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}

	return registration.BuildConnection(connectionConfig)
}

func (r Registry) ValidateConnection(connectionConfig connection.Record) []string {
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

func (r Registry) GetAccessToken(connectionConfig connection.Record) (string, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return "", fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.GetAccessToken == nil {
		return "", fmt.Errorf("provider %q does not support access token resolution", connectionConfig.ProviderID)
	}

	return registration.GetAccessToken(connectionConfig)
}

func (r Registry) GetUsage(ctx context.Context, connectionConfig connection.Record) (UsageInfo, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return UsageInfo{}, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.GetUsage == nil {
		return UsageInfo{}, fmt.Errorf("provider %q does not support usage lookup", connectionConfig.ProviderID)
	}

	return registration.GetUsage(ctx, connectionConfig)
}

func (r Registry) GenerateOAuthURL(connectionConfig connection.Record) (string, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return "", fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.GenerateOAuthURL == nil {
		return "", fmt.Errorf("provider %q does not support oauth authorization url generation", connectionConfig.ProviderID)
	}

	return registration.GenerateOAuthURL(connectionConfig)
}

func (r Registry) StartOAuth(connectionConfig connection.Record) (OAuthSession, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return OAuthSession{}, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.StartOAuth == nil {
		return OAuthSession{}, fmt.Errorf("provider %q does not support oauth start", connectionConfig.ProviderID)
	}

	return registration.StartOAuth(connectionConfig)
}

func (r Registry) CompleteOAuth(connectionConfig connection.Record, pending map[string]string, callbackURL string) (OAuthResult, error) {
	registration, ok := r.byID[connectionConfig.ProviderID]
	if !ok {
		return OAuthResult{}, fmt.Errorf("unsupported provider %q", connectionConfig.ProviderID)
	}
	if registration.CompleteOAuth == nil {
		return OAuthResult{}, fmt.Errorf("provider %q does not support oauth completion", connectionConfig.ProviderID)
	}

	return registration.CompleteOAuth(connectionConfig, pending, callbackURL)
}
