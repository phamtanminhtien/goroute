package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/phamtanminhtien/goroute/internal/logging"
)

var logsStreamHeartbeatInterval = 15 * time.Second

func logsStreamHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(r, w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(r, w, http.StatusInternalServerError, "stream_unsupported", "streaming not supported")
			return
		}

		stream, unsubscribe := logging.DefaultBroadcaster().Subscribe()
		defer unsubscribe()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		ticker := time.NewTicker(logsStreamHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case line, ok := <-stream:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "event: log\ndata: %s\n\n", line); err != nil {
					return
				}
				flusher.Flush()
			case <-ticker.C:
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})
}
