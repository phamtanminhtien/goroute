package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/provider"
	"github.com/phamtanminhtien/goroute/internal/logging"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/providerregistry"
	"github.com/phamtanminhtien/goroute/internal/storage/gormsqlite"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
	connectionsusecase "github.com/phamtanminhtien/goroute/internal/usecase/connections"
)

func TestProviderModelTestHandlerRejectsUnknownProvider(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: []connection.Record{{
			ID:          "codex-1",
			ProviderID:  "cx",
			Name:        "codex-user",
			AccessToken: "secret-token",
		}},
		registryEntries: map[string][]chatcompletion.ConnectionEntry{
			"cx": {{
				ID:         "codex-1",
				Name:       "codex-user",
				ProviderID: "cx",
				ProtocolConnections: chatcompletion.ProtocolConnections{
					ChatCompletions: &testProvider{},
					Responses:       &testProvider{},
				},
			}},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/missing/test", strings.NewReader(`{"model":"cx/gpt-5.4"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"message":"provider not found"`) {
		t.Fatalf("expected provider not found error, got %s", rec.Body.String())
	}
}

func TestProviderModelsHandlerCreatesModelWhenPrefixMatches(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/models", strings.NewReader(`{"id":"cx/gpt-5.5","name":"GPT-5.5","description":"New test model"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"cx/gpt-5.5"`) {
		t.Fatalf("expected created model response, got %s", rec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/providers", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"id":"cx/gpt-5.5"`) {
		t.Fatalf("expected custom model in provider list, got %s", listRec.Body.String())
	}
}

func TestProviderModelsHandlerRejectsModelPrefixMismatch(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/models", strings.NewReader(`{"id":"openai/gpt-5.5","name":"GPT-5.5"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `model id must start with provider prefix \"cx/\"`) {
		t.Fatalf("expected prefix mismatch error, got %s", rec.Body.String())
	}
}

