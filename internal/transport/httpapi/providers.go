package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func connectionsHandler(service *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"object": "list",
				"data":   service.List(),
			})
		case http.MethodPost:
			var input connection.Record
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			item, err := service.Create(input)
			if err != nil {
				writeConnectionMutationError(r, w, err)
				return
			}

			writeJSON(w, http.StatusCreated, item)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func connectionByIDHandler(service *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" || strings.Contains(id, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "connection not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			item, ok := service.Get(id)
			if !ok {
				writeError(r, w, http.StatusNotFound, "not_found", "connection not found")
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodPut:
			var input connection.Record
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
				return
			}

			item, err := service.Update(id, input)
			if err != nil {
				writeConnectionMutationError(r, w, err)
				return
			}

			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if err := service.Delete(id); err != nil {
				writeConnectionMutationError(r, w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	})
}

func connectionUsageHandler(service *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		id := chi.URLParam(r, "id")
		if id == "" || strings.Contains(id, "/") {
			writeError(r, w, http.StatusNotFound, "not_found", "connection not found")
			return
		}

		usage, err := service.GetUsage(r.Context(), id)
		if err != nil {
			var notFound connectionsusecase.ErrNotFound
			var unavailable providerregistry.UsageUnavailableError
			switch {
			case errors.As(err, &notFound):
				writeError(r, w, http.StatusNotFound, "not_found", err.Error())
			case errors.As(err, &unavailable):
				writeJSON(w, http.StatusOK, map[string]string{
					"message": fmt.Sprintf("Codex connected. Usage API temporarily unavailable (%d).", unavailable.StatusCode),
				})
			default:
				writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
			}
			return
		}

		writeJSON(w, http.StatusOK, usage)
	})
}

type completeConnectionOAuthRequest struct {
	SessionID   string `json:"session_id"`
	CallbackURL string `json:"callback_url"`
}

func connectionOAuthHandler(service *connectionsusecase.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var input completeConnectionOAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(r, w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
			return
		}

		item, err := service.CompleteOAuth(input.SessionID, input.CallbackURL)
		if err != nil {
			writeConnectionMutationError(r, w, err)
			return
		}

		writeJSON(w, http.StatusCreated, item)
	})
}

func writeConnectionMutationError(r *http.Request, w http.ResponseWriter, err error) {
	var notFound connectionsusecase.ErrNotFound
	switch {
	case errors.As(err, &notFound):
		writeError(r, w, http.StatusNotFound, "not_found", err.Error())
	case strings.Contains(err.Error(), "already exists"):
		writeError(r, w, http.StatusConflict, "conflict", err.Error())
	default:
		writeError(r, w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
