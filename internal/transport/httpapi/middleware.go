package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/systemapikey"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	"github.com/rs/zerolog"
)

type systemAPIKeyContextKey struct{}

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

func openAICompatibleAuthMiddleware(settingsManager *config.SettingsManager, repo systemAPIKeyAuthRepository, enforceTokenQuota bool, next http.Handler) http.Handler {
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
		record, ok, err := repo.AuthenticateSystemAPIKey(key)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !ok {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			return
		}
		if !enforceSystemAPIKeyQuota(r, w, repo, record, enforceTokenQuota) {
			return
		}

		next.ServeHTTP(w, r.WithContext(withSystemAPIKey(r.Context(), record)))
	})
}

func anthropicCompatibleAuthMiddleware(settingsManager *config.SettingsManager, repo systemAPIKeyAuthRepository, enforceTokenQuota bool, next http.Handler) http.Handler {
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
		record, ok, err := repo.AuthenticateSystemAPIKey(key)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !ok {
			writeError(r, w, http.StatusUnauthorized, "unauthorized", "invalid API key")
			return
		}
		if !enforceSystemAPIKeyQuota(r, w, repo, record, enforceTokenQuota) {
			return
		}

		next.ServeHTTP(w, r.WithContext(withSystemAPIKey(r.Context(), record)))
	})
}

func withSystemAPIKey(ctx context.Context, record systemapikey.Record) context.Context {
	return context.WithValue(ctx, systemAPIKeyContextKey{}, record)
}

func systemAPIKeyFromContext(ctx context.Context) (systemapikey.Record, bool) {
	record, ok := ctx.Value(systemAPIKeyContextKey{}).(systemapikey.Record)
	return record, ok
}

func enforceSystemAPIKeyQuota(r *http.Request, w http.ResponseWriter, repo systemAPIKeyAuthRepository, record systemapikey.Record, enforceTokenQuota bool) bool {
	now := time.Now().UTC()
	if record.RequestsPerMinuteLimit != nil {
		count, err := repo.CountSystemAPIKeyRequestsSince(record.ID, now.Add(-time.Minute))
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
			return false
		}
		if count >= *record.RequestsPerMinuteLimit {
			w.Header().Set("Retry-After", "60")
			writeError(r, w, http.StatusTooManyRequests, "rate_limit_exceeded", "system API key request-per-minute limit exceeded")
			return false
		}
	}

	if enforceTokenQuota {
		if record.DailyTokenLimit != nil {
			used, err := repo.SystemAPIKeyTokenUsage(record.ID, utcDayStart(now), utcDayStart(now).AddDate(0, 0, 1))
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return false
			}
			if used >= *record.DailyTokenLimit {
				writeError(r, w, http.StatusTooManyRequests, "quota_exceeded", "system API key daily token quota exceeded")
				return false
			}
		}
		if record.MonthlyTokenLimit != nil {
			used, err := repo.SystemAPIKeyTokenUsage(record.ID, utcMonthStart(now), utcMonthStart(now).AddDate(0, 1, 0))
			if err != nil {
				writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
				return false
			}
			if used >= *record.MonthlyTokenLimit {
				writeError(r, w, http.StatusTooManyRequests, "quota_exceeded", "system API key monthly token quota exceeded")
				return false
			}
		}
	}

	if err := repo.CreateSystemAPIKeyRequestEvent(systemapikey.RequestEvent{
		SystemAPIKeyID: record.ID,
		RequestID:      chatcompletion.RequestID(r.Context()),
		Path:           r.URL.Path,
		CreatedAt:      now.UnixMilli(),
	}); err != nil {
		writeError(r, w, http.StatusInternalServerError, "internal_error", err.Error())
		return false
	}

	return true
}

func utcDayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func utcMonthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
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
