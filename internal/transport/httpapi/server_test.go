package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

func testCatalog() driver.Catalog {
	return driver.Catalog{
		Drivers: []driver.Driver{
			{ID: "cx", Name: "Codex", Provider: "codex", DefaultModel: "cx/gpt-5.4"},
			{ID: "opena", Name: "OpenAI", Provider: "openai", DefaultModel: "opena/gpt-4.1"},
		},
	}
}

func testProviderRegistry(provider chatcompletion.Provider) chatcompletion.ProviderRegistry {
	return chatcompletion.NewProviderRegistry(map[string][]chatcompletion.Provider{"codex": {provider}})
}

func TestAuthMiddlewareRequiresBearerToken(t *testing.T) {
	handler := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddlewareAllowsValidBearerToken(t *testing.T) {
	handler := authMiddleware("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestModelsDoesNotRequireAuth(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(testProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestModelsReturnsConfiguredPrefixes(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(testProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChatCompletionsAcceptsPrefixedModel(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(testProvider{response: openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt-5.4", Choices: []openaiwire.ChatChoice{{Index: 0, Message: openaiwire.ChatMessage{Role: "assistant", Content: "hello back"}}}}}))
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"model":"cx/gpt-5.4"`) {
		t.Fatalf("expected response model to stay prefixed, got body=%s", rec.Body.String())
	}
}

func TestChatCompletionsMapsUpstreamErrorsToBadGateway(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(testProvider{err: chatcompletion.UpstreamError{StatusCode: http.StatusTooManyRequests, Message: "rate limited"}}))
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadGateway, rec.Code, rec.Body.String())
	}
}

func TestChatCompletionsStreamsProviderBody(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(streamingTestProvider{body: "data: first\n\n"}))
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", got)
	}
	if rec.Body.String() != "data: first\n\n" {
		t.Fatalf("unexpected stream body=%q", rec.Body.String())
	}
}
