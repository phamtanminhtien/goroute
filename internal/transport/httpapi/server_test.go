package httpapi

import (
	"bytes"
	"encoding/json"
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

const testAdminToken = "secret"

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
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{}), testAdminToken)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestModelsReturnsConfiguredPrefixes(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{}), testAdminToken)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	var response openaiwire.ListModelsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) == 0 || response.Data[0].Metadata["driver_id"] == "" {
		t.Fatalf("expected model metadata, got %#v", response.Data)
	}
}

func TestChatCompletionsAcceptsPrefixedModel(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{response: openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt-5.4", Choices: []openaiwire.ChatChoice{{Index: 0, Message: openaiwire.ChatMessage{Role: "assistant", Content: "hello back"}}}}}), testAdminToken)
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
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{err: chatcompletion.UpstreamError{StatusCode: http.StatusTooManyRequests, Message: "rate limited"}}), testAdminToken)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadGateway, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"request_id":"req-`) {
		t.Fatalf("expected request_id in error body, got body=%s", rec.Body.String())
	}
}

func TestChatCompletionsStreamsProviderBody(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(streamingTestProvider{testProvider: &testProvider{}, body: "data: first\n\n"}), testAdminToken)
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

func TestChatCompletionsAcceptsCommonOpenAIFields(t *testing.T) {
	provider := &testProvider{
		response: openaiwire.ChatCompletionsResponse{
			ID:     "chatcmpl-2",
			Object: "chat.completion",
			Model:  "gpt-5.4",
			Choices: []openaiwire.ChatChoice{{
				Index:   0,
				Message: openaiwire.ChatMessage{Role: "assistant", Content: "weather ready"},
			}},
		},
	}
	handler := NewServer(testCatalog(), testProviderRegistry(provider), testAdminToken)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}],"temperature":0.5,"max_tokens":64,"tools":[{"type":"function","function":{"name":"lookup_weather","parameters":{"type":"object"}}}],"tool_choice":{"type":"function","function":{"name":"lookup_weather"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if provider.lastReq.Temperature == nil || *provider.lastReq.Temperature != 0.5 {
		t.Fatalf("expected temperature to decode, got %#v", provider.lastReq.Temperature)
	}
	if provider.lastReq.MaxTokens == nil || *provider.lastReq.MaxTokens != 64 {
		t.Fatalf("expected max_tokens to decode, got %#v", provider.lastReq.MaxTokens)
	}
	if len(provider.lastReq.Tools) != 1 || provider.lastReq.Tools[0].Function.Name != "lookup_weather" {
		t.Fatalf("expected tools to decode, got %#v", provider.lastReq.Tools)
	}
}

func TestDebugRequestsReturnsStructuredAttemptHistory(t *testing.T) {
	provider := &testProvider{
		response: openaiwire.ChatCompletionsResponse{
			ID:     "chatcmpl-3",
			Object: "chat.completion",
			Model:  "gpt-5.4",
			Choices: []openaiwire.ChatChoice{{
				Index:   0,
				Message: openaiwire.ChatMessage{Role: "assistant", Content: "ok"},
			}},
		},
	}
	handler := NewServer(testCatalog(), testProviderRegistry(provider), testAdminToken)

	requestBody := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(requestBody))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	req := httptest.NewRequest(http.MethodGet, "/debug/requests?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"requested_model":"cx/gpt-5.4"`) {
		t.Fatalf("expected requested model in history, got body=%s", body)
	}
	if !strings.Contains(body, `"final_status":"success"`) {
		t.Fatalf("expected success final status in history, got body=%s", body)
	}
}

func TestDebugRequestsRequiresBearerToken(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{}), testAdminToken)
	req := httptest.NewRequest(http.MethodGet, "/debug/requests", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"unauthorized"`) {
		t.Fatalf("expected unauthorized error envelope, got body=%s", rec.Body.String())
	}
}

func TestDebugRequestsRejectsInvalidBearerToken(t *testing.T) {
	handler := NewServer(testCatalog(), testProviderRegistry(&testProvider{}), testAdminToken)
	req := httptest.NewRequest(http.MethodGet, "/debug/requests", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}
