package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
)

func testCatalog() driver.Catalog {
	return driver.Catalog{
		Drivers: []driver.Driver{
			{ID: "cx", Name: "Codex", DefaultModel: "cx/gpt-5.4"},
			{ID: "opena", Name: "OpenAI", DefaultModel: "opena/gpt-4.1"},
		},
	}
}

func TestModelsRequiresAuth(t *testing.T) {
	handler := NewServer("secret", testCatalog())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestModelsReturnsConfiguredPrefixes(t *testing.T) {
	handler := NewServer("secret", testCatalog())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChatCompletionsAcceptsPrefixedModel(t *testing.T) {
	handler := NewServer("secret", testCatalog())
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}
