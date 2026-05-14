package httpapi

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

var requestCounter uint64

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d", atomic.AddUint64(&requestCounter, 1))
		ctx := chatcompletion.WithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("request_id=%s method=%s path=%s duration=%s", chatcompletion.RequestID(r.Context()), r.Method, r.URL.Path, time.Since(started))
	})
}

func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		if strings.TrimPrefix(authHeader, "Bearer ") != token {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