func TestProviderModelTestHandlerRejectsInvalidModelInput(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: []connection.Record{{
			ID:          "codex-1",
			ProviderID:  "cx",
			Name:        "codex-user",
			AccessToken: "secret-token",
		}},
		registryEntries: map[string][]chatcompletion.ConnectionEntry{
			"cx": {{
				ID:         "codex-1",
				Name:       "codex-user",
				ProviderID: "cx",
				ProtocolConnections: chatcompletion.ProtocolConnections{
					ChatCompletions: &testProvider{},
					Responses:       &testProvider{},
				},
			}},
		},
	})

	testCases := []struct {
		name           string
		body           string
		expectedSubstr string
	}{
		{
			name:           "missing model",
			body:           `{"model":"   "}`,
			expectedSubstr: `"message":"model is required"`,
		},
		{
			name:           "model outside provider",
			body:           `{"model":"opena/gpt-4.1"}`,
			expectedSubstr: `"message":"model does not belong to provider"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/test", strings.NewReader(testCase.body))
			req.Header.Set("Authorization", "Bearer "+testAdminToken)

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), testCase.expectedSubstr) {
				t.Fatalf("expected %q in %s", testCase.expectedSubstr, rec.Body.String())
			}
		})
	}
}

func TestProviderModelTestHandlerReturnsSuccessAndUsesChatCompletions(t *testing.T) {
	connectionClient := &testProvider{
		response: openaiwire.ChatCompletionsResponse{
			Choices: []openaiwire.ChatCompletionChoice{{
				Message: openaiwire.Message{
					Role:    openaiwire.ChatRoleAssistant,
					Content: "OK",
				},
			}},
		},
	}
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: []connection.Record{{
			ID:          "codex-1",
			ProviderID:  "cx",
			Name:        "codex-user",
			AccessToken: "secret-token",
		}},
		registryEntries: map[string][]chatcompletion.ConnectionEntry{
			"cx": {{
				ID:         "codex-1",
				Name:       "codex-user",
				ProviderID: "cx",
				ProtocolConnections: chatcompletion.ProtocolConnections{
					ChatCompletions: connectionClient,
					Responses:       connectionClient,
				},
			}},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/test", strings.NewReader(`{"model":"cx/gpt-5.4"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var response providerModelTestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "success" || response.OutputText != "OK" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if connectionClient.lastReq.Model != "cx/gpt-5.4" {
		t.Fatalf("expected test model request, got %#v", connectionClient.lastReq)
	}
	if len(connectionClient.lastReq.Messages) != 1 {
		t.Fatalf("expected one test message, got %#v", connectionClient.lastReq.Messages)
	}
}

func TestProviderModelTestHandlerReturnsClearErrorWhenNoExecutorConfigured(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: nil,
		registryEntries:    map[string][]chatcompletion.ConnectionEntry{},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/test", strings.NewReader(`{"model":"cx/gpt-5.4"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `no executor configured for provider \"cx\"`) {
		t.Fatalf("expected missing executor error, got %s", rec.Body.String())
	}
}

func TestProviderModelTestHandlerUsesFallbackChain(t *testing.T) {
	firstConnection := &testProvider{err: context.DeadlineExceeded}
	secondConnection := &testProvider{
		response: openaiwire.ChatCompletionsResponse{
			Choices: []openaiwire.ChatCompletionChoice{{
				Message: openaiwire.Message{
					Role:    openaiwire.ChatRoleAssistant,
					Content: "OK from fallback",
				},
			}},
		},
	}
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: []connection.Record{
			{
				ID:          "codex-1",
				ProviderID:  "cx",
				Name:        "primary",
				AccessToken: "secret-token-1",
			},
			{
				ID:          "codex-2",
				ProviderID:  "cx",
				Name:        "secondary",
				AccessToken: "secret-token-2",
			},
		},
		registryEntries: map[string][]chatcompletion.ConnectionEntry{
			"cx": {
				{
					ID:         "codex-1",
					Name:       "primary",
					ProviderID: "cx",
					ProtocolConnections: chatcompletion.ProtocolConnections{
						ChatCompletions: firstConnection,
						Responses:       firstConnection,
					},
				},
				{
					ID:         "codex-2",
					Name:       "secondary",
					ProviderID: "cx",
					ProtocolConnections: chatcompletion.ProtocolConnections{
						ChatCompletions: secondConnection,
						Responses:       secondConnection,
					},
				},
			},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/test", strings.NewReader(`{"model":"cx/gpt-5.4"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if firstConnection.lastReq.Model != "cx/gpt-5.4" {
		t.Fatalf("expected first connection to be attempted, got %#v", firstConnection.lastReq)
	}
	if secondConnection.lastReq.Model != "cx/gpt-5.4" {
		t.Fatalf("expected fallback connection to be attempted, got %#v", secondConnection.lastReq)
	}
	if !strings.Contains(rec.Body.String(), `"output_text":"OK from fallback"`) {
		t.Fatalf("expected fallback response, got %s", rec.Body.String())
	}
}

type newProviderModelTestServerInput struct {
	initialConnections []connection.Record
	registryEntries    map[string][]chatcompletion.ConnectionEntry
}

func newProviderModelTestServer(t *testing.T, input newProviderModelTestServerInput) http.Handler {
	t.Helper()

	databasePath := filepath.Join(t.TempDir(), "goroute.db")
	repo, err := gormsqlite.Open(databasePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	if err := repo.ReplaceConnections(input.initialConnections); err != nil {
		t.Fatalf("seed sqlite connections: %v", err)
	}

	cfg := testSettingsConfig()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.SavePath(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	settingsManager := config.NewSettingsManager(configPath, cfg)

	logger := logging.NewWithWriter("prod", &bytes.Buffer{})
	providers, err := providerregistry.New(
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "cx", Name: "Codex"},
			BuildConnection: func(connection.Record) (chatcompletion.ProtocolConnections, error) {
				return chatcompletion.ProtocolConnections{}, nil
			},
		},
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "opena", Name: "OpenAI"},
			BuildConnection: func(connection.Record) (chatcompletion.ProtocolConnections, error) {
				return chatcompletion.ProtocolConnections{}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}

	registry := chatcompletion.NewConnectionRegistryWithEntries(input.registryEntries, &logger)
	service := connectionsusecase.NewService(
		repo,
		testRuntime{repo: repo, providers: providers, registry: &registry},
		providers,
		&logger,
	)

	return NewServer(testCatalog(), &registry, service, repo, repo, settingsManager, testAdminToken, nil, &logger)
}
