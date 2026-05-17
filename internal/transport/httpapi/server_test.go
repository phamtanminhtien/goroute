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

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/logging"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/storage/gormsqlite"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func testCatalog() provider.Catalog {
	return provider.Catalog{
		Providers: []provider.Provider{
			{ID: "cx", Name: "Codex", AuthType: provider.AuthTypeOAuth, Category: "oauth", DefaultModel: "cx/gpt-5.4", Models: []provider.Model{{ID: "cx/gpt-5.4", Name: "GPT-5.4", InputPricePerMillionUSD: 1.25, OutputPricePerMillionUSD: 10}}},
			{ID: "opena", Name: "OpenAI", AuthType: provider.AuthTypeAPIKey, Category: "api_key", DefaultModel: "opena/gpt-4.1", Models: []provider.Model{{ID: "opena/gpt-4.1", Name: "GPT-4.1", InputPricePerMillionUSD: 2, OutputPricePerMillionUSD: 8}}},
		},
	}
}

type protocolTestConnection interface {
	chatcompletion.ChatCompletionsConnection
	chatcompletion.ResponsesConnection
}

func testConnectionRegistry(connection protocolTestConnection) *chatcompletion.ConnectionRegistry {
	registry := chatcompletion.NewConnectionRegistry(map[string][]chatcompletion.ConnectionEntry{
		"cx": {{
			ID:         "cx-1",
			Name:       "cx-1",
			ProviderID: "cx",
			ProtocolConnections: chatcompletion.ProtocolConnections{
				ChatCompletions: connection,
				Responses:       connection,
			},
		}},
	})
	return &registry
}

const testAdminToken = "secret"

type testRuntime struct {
	repo interface {
		ListConnections() ([]connection.Record, error)
	}
	providers providerregistry.Registry
	registry  *chatcompletion.ConnectionRegistry
}

func (r testRuntime) ReloadConnections() error {
	connectionConfigs, err := r.repo.ListConnections()
	if err != nil {
		return err
	}

	entries := make(map[string][]chatcompletion.ConnectionEntry, len(connectionConfigs))
	for _, connectionConfig := range connectionConfigs {
		connectionClient, err := r.providers.BuildConnection(connectionConfig)
		if err != nil {
			return err
		}
		entries[connectionConfig.ProviderID] = append(entries[connectionConfig.ProviderID], chatcompletion.ConnectionEntry{
			ID:                  connectionConfig.ID,
			Name:                connectionConfig.Name,
			ProviderID:          connectionConfig.ProviderID,
			ProtocolConnections: connectionClient,
		})
	}
	r.registry.ReplaceConnections(entries)
	return nil
}

func testServer(t *testing.T, connection protocolTestConnection) http.Handler {
	return testServerWithUsageAndConnection(t, nil, connection)
}

func testServerWithWebUI(t *testing.T, connection protocolTestConnection, webUIRoot fs.FS) http.Handler {
	return testServerWithUsageAndConnectionAndWebUI(t, nil, connection, webUIRoot)
}

func testServerWithUsage(t *testing.T, getUsage func(context.Context, connection.Record) (providerregistry.UsageInfo, error)) http.Handler {
	return testServerWithUsageAndConnection(t, getUsage, &testProvider{})
}

func testServerWithUsageAndConnection(t *testing.T, getUsage func(context.Context, connection.Record) (providerregistry.UsageInfo, error), connectionClient protocolTestConnection) http.Handler {
	return testServerWithUsageAndConnectionAndWebUI(t, getUsage, connectionClient, nil)
}

func testServerWithUsageAndConnectionAndWebUI(t *testing.T, getUsage func(context.Context, connection.Record) (providerregistry.UsageInfo, error), connectionClient protocolTestConnection, webUIRoot fs.FS) http.Handler {
	return testServerWithUsageAndConnectionAndWebUIAtPath(t, getUsage, connectionClient, webUIRoot, filepath.Join(t.TempDir(), "goroute.db"))
}

