package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/logging"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func testCatalog() provider.Catalog {
	return provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", AuthType: provider.AuthTypeOAuth, Category: "oauth", DefaultModel: "cx/gpt-5.4"},
			{ID: "opena", Name: "OpenAI", AuthType: provider.AuthTypeAPIKey, Category: "api_key", DefaultModel: "opena/gpt-4.1"},
		},
	}
}

func testConnectionRegistry(connection chatcompletion.Connection) *chatcompletion.ConnectionRegistry {
	registry := chatcompletion.NewConnectionRegistry(map[string][]chatcompletion.Connection{"cx": {connection}})
	return &registry
}

const testAdminToken = "secret"

type testRuntime struct {
	registry *chatcompletion.ConnectionRegistry
}

func (r testRuntime) ReplaceConnections(connectionConfigs []config.ConnectionConfig) error {
	entries := make(map[string][]chatcompletion.ConnectionEntry, len(connectionConfigs))
	for _, connectionConfig := range connectionConfigs {
		entries[connectionConfig.ProviderID] = append(entries[connectionConfig.ProviderID], chatcompletion.ConnectionEntry{
			ID:         connectionConfig.ID,
			Name:       connectionConfig.Name,
			ProviderID: connectionConfig.ProviderID,
			Connection: &testProvider{},
		})
	}
	r.registry.ReplaceConnections(entries)
	return nil
}

func testServer(t *testing.T, connection chatcompletion.Connection) http.Handler {
	return testServerWithUsageAndConnection(t, nil, connection)
}

func testServerWithWebUI(t *testing.T, connection chatcompletion.Connection, webUIRoot fs.FS) http.Handler {
	return testServerWithUsageAndConnectionAndWebUI(t, nil, connection, webUIRoot)
}

func testServerWithUsage(t *testing.T, getUsage func(context.Context, config.ConnectionConfig) (providerregistry.UsageInfo, error)) http.Handler {
	return testServerWithUsageAndConnection(t, getUsage, &testProvider{})
}

func testServerWithUsageAndConnection(t *testing.T, getUsage func(context.Context, config.ConnectionConfig) (providerregistry.UsageInfo, error), connection chatcompletion.Connection) http.Handler {
	return testServerWithUsageAndConnectionAndWebUI(t, getUsage, connection, nil)
}

func testServerWithUsageAndConnectionAndWebUI(t *testing.T, getUsage func(context.Context, config.ConnectionConfig) (providerregistry.UsageInfo, error), connection chatcompletion.Connection, webUIRoot fs.FS) http.Handler {
	t.Helper()

	registry := testConnectionRegistry(connection)
	cfg := config.Config{
		Server: config.ServerConfig{
			Listen:    ":2232",
			AuthToken: testAdminToken,
		},
		Connections: []config.ConnectionConfig{
			{
				ID:          "codex-1",
				ProviderID:  "cx",
				Name:        "codex-user",
				AccessToken: "secret-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			},
		},
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.SavePath(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	logger := logging.NewWithWriter("prod", &bytes.Buffer{})
	providers, err := providerregistry.New(
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
			BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
				return connection, nil
			},
			GetUsage: func(ctx context.Context, connection config.ConnectionConfig) (providerregistry.UsageInfo, error) {
				if getUsage != nil {
					return getUsage(ctx, connection)
				}

				return providerregistry.UsageInfo{
					Plan:               "plus",
					LimitReached:       false,
					ReviewLimitReached: false,
					Quotas: map[string]providerregistry.UsageWindow{
						"session": {
							Used:      42,
							Total:     100,
							Remaining: 58,
							ResetAt:   "2026-05-16T10:00:00.000Z",
						},
					},
				}, nil
			},
			GenerateOAuthURL: func(config.ConnectionConfig) (string, error) {
				return "https://auth.openai.com/oauth/authorize?provider=cx", nil
			},
			StartOAuth: func(connection config.ConnectionConfig) (providerregistry.OAuthSession, error) {
				return providerregistry.OAuthSession{
					AuthorizationURL: "https://auth.openai.com/oauth/authorize?provider=cx&flow=start",
					Pending: map[string]string{
						"state":         "test-state",
						"code_verifier": "test-verifier",
						"redirect_uri":  "http://localhost:1455/auth/callback",
					},
				}, nil
			},
			CompleteOAuth: func(connection config.ConnectionConfig, pending map[string]string, callbackURL string) (providerregistry.OAuthResult, error) {
				if pending["state"] != "test-state" {
					return providerregistry.OAuthResult{}, errors.New("state mismatch")
				}
				if !strings.Contains(callbackURL, "state=test-state") {
					return providerregistry.OAuthResult{}, errors.New("state mismatch")
				}
				return providerregistry.OAuthResult{
					AccessToken:  "oauth-access-token",
					RefreshToken: "oauth-refresh-token",
					TokenType:    "Bearer",
					ExpiresIn:    3600,
					Name:         "oauth-user@example.com",
				}, nil
			},
			ValidateConnection: func(connection config.ConnectionConfig) []string {
				if connection.AccessToken == "" && connection.APIKey == "" {
					return []string{"missing access_token or api_key"}
				}

				return nil
			},
		},
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "opena", Name: "OpenAI"},
			BuildConnection: func(config.ConnectionConfig) (chatcompletion.Connection, error) {
				return &testProvider{}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	service := connectionsusecase.NewService(configPath, cfg, testRuntime{registry: registry}, providers, &logger)
	return NewServer(testCatalog(), registry, service, testAdminToken, webUIRoot, &logger)
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
	if len(response.Data) == 0 || response.Data[0].Metadata["provider_id"] == "" {
		t.Fatalf("expected model metadata, got %#v", response.Data)
	}
}

func TestProviderOAuthURLReturnsGeneratedURL(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/oauth-url", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider_id":"cx"`) {
		t.Fatalf("expected provider id in body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"session_id":"`) {
		t.Fatalf("expected oauth session id in body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"url":"https://auth.openai.com/oauth/authorize?provider=cx\u0026flow=start"`) {
		t.Fatalf("expected started oauth url in body=%s", rec.Body.String())
	}
}

func TestProviderOAuthURLRejectsUnsupportedProvider(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/opena/oauth-url", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `does not support oauth start`) {
		t.Fatalf("expected unsupported oauth error in body=%s", rec.Body.String())
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

func TestWebUIServesBuiltIndexForRoot(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Fatalf("expected index html, got body=%s", rec.Body.String())
	}
}

func TestWebUIServesBuiltAssetPaths(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != `console.log("goroute");` {
		t.Fatalf("expected built asset, got body=%s", rec.Body.String())
	}
}

func TestWebUIFallsBackToIndexForSPARoutes(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/providers/cx", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<title>goroute</title>`) {
		t.Fatalf("expected index fallback, got body=%s", rec.Body.String())
	}
}

func TestWebUIFallsBackWithoutRedirectForLoginRoute(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect location, got %q", location)
	}
	if !strings.Contains(rec.Body.String(), `<title>goroute</title>`) {
		t.Fatalf("expected index fallback, got body=%s", rec.Body.String())
	}
}

