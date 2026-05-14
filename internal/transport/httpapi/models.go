package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/listmodels"
)

func modelsHandler(catalog driver.Catalog) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		items := listmodels.Execute(catalog)
		response := openaiwire.ListModelsResponse{
			Object: "list",
			Data:   make([]openaiwire.Model, 0, len(items)),
		}
		for _, item := range items {
			response.Data = append(response.Data, openaiwire.Model{
				ID:       item.ID,
				Object:   item.Object,
				OwnedBy:  item.OwnedBy,
				Root:     item.Root,
				Parent:   item.Parent,
				Metadata: item.Metadata,
			})
		}

		writeJSON(w, http.StatusOK, response)
	})
}
