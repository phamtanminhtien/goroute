package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/rs/zerolog"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		ctx := chatcompletion.WithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	return fmt.Sprintf("req-%s", uuid.NewString())
}

func loggingMiddleware(logger *zerolog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		noop := zerolog.Nop()
		logger = &noop
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(recorder, r)

			logger.Info().
				Str("request_id", chatcompletion.RequestID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Int("status", recorder.statusCode).
				Int("bytes_written", recorder.bytesWritten).
				Int64("duration_ms", time.Since(started).Milliseconds()).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("http_request")
		})
	}
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

func openAICompatibleAuthMiddleware(settingsManager *config.SettingsManager, repo systemAPIKeyAuthRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settingsManager == nil || !settingsManager.OpenAICompatibleAuth().Enabled || repo == nil {
			next.ServeHTTP(w, r)
			return
		}

		hasKeys, err := repo.HasSystemAPIKeys()
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !hasKeys {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}

		key := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if key == "" {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		if _, ok, err := repo.AuthenticateSystemAPIKey(key); err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		} else if !ok {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func anthropicCompatibleAuthMiddleware(settingsManager *config.SettingsManager, repo systemAPIKeyAuthRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if settingsManager == nil || !settingsManager.OpenAICompatibleAuth().Enabled || repo == nil {
			next.ServeHTTP(w, r)
			return
		}

		hasKeys, err := repo.HasSystemAPIKeys()
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !hasKeys {
			next.ServeHTTP(w, r)
			return
		}

		key := strings.TrimSpace(r.Header.Get("x-api-key"))
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(r, w, http.StatusUnauthorized, "unauthorized", "missing bearer token or x-api-key")
				return
			}
			key = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if key == "" {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "missing bearer token or x-api-key")
			return
		}
		if _, ok, err := repo.AuthenticateSystemAPIKey(key); err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		} else if !ok {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	written, err := r.ResponseWriter.Write(body)
	r.bytesWritten += written
	return written, err
}

func (r *statusRecorder) Flush() {
	flusher, ok := r.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	flusher.Flush()
}
