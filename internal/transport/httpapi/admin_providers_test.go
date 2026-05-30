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

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/modelcombo"
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
	if !strings.Contains(listRec.Body.String(), `"source":"system"`) || !strings.Contains(listRec.Body.String(), `"source":"custom"`) {
		t.Fatalf("expected system and custom model sources, got %s", listRec.Body.String())
	}
}

func TestProviderModelByIDHandlerUpdatesAndDeletesCustomModel(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/models", strings.NewReader(`{"id":"cx/gpt-5.5","name":"GPT-5.5"}`))
	createReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/admin/api/providers/cx/models/cx/gpt-5.5", strings.NewReader(`{"id":"cx/gpt-5.5-turbo","name":"GPT-5.5 Turbo","description":"Updated custom model"}`))
	updateReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	if !strings.Contains(updateRec.Body.String(), `"id":"cx/gpt-5.5-turbo"`) {
		t.Fatalf("expected updated model response, got %s", updateRec.Body.String())
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/admin/api/providers/cx/models/cx/gpt-5.5-turbo", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestProviderModelByIDHandlerRejectsSystemModelChanges(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/api/providers/cx/models/cx/gpt-5.4", nil)
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "custom model not found") {
		t.Fatalf("expected custom model error, got %s", rec.Body.String())
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

func TestModelCombosHandlerCreatesListsAndAdvertisesComboModel(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/model-combos", strings.NewReader(`{
		"alias":"combo/fast",
		"name":"Fast Combo",
		"description":"Try Codex first",
		"targets":[{"provider_id":"cx","model_id":"cx/gpt-5.4"},{"provider_id":"opena","model_id":"opena/gpt-4.1"}]
	}`))
	createReq.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `"alias":"combo/fast"`) {
		t.Fatalf("expected created combo response, got %s", createRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/model-combos", nil)
	listReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"alias":"combo/fast"`) {
		t.Fatalf("expected combo in list response, got %s", listRec.Body.String())
	}

	modelsRec := httptest.NewRecorder()
	modelsReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	handler.ServeHTTP(modelsRec, modelsReq)
	if modelsRec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, modelsRec.Code, modelsRec.Body.String())
	}
	if !strings.Contains(modelsRec.Body.String(), `"id":"combo/fast"`) || !strings.Contains(modelsRec.Body.String(), `"is_combo":"true"`) {
		t.Fatalf("expected combo in model list, got %s", modelsRec.Body.String())
	}
}

func TestModelCombosHandlerValidatesDuplicateAndUnknownTargets(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	create := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/api/model-combos", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testAdminToken)
		handler.ServeHTTP(rec, req)
		return rec
	}

	first := create(`{"alias":"fast","targets":[{"provider_id":"cx","model_id":"cx/gpt-5.4"}]}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected initial create, got %d body=%s", first.Code, first.Body.String())
	}

	duplicate := create(`{"alias":"fast","targets":[{"provider_id":"cx","model_id":"cx/gpt-5.4"}]}`)
	if duplicate.Code != http.StatusBadRequest || !strings.Contains(duplicate.Body.String(), "already exists") {
		t.Fatalf("expected duplicate error, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	unknownProvider := create(`{"alias":"bad-provider","targets":[{"provider_id":"missing","model_id":"missing/model"}]}`)
	if unknownProvider.Code != http.StatusBadRequest || !strings.Contains(unknownProvider.Body.String(), `provider \"missing\" not found`) {
		t.Fatalf("expected unknown provider error, got %d body=%s", unknownProvider.Code, unknownProvider.Body.String())
	}

	unknownModel := create(`{"alias":"bad-model","targets":[{"provider_id":"cx","model_id":"cx/missing"}]}`)
	if unknownModel.Code != http.StatusBadRequest || !strings.Contains(unknownModel.Body.String(), `model \"cx/missing\" does not belong`) {
		t.Fatalf("expected unknown model error, got %d body=%s", unknownModel.Code, unknownModel.Body.String())
	}
}