func TestWebUIReturnsNotFoundForMissingAsset(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestWebUIDoesNotCaptureUnknownAPIPaths(t *testing.T) {
	webUIRoot := writeTestWebUI(t)
	handler := testServerWithWebUI(t, &testProvider{}, os.DirFS(webUIRoot))
	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `<title>goroute</title>`) {
		t.Fatalf("expected api 404 instead of spa fallback, got body=%s", rec.Body.String())
	}
}

func writeTestWebUI(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<!doctype html><html><head><title>goroute</title></head><body><div id="root"></div></body></html>`), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte(`console.log("goroute");`), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}

	return root
}

func TestChatCompletionsStreamsConnectionBody(t *testing.T) {
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

func TestConnectionsListReturnsRedactedItems(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/admin/api/connections", nil)
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

func TestProvidersListReturnsCatalogWithGroupedConnections(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/admin/api/providers", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Object string `json:"object"`
		Data   []struct {
			ID              string                    `json:"id"`
			Category        string                    `json:"category"`
			ConnectionCount int                       `json:"connection_count"`
			Connections     []connectionsusecase.Item `json:"connections"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Object != "list" {
		t.Fatalf("expected list object, got %q", response.Object)
	}
	if len(response.Data) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(response.Data))
	}
	if response.Data[0].ID != "cx" || response.Data[0].Category != "oauth" {
		t.Fatalf("expected codex provider metadata, got %#v", response.Data[0])
	}
	if response.Data[0].ConnectionCount != 1 {
		t.Fatalf("expected codex connection count, got %#v", response.Data[0])
	}
	if len(response.Data[0].Connections) != 1 || response.Data[0].Connections[0].ID != "codex-1" {
		t.Fatalf("expected codex connection to be grouped, got %#v", response.Data[0].Connections)
	}
	if response.Data[1].ID != "opena" || response.Data[1].Category != "api_key" {
		t.Fatalf("expected openai provider metadata, got %#v", response.Data[1])
	}
	if response.Data[1].ConnectionCount != 0 {
		t.Fatalf("expected openai connection count, got %#v", response.Data[1])
	}
	if len(response.Data[1].Connections) != 0 {
		t.Fatalf("expected openai provider to return empty connections, got %#v", response.Data[1].Connections)
	}
	if strings.Contains(rec.Body.String(), "secret-token") {
		t.Fatalf("expected grouped provider response to keep secrets redacted, got body=%s", rec.Body.String())
	}
}