func testServerWithUsageAndConnectionAndWebUIAtPath(t *testing.T, getUsage func(context.Context, connection.Record) (providerregistry.UsageInfo, error), connectionClient protocolTestConnection, webUIRoot fs.FS, databasePath string) http.Handler {
	t.Helper()
	initialConnections := []connection.Record{{
		ID:          "codex-1",
		ProviderID:  "cx",
		Name:        "codex-user",
		AccessToken: "secret-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}}
	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	if err := repo.ReplaceConnections(initialConnections); err != nil {
		t.Fatalf("seed sqlite connections: %v", err)
	}

	logger := logging.NewWithWriter("prod", &bytes.Buffer{})
	providers, err := providerregistry.New(
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
			BuildConnection: func(connection.Record) (chatcompletion.ProtocolConnections, error) {
				return chatcompletion.ProtocolConnections{
					ChatCompletions: connectionClient,
					Responses:       connectionClient,
				}, nil
			},
			GetUsage: func(ctx context.Context, record connection.Record) (providerregistry.UsageInfo, error) {
				if getUsage != nil {
					return getUsage(ctx, record)
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
			GenerateOAuthURL: func(connection.Record) (string, error) {
				return "https://auth.openai.com/oauth/authorize?provider=cx", nil
			},
			StartOAuth: func(connection.Record) (providerregistry.OAuthSession, error) {
				return providerregistry.OAuthSession{
					AuthorizationURL: "https://auth.openai.com/oauth/authorize?provider=cx&flow=start",
					Pending: map[string]string{
						"state":         "test-state",
						"code_verifier": "test-verifier",
						"redirect_uri":  "http://localhost:1455/auth/callback",
					},
				}, nil
			},
			CompleteOAuth: func(_ connection.Record, pending map[string]string, callbackURL string) (providerregistry.OAuthResult, error) {
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
			ValidateConnection: func(record connection.Record) []string {
				if record.AccessToken == "" && record.APIKey == "" {
					return []string{"missing access_token or api_key"}
				}

				return nil
			},
		},
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "opena", Name: "OpenAI"},
			BuildConnection: func(connection.Record) (chatcompletion.ProtocolConnections, error) {
				return chatcompletion.ProtocolConnections{
					ChatCompletions: &testProvider{},
					Responses:       &testProvider{},
				}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	registry := chatcompletion.NewConnectionRegistryWithEntries(map[string][]chatcompletion.ConnectionEntry{
		"cx": {{
			ID:         "codex-1",
			Name:       "codex-user",
			ProviderID: "cx",
			ProtocolConnections: chatcompletion.ProtocolConnections{
				ChatCompletions: connectionClient,
				Responses:       connectionClient,
			},
		}},
	}, &logger)
	service := connectionsusecase.NewService(repo, testRuntime{repo: repo, providers: providers, registry: &registry}, providers, &logger)
	return NewServer(testCatalog(), &registry, service, repo, testAdminToken, webUIRoot, &logger)
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
	handler := testServer(t, &testProvider{response: openaiwire.ChatCompletionsResponse{ID: "chatcmpl-1", Object: "chat.completion", Model: "gpt-5.4", Choices: []openaiwire.ChatChoice{{Index: 0, Message: openaiwire.Message{Role: "assistant", Content: "hello back"}}}}})
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

func TestChatCompletionsAcceptsMixedTextAndImageContent(t *testing.T) {
	provider := &testProvider{response: openaiwire.ChatCompletionsResponse{
		ID:     "chatcmpl-1",
		Object: "chat.completion",
		Model:  "gpt-5.4",
		Choices: []openaiwire.ChatChoice{{
			Index:   0,
			Message: openaiwire.Message{Role: "assistant", Content: "looks good"},
		}},
	}}
	handler := testServer(t, provider)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":[{"type":"text","text":"describe this image"},{"type":"image_url","image_url":{"url":"https://example.com/cat.png","detail":"high"}}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if len(provider.lastReq.Messages) != 1 {
		t.Fatalf("expected one message, got %#v", provider.lastReq.Messages)
	}
	parts := provider.lastReq.Messages[0].Content.Parts()
	if len(parts) != 2 {
		t.Fatalf("expected two content parts, got %#v", parts)
	}
	if parts[0].Type != "text" || parts[0].Text != "describe this image" {
		t.Fatalf("unexpected text part %#v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil || parts[1].ImageURL.URL != "https://example.com/cat.png" || parts[1].ImageURL.Detail != "high" {
		t.Fatalf("unexpected image part %#v", parts[1])
	}
}

func TestChatCompletionsRejectsImageContentWithoutURL(t *testing.T) {
	handler := testServer(t, &testProvider{})
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"detail":"high"}}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `messages[0].content[0].image_url.url is required`) {
		t.Fatalf("expected validation error, got body=%s", rec.Body.String())
	}
}

func TestChatCompletionsPassesThroughUpstreamErrors(t *testing.T) {
	handler := testServer(t, &testProvider{err: chatcompletion.UpstreamError{
		StatusCode: http.StatusTooManyRequests,
		Message:    `{"error":{"message":"rate limited","type":"rate_limit_exceeded","code":"rate_limit_exceeded"}}`,
	}})
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d body=%s", http.StatusTooManyRequests, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected application/json content type, got %q", got)
	}
	if rec.Body.String() != `{"error":{"message":"rate limited","type":"rate_limit_exceeded","code":"rate_limit_exceeded"}}` {
		t.Fatalf("expected upstream body to pass through, got body=%s", rec.Body.String())
	}
}

func TestResponsesAcceptsPrefixedModel(t *testing.T) {
	handler := testServer(t, &testProvider{responsesResponse: openaiwire.ResponsesResponse{
		ID:        "resp_1",
		Object:    "response",
		CreatedAt: 1712345678,
		Model:     "gpt-5.4",
		Status:    openaiwire.ResponsesStatusCompleted,
		Output: []openaiwire.OutputItem{{
			ID:     "msg_1",
			Type:   openaiwire.OutputItemTypeMessage,
			Status: "completed",
			Phase:  "final_answer",
			Role:   string(openaiwire.ChatRoleAssistant),
			Content: []openaiwire.OutputContent{{
				Type:        openaiwire.OutputContentTypeOutputText,
				Text:        "hello back",
				Annotations: []any{},
				LogProbs:    []any{},
			}},
		}},
	}})
	body := []byte(`{"model":"cx/gpt-5.4","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"model":"cx/gpt-5.4"`) {
		t.Fatalf("expected response model to stay prefixed, got body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"completed"`) || !strings.Contains(rec.Body.String(), `"phase":"final_answer"`) || !strings.Contains(rec.Body.String(), `"annotations":[]`) || !strings.Contains(rec.Body.String(), `"logprobs":[]`) {
		t.Fatalf("expected response output fields to be preserved, got body=%s", rec.Body.String())
	}
}

func TestResponsesAcceptsStringInput(t *testing.T) {
	handler := testServer(t, &testProvider{responsesResponse: openaiwire.ResponsesResponse{
		ID:        "resp_1",
		Object:    "response",
		CreatedAt: 1712345678,
		Model:     "gpt-5.4",
		Status:    openaiwire.ResponsesStatusCompleted,
		Output: []openaiwire.OutputItem{{
			Type: openaiwire.OutputItemTypeMessage,
			Role: string(openaiwire.ChatRoleAssistant),
			Content: []openaiwire.OutputContent{{
				Type: openaiwire.OutputContentTypeOutputText,
				Text: "hello back",
			}},
		}},
	}})
	body := []byte(`{"model":"cx/gpt-5.4","input":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestResponsesStreamsSSE(t *testing.T) {
	handler := testServer(t, streamingTestProvider{testProvider: &testProvider{}, body: "data: {\"type\":\"response.created\"}\n\ndata: [DONE]\n\n"})
	body := []byte(`{"model":"cx/gpt-5.4","stream":true,"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", got)
	}
	if rec.Body.String() != "data: {\"type\":\"response.created\"}\n\ndata: [DONE]\n\n" {
		t.Fatalf("unexpected stream body %q", rec.Body.String())
	}
}

func TestResponsesRejectsInvalidJSON(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"cx/gpt-5.4",`)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"type":"invalid_request"`) {
		t.Fatalf("expected invalid request body=%s", rec.Body.String())
	}
}

func TestResponsesPersistsSyncLogs(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, &loggingTestProvider{
		testProvider: testProvider{
			responsesResponse: openaiwire.ResponsesResponse{
				ID:        "resp_1",
				Object:    "response",
				CreatedAt: 1712345678,
				Model:     "gpt-5.4",
				Status:    openaiwire.ResponsesStatusCompleted,
				Output: []openaiwire.OutputItem{{
					Type: openaiwire.OutputItemTypeMessage,
					Role: string(openaiwire.ChatRoleAssistant),
					Content: []openaiwire.OutputContent{{
						Type: openaiwire.OutputContentTypeOutputText,
						Text: "hello back",
					}},
				}},
				Usage: &openaiwire.ResponseUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
			},
		},
	}, nil, databasePath)
	body := []byte(`{"model":"cx/gpt-5.4","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	defer repo.Close()

	runs, err := repo.ListAIRequestRuns()
	if err != nil {
		t.Fatalf("list ai request runs: %v", err)
	}
	flows, err := repo.ListAIRequestFlows()
	if err != nil {
		t.Fatalf("list ai request flows: %v", err)
	}

	if len(runs) != 1 || runs[0].Type != chatcompletion.RequestTypeResponses || runs[0].TotalTokens != 15 {
		t.Fatalf("unexpected run records %#v", runs)
	}
	if len(flows) != 1 || flows[0].Type != chatcompletion.RequestTypeResponses {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if !strings.Contains(flows[0].TranslatedResponseBody, `"model":"cx/gpt-5.4"`) {
		t.Fatalf("unexpected flow records %#v", flows)
	}
}

func TestChatCompletionsPersistsSyncLogs(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, &loggingTestProvider{
		testProvider: testProvider{
			response: openaiwire.ChatCompletionsResponse{
				ID:     "chatcmpl-1",
				Object: "chat.completion",
				Model:  "gpt-5.4",
				Choices: []openaiwire.ChatChoice{{
					Index:   0,
					Message: openaiwire.Message{Role: "assistant", Content: "hello back"},
				}},
				Usage: &openaiwire.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			},
		},
	}, nil, databasePath)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	defer repo.Close()

	runs, err := repo.ListAIRequestRuns()
	if err != nil {
		t.Fatalf("list ai request runs: %v", err)
	}
	flows, err := repo.ListAIRequestFlows()
	if err != nil {
		t.Fatalf("list ai request flows: %v", err)
	}
	thirdPartyLogs, err := repo.ListThirdPartyRequestLogs()
	if err != nil {
		t.Fatalf("list third party request logs: %v", err)
	}

	if len(runs) != 1 || runs[0].RequestMode != chatcompletion.RequestModeSync || runs[0].TotalTokens != 15 {
		t.Fatalf("unexpected run records %#v", runs)
	}
	if runs[0].ProviderRequestMode != chatcompletion.RequestModeSync {
		t.Fatalf("unexpected run provider request mode %#v", runs)
	}
	if len(flows) != 1 {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if flows[0].ProviderRequestMode != chatcompletion.RequestModeSync {
		t.Fatalf("unexpected flow provider request mode %#v", flows)
	}
	if !strings.Contains(flows[0].RequestBody, `"role":"user"`) || !strings.Contains(flows[0].RequestBody, `"content":"hello"`) {
		t.Fatalf("expected raw request body, got %#v", flows[0])
	}
	if !strings.Contains(flows[0].TranslatedRequestBody, `"model":"gpt-5.4"`) {
		t.Fatalf("expected translated request body, got %#v", flows[0])
	}
	if !strings.Contains(flows[0].ResponseBody, `"id":"upstream-1"`) {
		t.Fatalf("expected raw response body, got %#v", flows[0])
	}
	if !strings.Contains(flows[0].TranslatedResponseBody, `"model":"cx/gpt-5.4"`) {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if len(thirdPartyLogs) != 1 || thirdPartyLogs[0].RequestMode != chatcompletion.RequestModeSync || thirdPartyLogs[0].ProviderRequestMode != chatcompletion.RequestModeSync {
		t.Fatalf("unexpected third party logs %#v", thirdPartyLogs)
	}
}

func TestChatCompletionsRedactsImageURLsInPersistedLogs(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, &loggingTestProvider{
		testProvider: testProvider{
			response: openaiwire.ChatCompletionsResponse{
				ID:     "chatcmpl-1",
				Object: "chat.completion",
				Model:  "gpt-5.4",
				Choices: []openaiwire.ChatChoice{{
					Index:   0,
					Message: openaiwire.Message{Role: "assistant", Content: "hello back"},
				}},
			},
		},
	}, nil, databasePath)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":[{"type":"text","text":"what is in this image?"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc123","detail":"high"}},{"type":"image_url","image_url":{"url":"https://signed.example.com/private.png","detail":"low"}}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	defer repo.Close()

	flows, err := repo.ListAIRequestFlows()
	if err != nil {
		t.Fatalf("list ai request flows: %v", err)
	}
	thirdPartyLogs, err := repo.ListThirdPartyRequestLogs()
	if err != nil {
		t.Fatalf("list third party request logs: %v", err)
	}

	if len(flows) != 1 {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if strings.Contains(flows[0].RequestBody, "data:image/png;base64,abc123") || strings.Contains(flows[0].RequestBody, "https://signed.example.com/private.png") {
		t.Fatalf("expected request body to redact image urls, got %#v", flows[0].RequestBody)
	}
	if !strings.Contains(flows[0].RequestBody, "[REDACTED_DATA_URL]") || !strings.Contains(flows[0].RequestBody, "[REDACTED_IMAGE_URL]") {
		t.Fatalf("expected request body redaction markers, got %#v", flows[0].RequestBody)
	}
	if strings.Contains(flows[0].TranslatedRequestBody, "data:image/png;base64,abc123") || strings.Contains(flows[0].TranslatedRequestBody, "https://signed.example.com/private.png") {
		t.Fatalf("expected translated request body to redact image urls, got %#v", flows[0].TranslatedRequestBody)
	}
	if len(thirdPartyLogs) != 1 {
		t.Fatalf("unexpected third party logs %#v", thirdPartyLogs)
	}
	if strings.Contains(thirdPartyLogs[0].RequestBody, "data:image/png;base64,abc123") || strings.Contains(thirdPartyLogs[0].RequestBody, "https://signed.example.com/private.png") {
		t.Fatalf("expected third party request body to redact image urls, got %#v", thirdPartyLogs[0].RequestBody)
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

func TestChatCompletionsPersistsStreamLogsWithReconstructedResponse(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, loggingStreamingTestProvider{testProvider: &testProvider{}, body: "data: {\"text\":\"first\"}\n\ndata: [DONE]\n\n"}, nil, databasePath)
	body := []byte(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	defer repo.Close()

	runs, err := repo.ListAIRequestRuns()
	if err != nil {
		t.Fatalf("list ai request runs: %v", err)
	}
	flows, err := repo.ListAIRequestFlows()
	if err != nil {
		t.Fatalf("list ai request flows: %v", err)
	}
	thirdPartyLogs, err := repo.ListThirdPartyRequestLogs()
	if err != nil {
		t.Fatalf("list third party request logs: %v", err)
	}

	if len(runs) != 1 || runs[0].RequestMode != chatcompletion.RequestModeStream {
		t.Fatalf("unexpected run records %#v", runs)
	}
	if runs[0].ProviderRequestMode != chatcompletion.RequestModeStream {
		t.Fatalf("unexpected run provider request mode %#v", runs)
	}
	if len(flows) != 1 {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if flows[0].ProviderRequestMode != chatcompletion.RequestModeStream {
		t.Fatalf("unexpected flow provider request mode %#v", flows)
	}
	if !strings.Contains(flows[0].ResponseBody, `"content":"first"`) {
		t.Fatalf("expected raw SSE response body, got %#v", flows[0])
	}
	if !strings.Contains(flows[0].TranslatedRequestBody, `"model":"gpt-5.4"`) || !strings.Contains(flows[0].TranslatedRequestBody, `"stream":true`) {
		t.Fatalf("expected translated request body, got %#v", flows[0])
	}
	if !strings.Contains(flows[0].TranslatedResponseBody, `"content":"first"`) {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if len(thirdPartyLogs) != 1 || thirdPartyLogs[0].ProviderRequestMode != chatcompletion.RequestModeStream || !strings.Contains(thirdPartyLogs[0].ResponseBody, `"content":"first"`) {
		t.Fatalf("unexpected third party logs %#v", thirdPartyLogs)
	}
}

func TestChatCompletionsPersistsMalformedRequestWithoutThirdPartyLogs(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	handler := testServerWithUsageAndConnectionAndWebUIAtPath(t, nil, &testProvider{}, nil, databasePath)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"cx/gpt-5.4",`)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	defer repo.Close()

	runs, err := repo.ListAIRequestRuns()
	if err != nil {
		t.Fatalf("list ai request runs: %v", err)
	}
	flows, err := repo.ListAIRequestFlows()
	if err != nil {
		t.Fatalf("list ai request flows: %v", err)
	}
	thirdPartyLogs, err := repo.ListThirdPartyRequestLogs()
	if err != nil {
		t.Fatalf("list third party request logs: %v", err)
	}

	if len(runs) != 1 || runs[0].ErrorType != "invalid_request" {
		t.Fatalf("unexpected run records %#v", runs)
	}
	if len(flows) != 1 || flows[0].ErrorType != "invalid_request" {
		t.Fatalf("unexpected flow records %#v", flows)
	}
	if flows[0].TranslatedRequestBody != "" {
		t.Fatalf("expected empty translated request body, got %#v", flows[0].TranslatedRequestBody)
	}
	if flows[0].TranslatedResponseBody != "" {
		t.Fatalf("expected empty translated response body, got %#v", flows[0].TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 0 {
		t.Fatalf("expected no third party logs, got %#v", thirdPartyLogs)
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
				Message: openaiwire.Message{Role: "assistant", Content: "weather ready"},
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

func TestDebugRequestsRouteIsNotRegistered(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodGet, "/debug/requests", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
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

func TestConnectionsDeleteAllowsLastConnection(t *testing.T) {
	handler := testServer(t, &testProvider{})
	req := httptest.NewRequest(http.MethodDelete, "/admin/api/connections/codex-1", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNoContent, rec.Code, rec.Body.String())
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
	handler := testServerWithUsage(t, func(context.Context, connection.Record) (providerregistry.UsageInfo, error) {
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
