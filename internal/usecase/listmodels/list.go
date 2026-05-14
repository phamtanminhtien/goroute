package listmodels

import "github.com/phamtanminhtien/goroute/internal/domain/driver"

type ModelView struct {
	ID       string
	Object   string
	OwnedBy  string
	Root     string
	Parent   string
	Metadata map[string]string
}

func Execute(catalog driver.Catalog) []ModelView {
	models := make([]ModelView, 0)
	for _, drv := range catalog.Drivers {
		if len(drv.Models) == 0 && drv.DefaultModel != "" {
			models = append(models, buildModelView(drv, driver.Model{ID: drv.DefaultModel}))
			continue
		}
		for _, model := range drv.Models {
			models = append(models, buildModelView(drv, model))
		}
	}

	return models
}

func buildModelView(drv driver.Driver, model driver.Model) ModelView {
	metadata := map[string]string{
		"driver_id":     drv.ID,
		"driver_name":   drv.Name,
		"provider_type": drv.Provider,
		"auth_type":     drv.AuthType,
		"is_default":    boolString(model.ID == drv.DefaultModel),
		"display_name":  model.Name,
		"description":   model.Description,
		"default_model": drv.DefaultModel,
	}

	return ModelView{
		ID:       model.ID,
		Object:   "model",
		OwnedBy:  drv.Provider,
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
