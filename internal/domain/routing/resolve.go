package routing

import (
	"fmt"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
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

func ResolvePlan(catalog provider.Catalog, combos []modelcombo.Combo, model string) (Plan, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return Plan{}, fmt.Errorf("model is required")
	}

	for _, combo := range combos {
		if combo.Alias != model {
			continue
		}
		if !combo.Enabled {
			break
		}
		if len(combo.Targets) == 0 {
			return Plan{}, fmt.Errorf("model combo %q has no targets", combo.Alias)
		}

		targets := make([]Target, 0, len(combo.Targets))
		for _, comboTarget := range combo.Targets {
			if !comboTarget.Enabled {
				continue
			}
			target, err := ResolveModel(catalog, comboTarget.ModelID)
			if err != nil {
				return Plan{}, fmt.Errorf("resolve combo target %q: %w", comboTarget.ModelID, err)
			}
			if comboTarget.ProviderID != "" && target.ProviderID != comboTarget.ProviderID {
				return Plan{}, fmt.Errorf("combo target %q does not belong to provider %q", comboTarget.ModelID, comboTarget.ProviderID)
			}
			target.ConnectionID = comboTarget.ConnectionID
			targets = append(targets, target)
		}
		if len(targets) == 0 {
			return Plan{}, fmt.Errorf("model combo %q has no enabled targets", combo.Alias)
		}

		return Plan{
			RequestedModel: model,
			ResponseModel:  combo.Alias,
			Targets:        targets,
			ComboAlias:     combo.Alias,
		}, nil
	}

	target, err := ResolveModel(catalog, model)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		RequestedModel: model,
		ResponseModel:  resolvedModel(target),
		Targets:        []Target{target},
	}, nil
}

func resolvedModel(target Target) string {
	if target.Prefix == "" || target.RequestedModel == "" {
		return target.RequestedModel
	}

	return target.Prefix + "/" + target.RequestedModel
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
