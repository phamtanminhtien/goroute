package httpapi

import (
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/listmodels"
)

func modelsHandler(catalog provider.Catalog, modelRepo providerModelRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		resolvedCatalog, err := catalogWithCustomModels(catalog, modelRepo)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		items := listmodels.Execute(resolvedCatalog)
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
