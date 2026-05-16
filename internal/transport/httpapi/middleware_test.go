package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/phamtanminhtien/goroute/internal/logging"
)

func TestLoggingMiddlewareEmitsStructuredRequestFields(t *testing.T) {
	var logs bytes.Buffer
	logger := logging.NewWithWriter("prod", &logs)
	handler := requestIDMiddleware(loggingMiddleware(&logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})))

	req := httptest.NewRequest(http.MethodGet, "/v1/models?limit=1", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("User-Agent", "goroute-test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(logs.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON log output, got error: %v", err)
	}
	if payload["message"] != "http_request" {
		t.Fatalf("expected http_request message, got %#v", payload["message"])
	}
	if payload["method"] != http.MethodGet {
		t.Fatalf("expected method field, got %#v", payload["method"])
	}
	if payload["path"] != "/v1/models" {
		t.Fatalf("expected path field, got %#v", payload["path"])
	}
	if payload["query"] != "limit=1" {
		t.Fatalf("expected query field, got %#v", payload["query"])
	}
	if payload["status"] != float64(http.StatusCreated) {
		t.Fatalf("expected status field, got %#v", payload["status"])
	}
	if payload["request_id"] == "" {
		t.Fatalf("expected request_id field, got %#v", payload["request_id"])
	}

	headerRequestID := rec.Header().Get("X-Request-ID")
	if headerRequestID == "" {
		t.Fatal("expected X-Request-ID header")
	}

	loggedRequestID, ok := payload["request_id"].(string)
	if !ok {
		t.Fatalf("expected request_id field to be a string, got %#v", payload["request_id"])
	}
	if loggedRequestID != headerRequestID {
		t.Fatalf("expected header and log request IDs to match, got header=%q log=%q", headerRequestID, loggedRequestID)
	}
	if !strings.HasPrefix(loggedRequestID, "req-") {
		t.Fatalf("expected request_id to start with req-, got %q", loggedRequestID)
	}
	if _, err := uuid.Parse(strings.TrimPrefix(loggedRequestID, "req-")); err != nil {
		t.Fatalf("expected request_id suffix to be a UUID, got %q: %v", loggedRequestID, err)
	}
}
