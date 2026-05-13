package listmodels

import "github.com/phamtanminhtien/goroute/internal/domain/driver"

type ModelView struct {
	ID      string
	Object  string
	OwnedBy string
}

func Execute(catalog driver.Catalog) []ModelView {
	models := make([]ModelView, 0, len(catalog.Drivers))
	for _, drv := range catalog.Drivers {
		models = append(models, ModelView{
			ID:      drv.ID,
			Object:  "model",
			OwnedBy: "goroute",
		})
	}

	return models
}
