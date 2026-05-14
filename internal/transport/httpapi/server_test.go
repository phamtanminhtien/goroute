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

func TestModelsRequiresAuth(t *testing.T) {
	handler := NewServer("secret", testCatalog(), testProviderRegistry(testProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestModelsReturnsConfiguredPrefixes(t *testing.T) {
	handler := NewServer("secret", testCatalog(), testProviderRegistry(testProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChatCompletionsAcceptsPrefixedModel(t *testing.T) {
	handler := NewServer("secret", testCatalog(), testProviderRegistry(testProvider{response: openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt-5.4", Choices: []openaiwire.ChatChoice{{Index: 0, Message: openaiwire.ChatMessage{Role: "assistant", Content: "hello back"}}}}}))
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
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
	handler := NewServer("secret", testCatalog(), testProviderRegistry(testProvider{err: chatcompletion.UpstreamError{StatusCode: http.StatusTooManyRequests, Message: "rate limited"}}))
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadGateway, rec.Code, rec.Body.String())
	}
}
