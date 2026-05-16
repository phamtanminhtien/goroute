package httpapi

import (
	"net/http"
	"strconv"

	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func requestHistoryHandler(connectionRegistry *chatcompletion.ConnectionRegistry) http.Handler {
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

		history, err := connectionRegistry.RecentRequestAttempts(limit)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", "failed to load request history")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list",
			"data":   history,
		})
	})
}
