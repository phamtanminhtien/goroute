package chatcompletion

import (
	"context"
	"fmt"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type ProviderRegistry struct {
	providers map[string][]Provider
}

func NewProviderRegistry(providers map[string][]Provider) ProviderRegistry {
	return ProviderRegistry{providers: providers}
}

func (r ProviderRegistry) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	providers := r.providers[target.ProviderType]
	if len(providers) == 0 {
		return openaiwire.ChatCompletionsResponse{}, fmt.Errorf("no executor configured for provider type %q", target.ProviderType)
	}

	var lastErr error
	for _, provider := range providers {
		response, err := provider.ChatCompletions(ctx, req, target)
		if err == nil {
			return response, nil
		}
		lastErr = err
	}

	return openaiwire.ChatCompletionsResponse{}, lastErr
}
