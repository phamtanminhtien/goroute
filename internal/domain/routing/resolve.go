package routing

import (
	"fmt"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

func ResolveModel(catalog provider.Catalog, model string) (Target, error) {
	prefix, requestedModel, err := splitModel(model)
	if err != nil {
		return Target{}, err
	}

	resolvedProvider, ok := catalog.FindByID(prefix)
	if !ok {
		return Target{}, fmt.Errorf("unknown model prefix %q", prefix)
	}

	if requestedModel == "" {
		requestedModel = strings.TrimPrefix(resolvedProvider.DefaultModel, resolvedProvider.ID+"/")
	}

	return Target{
		Prefix:         prefix,
		RequestedModel: requestedModel,
		ProviderID:     resolvedProvider.ID,
		ProviderName:   resolvedProvider.Name,
	}, nil
}

func splitModel(model string) (prefix string, requestedModel string, err error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", fmt.Errorf("model is required")
	}

	parts := strings.SplitN(model, "/", 2)
	prefix = parts[0]
	if prefix == "" {
		return "", "", fmt.Errorf("model prefix is required")
	}
	if len(parts) == 2 {
		requestedModel = strings.TrimSpace(parts[1])
	}

	return prefix, requestedModel, nil
}