func TestModelComboByAliasHandlerUpdatesTargetOrderAndSupportsSlashAlias(t *testing.T) {
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{})

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/model-combos", strings.NewReader(`{
		"alias":"combo/fast",
		"targets":[{"provider_id":"cx","model_id":"cx/gpt-5.4"},{"provider_id":"opena","model_id":"opena/gpt-4.1"}]
	}`))
	createReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/admin/api/model-combos/combo/fast", strings.NewReader(`{
		"alias":"combo/fast",
		"name":"Fast Combo",
		"targets":[{"provider_id":"opena","model_id":"opena/gpt-4.1"},{"provider_id":"cx","model_id":"cx/gpt-5.4"}]
	}`))
	updateReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var combo modelcombo.Combo
	if err := json.Unmarshal(updateRec.Body.Bytes(), &combo); err != nil {
		t.Fatalf("decode combo: %v", err)
	}
	if len(combo.Targets) != 2 || combo.Targets[0].ProviderID != "opena" || combo.Targets[0].Priority != 0 {
		t.Fatalf("expected updated target order, got %#v", combo.Targets)
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
	if connectionClient.lastTarget.RequestedModel != "gpt-5.4" {
		t.Fatalf("expected upstream target model to be stripped, got %#v", connectionClient.lastTarget)
	}
	if len(connectionClient.lastReq.Messages) != 1 {
		t.Fatalf("expected one test message, got %#v", connectionClient.lastReq.Messages)
	}
}

func TestProviderModelTestHandlerUsesAnthropicMessagesForAnthropicProvider(t *testing.T) {
	connectionClient := &testProvider{
		anthropicResponse: anthropicwire.MessagesResponse{
			ID:         "msg_1",
			Type:       anthropicwire.MessageTypeMessage,
			Role:       anthropicwire.RoleAssistant,
			Model:      "claude-sonnet-4-5",
			Content:    []anthropicwire.ContentBlock{{Type: anthropicwire.ContentTypeText, Text: "OK"}},
			StopReason: anthropicwire.StopReasonEndTurn,
		},
	}
	handler := newProviderModelTestServer(t, newProviderModelTestServerInput{
		initialConnections: []connection.Record{{
			ID:         "anthropic-1",
			ProviderID: "anthropic",
			Name:       "anthropic-user",
			APIKey:     "secret-token",
		}},
		registryEntries: map[string][]chatcompletion.ConnectionEntry{
			"anthropic": {{
				ID:         "anthropic-1",
				Name:       "anthropic-user",
				ProviderID: "anthropic",
				ProtocolConnections: chatcompletion.ProtocolConnections{
					ChatCompletions: connectionClient,
					Responses:       connectionClient,
					Anthropic:       connectionClient,
				},
			}},
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/anthropic/test", strings.NewReader(`{"model":"anthropic/claude-sonnet-4-5"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if connectionClient.lastAnthropicReq.Model != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("expected anthropic test request, got %#v", connectionClient.lastAnthropicReq)
	}
	if connectionClient.lastAnthropicTarget.RequestedModel != "claude-sonnet-4-5" {
		t.Fatalf("expected upstream target model to be stripped, got %#v", connectionClient.lastAnthropicTarget)
	}
	if connectionClient.lastReq.Model != "" {
		t.Fatalf("expected chat completions not to be used, got %#v", connectionClient.lastReq)
	}
	if !strings.Contains(rec.Body.String(), `"output_text":"OK"`) {
		t.Fatalf("expected anthropic output text, got %s", rec.Body.String())
	}
}

func TestProviderModelTestHandlerStripsOnlyRoutePrefixForNestedModel(t *testing.T) {
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

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/models", strings.NewReader(`{"id":"cx/cx/gpt-5.5","name":"GPT-5.5 via CX"}`))
	createReq.Header.Set("Authorization", "Bearer "+testAdminToken)
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected model create, got %d body=%s", createRec.Code, createRec.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/providers/cx/test", strings.NewReader(`{"model":"cx/cx/gpt-5.5"}`))
	req.Header.Set("Authorization", "Bearer "+testAdminToken)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if connectionClient.lastReq.Model != "cx/cx/gpt-5.5" {
		t.Fatalf("expected client-facing request model to remain unchanged, got %#v", connectionClient.lastReq)
	}
	if connectionClient.lastTarget.RequestedModel != "cx/gpt-5.5" {
		t.Fatalf("expected upstream target model to strip only route prefix, got %#v", connectionClient.lastTarget)
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

	for index := range input.initialConnections {
		if !input.initialConnections[index].Enabled {
			input.initialConnections[index].Enabled = true
		}
	}
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
			BuildConnection: func(connection.Record, config.ProviderRuntimeSettings) (chatcompletion.ProtocolConnections, error) {
				return chatcompletion.ProtocolConnections{}, nil
			},
		},
		providerregistry.Registration{
			Descriptor: provider.Provider{ID: "opena", Name: "OpenAI"},
			BuildConnection: func(connection.Record, config.ProviderRuntimeSettings) (chatcompletion.ProtocolConnections, error) {
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
		testRuntime{repo: repo, providers: providers, registry: &registry, settingsManager: settingsManager},
		providers,
		&logger,
	)

	catalog := testCatalog()
	catalog.Providers = append(catalog.Providers, provider.Provider{
		ID:           "anthropic",
		Name:         "Anthropic",
		AuthType:     provider.AuthTypeAPIKey,
		Category:     "api_key",
		DefaultModel: "anthropic/claude-sonnet-4-5",
		Models:       []provider.Model{{ID: "anthropic/claude-sonnet-4-5", Name: "Claude Sonnet 4.5", InputPricePerMillionUSD: 3, OutputPricePerMillionUSD: 15}},
	})
	return NewServer(catalog, &registry, service, nil, repo, repo, repo, repo, settingsManager, testAdminToken, nil, &logger)
}
