package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func chatCompletionsHandler(catalog driver.Catalog, providerRegistry chatcompletion.ProviderRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		defer r.Body.Close()

		var request openaiwire.ChatCompletionsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("invalid JSON body: %v", err))
			return
		}

		output, err := chatcompletion.Execute(r.Context(), catalog, providerRegistry, chatcompletion.Input{Request: request})
		if err != nil {
			var upstreamErr chatcompletion.UpstreamError
			switch {
			case errors.As(err, &upstreamErr):
				writeError(w, http.StatusBadGateway, "upstream_error", upstreamErr.Error())
			default:
				writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			}
			return
		}

		writeJSON(w, http.StatusOK, output.Response)
	})
}
