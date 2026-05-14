package listmodels

import "github.com/phamtanminhtien/goroute/internal/domain/driver"

type ModelView struct {
	ID      string
	Object  string
	OwnedBy string
}

func Execute(catalog driver.Catalog) []ModelView {
	models := make([]ModelView, 0)
	for _, drv := range catalog.Drivers {
		for _, model := range drv.Models {
			models = append(models, ModelView{
				ID:      model.ID,
				Object:  "model",
				OwnedBy: drv.Provider,
			})
		}
	}

	return models
}
