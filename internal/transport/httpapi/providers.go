package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/config"
	providersusecase "github.com/phamtanminhtien/goroute/internal/usecase/providers"
)

func providersHandler(service *providersusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"object": "list",
				"data":   service.List(),
			})
		case http.MethodPost:
			var input config.ProviderConfig
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			item, err := service.Create(input)
			if err != nil {
				writeProviderMutationError(r, w, err)
				return
			}

			writeJSON(w, http.StatusCreated, item)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func providerByIDHandler(service *providersusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/")
		if id == "" || strings.Contains(id, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			item, ok := service.Get(id)
			if !ok {
				writeError(r, w, http.StatusNotFound, "not_found", "provider not found")
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodPut:
			var input config.ProviderConfig
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			item, err := service.Update(id, input)
			if err != nil {
				writeProviderMutationError(r, w, err)
				return
			}

			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if err := service.Delete(id); err != nil {
				writeProviderMutationError(r, w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func writeProviderMutationError(r *http.Request, w http.ResponseWriter, err error) {
	var notFound providersusecase.ErrNotFound
	switch {
	case errors.As(err, &notFound):
		writeError(r, w, http.StatusNotFound, "not_found", err.Error())
	case strings.Contains(err.Error(), "already exists"):
		writeError(r, w, http.StatusConflict, "conflict", err.Error())
	default:
		writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