func TestConnectionsCreatePersistsConnection(t *testing.T) {
	handler := testServer(t, &testProvider{})
	body := []byte(`{"id":"openai-1","provider_id":"opena","name":"openai-user","api_key":"sk-test"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/connections", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/connections", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if !strings.Contains(listRec.Body.String(), `"id":"openai-1"`) {
		t.Fatalf("expected created connection in list, got body=%s", listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "sk-test") {
		t.Fatalf("expected created secret to stay redacted, got body=%s", listRec.Body.String())
	}
}

func TestConnectionsUpdatePreservesSecretsWhenOmitted(t *testing.T) {
	handler := testServer(t, &testProvider{})
	body := []byte(`{"id":"codex-1","provider_id":"cx","name":"renamed-user"}`)
	req := httptest.NewRequest(http.MethodPut, "/admin/api/connections/codex-1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/connections", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if !strings.Contains(listRec.Body.String(), `"has_access_token":true`) {
		t.Fatalf("expected access token to remain configured, got body=%s", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"token_type":"Bearer"`) || !strings.Contains(listRec.Body.String(), `"expires_in":3600`) {
		t.Fatalf("expected oauth metadata to remain configured, got body=%s", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"name":"renamed-user"`) {
		t.Fatalf("expected updated name in list, got body=%s", listRec.Body.String())
	}
}

func TestConnectionsOAuthCompletionCreatesConnection(t *testing.T) {
	handler := testServer(t, &testProvider{})

	startReq := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/oauth-url", nil)
	startReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	startRec := httptest.NewRecorder()
	handler.ServeHTTP(startRec, startReq)

	if startRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, startRec.Code, startRec.Body.String())
	}

	var startResponse struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResponse); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if startResponse.SessionID == "" {
		t.Fatalf("expected non-empty session id in body=%s", startRec.Body.String())
	}

	completeBody := []byte(`{"session_id":"` + startResponse.SessionID + `","callback_url":"http://localhost:1455/auth/callback?code=abc123&state=test-state"}`)
	completeReq := httptest.NewRequest(http.MethodPost, "/admin/api/connections/oauth", bytes.NewReader(completeBody))
	completeReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	completeRec := httptest.NewRecorder()
	handler.ServeHTTP(completeRec, completeReq)

	if completeRec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, completeRec.Code, completeRec.Body.String())
	}
	if !strings.Contains(completeRec.Body.String(), `"id":"codex-2"`) {
		t.Fatalf("expected created oauth connection, got body=%s", completeRec.Body.String())
	}
	if !strings.Contains(completeRec.Body.String(), `"name":"oauth-user@example.com"`) {
		t.Fatalf("expected email-backed connection name, got body=%s", completeRec.Body.String())
	}
	if !strings.Contains(completeRec.Body.String(), `"token_type":"Bearer"`) || !strings.Contains(completeRec.Body.String(), `"expires_in":3600`) {
		t.Fatalf("expected oauth metadata in response, got body=%s", completeRec.Body.String())
	}
	if strings.Contains(completeRec.Body.String(), "oauth-access-token") {
		t.Fatalf("expected oauth access token to stay redacted, got body=%s", completeRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/connections", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if !strings.Contains(listRec.Body.String(), `"id":"codex-2"`) {
		t.Fatalf("expected oauth connection in list, got body=%s", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"has_access_token":true`) || !strings.Contains(listRec.Body.String(), `"has_refresh_token":true`) {
		t.Fatalf("expected oauth tokens to be persisted, got body=%s", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"token_type":"Bearer"`) || !strings.Contains(listRec.Body.String(), `"expires_in":3600`) {
		t.Fatalf("expected oauth metadata to be persisted, got body=%s", listRec.Body.String())
	}
}

func TestConnectionsDeleteRejectsLastConnection(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodDelete, "/admin/api/connections/codex-1", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestConnectionUsageReturnsNormalizedQuotaPayload(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/admin/api/connections/codex-1/usage", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"plan":"plus"`) {
		t.Fatalf("expected plan in body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"limitReached":false`) {
		t.Fatalf("expected limitReached in body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"session"`) || !strings.Contains(rec.Body.String(), `"used":42`) {
		t.Fatalf("expected session quota in body=%s", rec.Body.String())
	}
}

func TestConnectionUsageReturnsTemporaryUnavailableMessage(t *testing.T) {
	handler := testServerWithUsage(t, func(context.Context, config.ConnectionConfig) (providerregistry.UsageInfo, error) {
		return providerregistry.UsageInfo{}, providerregistry.UsageUnavailableError{StatusCode: http.StatusBadGateway}
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/api/connections/codex-1/usage", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `Usage API temporarily unavailable (502)`) {
		t.Fatalf("expected unavailable message in body=%s", rec.Body.String())
	}
}
