package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

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

func writeUpstreamError(w http.ResponseWriter, upstreamErr chatcompletion.UpstreamError) {
	body := strings.TrimSpace(upstreamErr.Message)
	if body == "" {
		w.WriteHeader(upstreamErr.StatusCode)
		return
	}

	if json.Valid([]byte(body)) {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(upstreamErr.StatusCode)
	_, _ = w.Write([]byte(body))
}
