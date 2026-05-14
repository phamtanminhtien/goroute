package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/driver"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	providersusecase "github.com/phamtanminhtien/goroute/internal/usecase/providers"
)

func testCatalog() driver.Catalog {
	return driver.Catalog{
		Drivers: []driver.Driver{
			{ID: "cx", Name: "Codex", Provider: "codex", DefaultModel: "cx/gpt-5.4"},
			{ID: "opena", Name: "OpenAI", Provider: "openai", DefaultModel: "opena/gpt-4.1"},
		},
	}
}

func testProviderRegistry(provider chatcompletion.Provider) *chatcompletion.ProviderRegistry {
	registry := chatcompletion.NewProviderRegistry(map[string][]chatcompletion.Provider{"codex": {provider}})
	return &registry
}

const testAdminToken = "secret"

type testRuntime struct {
	registry *chatcompletion.ProviderRegistry
}

func (r testRuntime) ReplaceProviders(providerConfigs []config.ProviderConfig) error {
	entries := make(map[string][]chatcompletion.ProviderEntry, len(providerConfigs))
	for _, providerConfig := range providerConfigs {
		entries[providerConfig.Type] = append(entries[providerConfig.Type], chatcompletion.ProviderEntry{
			ID:       providerConfig.ID,
			Name:     providerConfig.Name,
			Provider: &testProvider{},
		})
	}
	r.registry.ReplaceProviders(entries)
	return nil
}

func testServer(t *testing.T, provider chatcompletion.Provider) http.Handler {
	t.Helper()

	registry := testProviderRegistry(provider)
	cfg := config.Config{
		Server: config.ServerConfig{
			Listen:    ":2232",
			AuthToken: testAdminToken,
		},
		Providers: []config.ProviderConfig{
			{
				ID:          "codex-1",
				Type:        "codex",
				Name:        "codex-user",
				AccessToken: "secret-token",
			},
		},
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.SavePath(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	service := providersusecase.NewService(configPath, cfg, testRuntime{registry: registry}, nil)
	return NewServer(testCatalog(), registry, service, testAdminToken)
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
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestModelsReturnsConfiguredPrefixes(t *testing.T) {
	handler := testServer(t, &testProvider{})
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
	handler := testServer(t, &testProvider{response: openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt-5.4", Choices: []openaiwire.ChatChoice{{Index: 0, Message: openaiwire.ChatMessage{Role: "assistant", Content: "hello back"}}}}})
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
	handler := testServer(t, &testProvider{err: chatcompletion.UpstreamError{StatusCode: http.StatusTooManyRequests, Message: "rate limited"}})
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
	handler := testServer(t, streamingTestProvider{testProvider: &testProvider{}, body: "data: first\n\n"})
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
	handler := testServer(t, provider)
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
	handler := testServer(t, provider)

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
	handler := testServer(t, &testProvider{})
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
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/debug/requests", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestProvidersListReturnsRedactedItems(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"has_access_token":true`) {
		t.Fatalf("expected token presence to be exposed, got body=%s", body)
	}
	if strings.Contains(body, "secret-token") {
		t.Fatalf("expected secrets to stay redacted, got body=%s", body)
	}
}

func TestProvidersCreatePersistsProvider(t *testing.T) {
	handler := testServer(t, &testProvider{})
	body := []byte(`{"id":"openai-1","type":"openai","name":"openai-user","api_key":"sk-test"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/providers", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if !strings.Contains(listRec.Body.String(), `"id":"openai-1"`) {
		t.Fatalf("expected created provider in list, got body=%s", listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "sk-test") {
		t.Fatalf("expected created secret to stay redacted, got body=%s", listRec.Body.String())
	}
}

func TestProvidersDeleteRejectsLastProvider(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodDelete, "/admin/api/providers/codex-1", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "config.providers must contain at least one provider") {
		t.Fatalf("expected validation error, got body=%s", rec.Body.String())
	}
}
