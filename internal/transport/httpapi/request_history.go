package httpapi

import (
	"net/http"
	"strconv"

	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func requestHistoryHandler(providerRegistry chatcompletion.ProviderRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeError(r, w, http.StatusBadRequest, "invalid_request", "limit must be a positive integer")
				return
			}
			limit = parsed
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   providerRegistry.RecentRequestAttempts(limit),
		})
	})
}
