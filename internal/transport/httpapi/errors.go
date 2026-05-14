package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(r *http.Request, w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, openaiwire.ErrorEnvelope{
		Error: openaiwire.ErrorBody{
			Message:   message,
			Type:      code,
			Code:      code,
			RequestID: chatcompletion.RequestID(r.Context()),
		},
	})
}
