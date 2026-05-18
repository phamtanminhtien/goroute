package listmodels

import (
	"strconv"

	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
)

type ModelView struct {
	ID       string
	Object   string
	OwnedBy  string
	Root     string
	Parent   string
	Metadata map[string]string
}

func Execute(catalog provider.Catalog) []ModelView {
	models := make([]ModelView, 0)
	for _, resolvedProvider := range catalog.Providers {
		if len(resolvedProvider.Models) == 0 && resolvedProvider.DefaultModel != "" {
			models = append(models, buildModelView(resolvedProvider, provider.Model{ID: resolvedProvider.DefaultModel}))
			continue
		}
		for _, model := range resolvedProvider.Models {
			models = append(models, buildModelView(resolvedProvider, model))
		}
	}

	return models
}

func ExecuteWithCombos(catalog provider.Catalog, combos []modelcombo.Combo) []ModelView {
	models := Execute(catalog)
	for _, combo := range combos {
		models = append(models, ModelView{
			ID:      combo.Alias,
			Object:  "model",
			OwnedBy: "combo",
			Root:    combo.Alias,
			Parent:  "",
			Metadata: map[string]string{
				"is_combo":     "true",
				"display_name": combo.Name,
				"description":  combo.Description,
				"target_count": strconv.Itoa(len(combo.Targets)),
			},
		})
	}

	return models
}

func buildModelView(resolvedProvider provider.Provider, model provider.Model) ModelView {
	metadata := map[string]string{
		"provider_id":   resolvedProvider.ID,
		"provider_name": resolvedProvider.Name,
		"auth_type":     string(resolvedProvider.AuthType),
		"is_default":    boolString(model.ID == resolvedProvider.DefaultModel),
		"display_name":  model.Name,
		"description":   model.Description,
		"default_model": resolvedProvider.DefaultModel,
	}

	return ModelView{
		ID:       model.ID,
		Object:   "model",
		OwnedBy:  resolvedProvider.ID,
		Root:     model.ID,
		Parent:   "",
		Metadata: metadata,
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}

	return "false"
}
