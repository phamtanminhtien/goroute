package chatcompletion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/logging"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/rs/zerolog"
)

func TestConnectionRegistryDispatchesChatCompletionsByTargetProviderID(t *testing.T) {
	codexConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "codex-response"}}
	openaiConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "openai-response"}}
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx":    {newConnectionEntry("cx", 1, codexConnection, codexConnection)},
		"opena": {newConnectionEntry("opena", 1, openaiConnection, openaiConnection)},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "codex-response" {
		t.Fatalf("expected codex response, got %q", response.ID)
	}
}

func TestConnectionRegistryFallsBackAcrossChatCompletionsConnections(t *testing.T) {
	attempts := make([]int, 0, 2)
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, recordingConnection{
				err:           UpstreamError{StatusCode: 503, Message: "first failed"},
				onCallContext: func(ctx context.Context) { attempts = append(attempts, AttemptIndex(ctx)) },
			}, nil),
			newConnectionEntry("cx", 2, recordingConnection{
				response:      openaiwire.ChatCompletionsResponse{ID: "second-response"},
				onCallContext: func(ctx context.Context) { attempts = append(attempts, AttemptIndex(ctx)) },
			}, nil),
		},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "second-response" {
		t.Fatalf("expected second connection response, got %q", response.ID)
	}
	if len(attempts) != 2 || attempts[0] != 0 || attempts[1] != 1 {
		t.Fatalf("expected attempt indexes [0 1], got %#v", attempts)
	}
}

func TestConnectionRegistryFallsBackAcrossComboTargets(t *testing.T) {
	firstConnection := recordingConnection{err: UpstreamError{StatusCode: 503, Message: "first failed"}}
	secondConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "second-response"}}
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx":    {newConnectionEntry("cx", 1, firstConnection, nil)},
		"opena": {newConnectionEntry("opena", 1, secondConnection, nil)},
	})

	response, err := registry.ChatCompletionsTargets(context.Background(), openaiwire.ChatCompletionsRequest{Model: "combo/fast"}, []routing.Target{
		{Prefix: "cx", RequestedModel: "gpt-5.4", ProviderID: "cx", ProviderName: "Codex"},
		{Prefix: "opena", RequestedModel: "gpt-4.1", ProviderID: "opena", ProviderName: "OpenAI"},
	})
	if err != nil {
		t.Fatalf("ChatCompletionsTargets returned error: %v", err)
	}

	if response.ID != "second-response" {
		t.Fatalf("expected second target response, got %q", response.ID)
	}
}

func TestConnectionRegistryUsesSpecificTargetConnection(t *testing.T) {
	firstCalled := false
	secondConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "second-response"}}
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			{
				ID:         "cx-1",
				Name:       "cx-1",
				ProviderID: "cx",
				ProtocolConnections: ProtocolConnections{
					ChatCompletions: recordingConnection{
						response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
						onCall: func() {
							firstCalled = true
						},
					},
				},
			},
			{
				ID:         "cx-2",
				Name:       "cx-2",
				ProviderID: "cx",
				ProtocolConnections: ProtocolConnections{
					ChatCompletions: secondConnection,
				},
			},
		},
	})

	response, err := registry.ChatCompletionsTargets(context.Background(), openaiwire.ChatCompletionsRequest{Model: "combo/fast"}, []routing.Target{
		{Prefix: "cx", RequestedModel: "gpt-5.4", ProviderID: "cx", ProviderName: "Codex", ConnectionID: "cx-2"},
	})
	if err != nil {
		t.Fatalf("ChatCompletionsTargets returned error: %v", err)
	}
	if firstCalled {
		t.Fatal("expected unrelated connection not to be called")
	}
	if response.ID != "second-response" {
		t.Fatalf("expected selected connection response, got %q", response.ID)
	}
}

func TestConnectionRegistryStopsComboFallbackOnTerminalErrors(t *testing.T) {
	secondCalled := false
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {newConnectionEntry("cx", 1, recordingConnection{err: UpstreamError{StatusCode: 401, Message: "unauthorized"}}, nil)},
		"opena": {newConnectionEntry("opena", 1, recordingConnection{
			response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
			onCall: func() {
				secondCalled = true
			},
		}, nil)},
	})

	_, err := registry.ChatCompletionsTargets(context.Background(), openaiwire.ChatCompletionsRequest{Model: "combo/fast"}, []routing.Target{
		{Prefix: "cx", RequestedModel: "gpt-5.4", ProviderID: "cx", ProviderName: "Codex"},
		{Prefix: "opena", RequestedModel: "gpt-4.1", ProviderID: "opena", ProviderName: "OpenAI"},
	})
	if err == nil {
		t.Fatal("expected terminal error")
	}
	if secondCalled {
		t.Fatal("expected fallback to stop on terminal error")
	}
}

func TestConnectionRegistryStopsFallbackOnTerminalErrors(t *testing.T) {
	secondCalled := false
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, recordingConnection{err: ConnectionConfigurationError{ConnectionID: "codex-1", Message: "missing token"}}, nil),
			newConnectionEntry("cx", 2, recordingConnection{
				response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
				onCall: func() {
					secondCalled = true
				},
			}, nil),
		},
	})

	_, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{Model: "cx/gpt-5.4"}, routing.Target{Prefix: "cx", RequestedModel: "gpt-5.4", ProviderID: "cx", ProviderName: "Codex"})
	if err == nil {
		t.Fatal("expected ChatCompletions to return an error")
	}
	if secondCalled {
		t.Fatal("expected fallback to stop on terminal error")
	}
}

func TestConnectionRegistryLogsAttemptsAndFinalCategory(t *testing.T) {
	var logs bytes.Buffer
	registry := NewConnectionRegistryWithEntries(map[string][]ConnectionEntry{
		"cx": {{
			ID:         "codex-primary",
			Name:       "Codex Primary",
			ProviderID: "cx",
			ProtocolConnections: ProtocolConnections{
				ChatCompletions: recordingConnection{err: UpstreamError{StatusCode: 429, Message: "rate limited"}},
			},
		}, {
			ID:         "codex-secondary",
			Name:       "Codex Secondary",
			ProviderID: "cx",
			ProtocolConnections: ProtocolConnections{
				ChatCompletions: recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "ok"}},
			},
		}},
	}, loggerPtr(logging.NewWithWriter("prod", &logs)))

	ctx := WithRequestID(context.Background(), "req-1")
	_, err := registry.ChatCompletions(ctx, openaiwire.ChatCompletionsRequest{Model: "cx/gpt-5.4"}, routing.Target{
		Prefix:         "cx",
		RequestedModel: "gpt-5.4",
		ProviderID:     "cx",
		ProviderName:   "Codex",
	})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `"provider_id":"cx"`) {
		t.Fatalf("expected logs to include provider metadata, got %s", output)
	}
	if !strings.Contains(output, `"connection_id":"codex-secondary"`) {
		t.Fatalf("expected logs to include second connection metadata, got %s", output)
	}
	if !strings.Contains(output, `"outcome":"success"`) {
		t.Fatalf("expected logs to include success outcome, got %s", output)
	}
	if !strings.Contains(output, `"request_id":"req-1"`) {
		t.Fatalf("expected logs to include request_id, got %s", output)
	}
}

func TestConnectionRegistryDispatchesResponsesByTargetProviderID(t *testing.T) {
	codexConnection := recordingConnection{responsesResponse: openaiwire.ResponsesResponse{ID: "resp_cx"}}
	openaiConnection := recordingConnection{responsesResponse: openaiwire.ResponsesResponse{ID: "resp_openai"}}
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx":    {newConnectionEntry("cx", 1, nil, codexConnection)},
		"opena": {newConnectionEntry("opena", 1, nil, openaiConnection)},
	})

	response, err := registry.Responses(context.Background(), openaiwire.ResponsesRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}

	if response.ID != "resp_cx" {
		t.Fatalf("expected codex response, got %q", response.ID)
	}
}

func TestConnectionRegistryResponsesOnlyUsesResponsesCapability(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {newConnectionEntry("cx", 1, nil, recordingConnection{responsesResponse: openaiwire.ResponsesResponse{ID: "resp"}})},
	})

	response, err := registry.Responses(context.Background(), openaiwire.ResponsesRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}
	if response.ID != "resp" {
		t.Fatalf("unexpected response id %q", response.ID)
	}
}

func TestConnectionRegistryBridgesChatCompletionsThroughResponsesCapability(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, nil, recordingConnection{responsesResponse: openaiwire.ResponsesResponse{ID: "resp"}}),
			newConnectionEntry("cx", 2, recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "chat"}}, nil),
		},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if response.ID != "chatcmpl-resp" {
		t.Fatalf("expected bridged responses result, got %q", response.ID)
	}
}

func TestConnectionRegistryBridgesResponsesThroughChatCompletionsCapability(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"opena": {newConnectionEntry("opena", 1, recordingConnection{
			response: openaiwire.ChatCompletionsResponse{
				ID:      "chat",
				Object:  "chat.completion",
				Created: 123,
				Model:   "gpt-4.1",
				Choices: []openaiwire.ChatChoice{{
					Message: openaiwire.Message{Role: openaiwire.ChatRoleAssistant, Content: "hello"},
				}},
			},
		}, nil)},
	})

	response, err := registry.Responses(context.Background(), openaiwire.ResponsesRequest{
		Model:     "opena/gpt-4.1",
		InputText: "hi",
	}, routing.Target{ProviderID: "opena", ProviderName: "OpenAI"})
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}
	if response.ID != "chat" || response.TextValue() != "hello" {
		t.Fatalf("expected bridged chat response, got %#v", response)
	}
}

func TestConnectionRegistryLogsChatToResponsesBridgeBeforeAndAfterTranslate(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {newConnectionEntry("cx", 1, nil, bridgeLoggingResponsesConnection{})},
	})
	recorder := NewFlowRecorder("req-bridge", time.Unix(0, 0).UTC())
	target := routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.4"}
	recorder.SetResolvedTarget(target)
	ctx := WithFlowRecorder(context.Background(), recorder)

	response, err := registry.ChatCompletions(ctx, openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{{
			Role:    openaiwire.ChatRoleUser,
			Content: openaiwire.TextContent("hello"),
		}},
	}, target)
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	recorder.SetFlowResponse(response, true)

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if !strings.Contains(flow.TranslatedRequestBody, `"input"`) || strings.Contains(flow.TranslatedRequestBody, `"messages"`) {
		t.Fatalf("expected translated request to be responses-shaped upstream body, got %q", flow.TranslatedRequestBody)
	}
	if !stringPointerContains(flow.ResponseBody, `"object":"response"`) {
		t.Fatalf("expected raw upstream response body before client translation, got %#v", flow.ResponseBody)
	}
	if !stringPointerContains(flow.TranslatedResponseBody, `"object":"chat.completion"`) || !stringPointerContains(flow.TranslatedResponseBody, `"model":"gpt-5.4"`) {
		t.Fatalf("expected translated response to be client-facing chat body, got %#v", flow.TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 1 || !strings.Contains(thirdPartyLogs[0].RequestBody, `"input"`) || !stringPointerContains(thirdPartyLogs[0].ResponseBody, `"object":"response"`) {
		t.Fatalf("unexpected third-party bridge logs %#v", thirdPartyLogs)
	}
}

func TestConnectionRegistryLogsResponsesToChatBridgeBeforeAndAfterTranslate(t *testing.T) {
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"opena": {newConnectionEntry("opena", 1, bridgeLoggingChatConnection{}, nil)},
	})
	recorder := NewFlowRecorder("req-bridge", time.Unix(0, 0).UTC())
	target := routing.Target{ProviderID: "opena", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"}
	recorder.SetRequestType(RequestTypeResponses)
	recorder.SetResolvedTarget(target)
	ctx := WithFlowRecorder(context.Background(), recorder)

	response, err := registry.Responses(ctx, openaiwire.ResponsesRequest{
		Model:     "opena/gpt-4.1",
		InputText: "hello",
	}, target)
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}
	recorder.SetResponsesResponse(response, true)

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if !strings.Contains(flow.TranslatedRequestBody, `"messages"`) || strings.Contains(flow.TranslatedRequestBody, `"input"`) {
		t.Fatalf("expected translated request to be chat-shaped upstream body, got %q", flow.TranslatedRequestBody)
	}
	if !stringPointerContains(flow.ResponseBody, `"object":"chat.completion"`) {
		t.Fatalf("expected raw upstream response body before client translation, got %#v", flow.ResponseBody)
	}
	if !stringPointerContains(flow.TranslatedResponseBody, `"object":"response"`) || !stringPointerContains(flow.TranslatedResponseBody, `"model":"gpt-4.1"`) {
		t.Fatalf("expected translated response to be client-facing responses body, got %#v", flow.TranslatedResponseBody)
	}
	if len(thirdPartyLogs) != 1 || !strings.Contains(thirdPartyLogs[0].RequestBody, `"messages"`) || !stringPointerContains(thirdPartyLogs[0].ResponseBody, `"object":"chat.completion"`) {
		t.Fatalf("unexpected third-party bridge logs %#v", thirdPartyLogs)
	}
}

func TestConnectionRegistryRecordsRetryableErrorCooldown(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	previousTimeNow := registryTimeNow
	registryTimeNow = func() time.Time { return now }
	t.Cleanup(func() { registryTimeNow = previousTimeNow })

	store := &recordingRuntimeStateStore{}
	registry := newTestRegistryWithStateStore(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, recordingConnection{err: UpstreamError{StatusCode: 503, Message: "first failed"}}, nil),
			newConnectionEntry("cx", 2, recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "ok"}}, nil),
		},
	}, store)

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if response.ID != "ok" {
		t.Fatalf("expected fallback response, got %q", response.ID)
	}
	if store.recordedID != "cx-1" || store.recordedCategory != "upstream_server_error" {
		t.Fatalf("expected retryable error to be recorded, got %#v", store)
	}
	if store.recordedAt != now.Unix() {
		t.Fatalf("expected recordedAt %d, got %d", now.Unix(), store.recordedAt)
	}
	wantRetryAfter := now.Add(RetryableConnectionCooldown).Unix()
	if store.recordedRetryAfter != wantRetryAfter {
		t.Fatalf("expected retryAfter %d, got %d", wantRetryAfter, store.recordedRetryAfter)
	}
}

func TestConnectionRegistryDoesNotCooldownUpstream4xx(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	previousTimeNow := registryTimeNow
	registryTimeNow = func() time.Time { return now }
	t.Cleanup(func() { registryTimeNow = previousTimeNow })

	store := &recordingRuntimeStateStore{}
	registry := newTestRegistryWithStateStore(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, recordingConnection{err: UpstreamError{StatusCode: 429, Message: "slow down"}}, nil),
			newConnectionEntry("cx", 2, recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "ok"}}, nil),
		},
	}, store)

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if response.ID != "ok" {
		t.Fatalf("expected fallback response, got %q", response.ID)
	}
	if store.recordedID != "cx-1" || store.recordedCategory != "upstream_retryable_error" {
		t.Fatalf("expected 4xx retryable error to be recorded, got %#v", store)
	}
	if store.recordedRetryAfter != 0 {
		t.Fatalf("expected upstream 4xx without cooldown, got retryAfter=%d", store.recordedRetryAfter)
	}
}

func TestConnectionRegistrySkipsActiveCooldownConnection(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	previousTimeNow := registryTimeNow
	registryTimeNow = func() time.Time { return now }
	t.Cleanup(func() { registryTimeNow = previousTimeNow })

	firstCalled := false
	secondCalled := false
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			{
				ID:         "cx-1",
				Name:       "cx-1",
				ProviderID: "cx",
				RetryAfter: now.Add(time.Minute).Unix(),
				ProtocolConnections: ProtocolConnections{
					ChatCompletions: recordingConnection{
						response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
						onCall:   func() { firstCalled = true },
					},
				},
			},
			{
				ID:         "cx-2",
				Name:       "cx-2",
				ProviderID: "cx",
				ProtocolConnections: ProtocolConnections{
					ChatCompletions: recordingConnection{
						response: openaiwire.ChatCompletionsResponse{ID: "second-response"},
						onCall:   func() { secondCalled = true },
					},
				},
			},
		},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if firstCalled {
		t.Fatal("expected cooling down connection not to be invoked")
	}
	if !secondCalled || response.ID != "second-response" {
		t.Fatalf("expected second connection response, called=%t response=%q", secondCalled, response.ID)
	}
}

func TestConnectionRegistryAllowsExpiredCooldownConnection(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	previousTimeNow := registryTimeNow
	registryTimeNow = func() time.Time { return now }
	t.Cleanup(func() { registryTimeNow = previousTimeNow })

	called := false
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {{
			ID:         "cx-1",
			Name:       "cx-1",
			ProviderID: "cx",
			RetryAfter: now.Add(-time.Second).Unix(),
			ProtocolConnections: ProtocolConnections{
				ChatCompletions: recordingConnection{
					response: openaiwire.ChatCompletionsResponse{ID: "ok"},
					onCall:   func() { called = true },
				},
			},
		}},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if !called || response.ID != "ok" {
		t.Fatalf("expected expired cooldown connection to run, called=%t response=%q", called, response.ID)
	}
}

func TestConnectionRegistryClearsRuntimeErrorOnSuccess(t *testing.T) {
	store := &recordingRuntimeStateStore{}
	registry := newTestRegistryWithStateStore(map[string][]ConnectionEntry{
		"cx": {{
			ID:                "cx-1",
			Name:              "cx-1",
			ProviderID:        "cx",
			LastErrorMessage:  "previous failure",
			LastErrorCategory: "upstream_server_error",
			LastErrorAt:       100,
			RetryAfter:        101,
			ProtocolConnections: ProtocolConnections{
				ChatCompletions: recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "ok"}},
			},
		}},
	}, store)

	_, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if store.clearedID != "cx-1" {
		t.Fatalf("expected runtime state to be cleared, got %#v", store)
	}
}

func TestConnectionRegistryRecordsTerminalErrorWithoutCooldown(t *testing.T) {
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	previousTimeNow := registryTimeNow
	registryTimeNow = func() time.Time { return now }
	t.Cleanup(func() { registryTimeNow = previousTimeNow })

	store := &recordingRuntimeStateStore{}
	registry := newTestRegistryWithStateStore(map[string][]ConnectionEntry{
		"cx": {newConnectionEntry("cx", 1, recordingConnection{err: UpstreamError{StatusCode: 401, Message: "unauthorized"}}, nil)},
	}, store)

	_, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err == nil {
		t.Fatal("expected terminal error")
	}
	if store.recordedCategory != "upstream_auth_error" {
		t.Fatalf("expected auth error to be recorded, got %#v", store)
	}
	if store.recordedRetryAfter != 0 {
		t.Fatalf("expected terminal error without cooldown, got retryAfter=%d", store.recordedRetryAfter)
	}
}

func newTestRegistry(connections map[string][]ConnectionEntry) ConnectionRegistry {
	return NewConnectionRegistryWithEntries(connections, loggerPtr(logging.NewWithWriter("prod", &bytes.Buffer{})))
}

func newTestRegistryWithStateStore(connections map[string][]ConnectionEntry, stateStore ConnectionRuntimeStateStore) ConnectionRegistry {
	return NewConnectionRegistryWithStateStore(connections, loggerPtr(logging.NewWithWriter("prod", &bytes.Buffer{})), stateStore)
}

func newConnectionEntry(providerID string, index int, chat ChatCompletionsConnection, responses ResponsesConnection) ConnectionEntry {
	return ConnectionEntry{
		ID:         fmt.Sprintf("%s-%d", providerID, index),
		Name:       fmt.Sprintf("%s-%d", providerID, index),
		ProviderID: providerID,
		ProtocolConnections: ProtocolConnections{
			ChatCompletions: chat,
			Responses:       responses,
		},
	}
}

type recordingConnection struct {
	response          openaiwire.ChatCompletionsResponse
	responsesResponse openaiwire.ResponsesResponse
	streamBody        string
	responsesStream   string
	err               error
	onCall            func()
	onCallContext     func(context.Context)
}

type bridgeLoggingResponsesConnection struct{}

func (c bridgeLoggingResponsesConnection) Responses(ctx context.Context, req openaiwire.ResponsesRequest, target routing.Target) (openaiwire.ResponsesResponse, error) {
	response := openaiwire.ResponsesResponse{
		ID:        "resp_bridge",
		Object:    "response",
		CreatedAt: 123,
		Status:    openaiwire.ResponsesStatusCompleted,
		Model:     target.RequestedModel,
		Output: []openaiwire.OutputItem{{
			Type: openaiwire.OutputItemTypeMessage,
			Role: string(openaiwire.ChatRoleAssistant),
			Content: []openaiwire.OutputContent{{
				Type: openaiwire.OutputContentTypeOutputText,
				Text: "hello from responses",
			}},
		}},
	}
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		upstreamReq := req
		upstreamReq.Model = target.RequestedModel
		requestPayload, _ := json.Marshal(upstreamReq)
		responsePayload, _ := json.Marshal(response)
		recorder.SetProviderRequestMode(false)
		recorder.SetTranslatedRequestBody(string(requestPayload))
		recorder.AddThirdPartyLog(ThirdPartyLog{
			ProviderID:          target.ProviderID,
			ProviderName:        target.ProviderName,
			ConnectionID:        "cx-1",
			ConnectionName:      "cx-1",
			AttemptIndex:        AttemptIndex(ctx),
			ProviderRequestMode: RequestModeSync,
			RequestMethod:       "POST",
			RequestURL:          "https://provider.example/v1/responses",
			RequestBody:         string(requestPayload),
			ResponseStatusCode:  200,
			ResponseHeaders:     `{"Content-Type":["application/json"]}`,
			ResponseBody:        string(responsePayload),
			StartedAt:           time.Now().UTC(),
			CompletedAt:         time.Now().UTC(),
		})
	}
	return response, nil
}

func (c bridgeLoggingResponsesConnection) ResponsesStream(context.Context, openaiwire.ResponsesRequest, routing.Target) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

type bridgeLoggingChatConnection struct{}

func (c bridgeLoggingChatConnection) ChatCompletions(ctx context.Context, req openaiwire.ChatCompletionsRequest, target routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	response := openaiwire.ChatCompletionsResponse{
		ID:      "chat_bridge",
		Object:  "chat.completion",
		Created: 123,
		Model:   target.RequestedModel,
		Choices: []openaiwire.ChatChoice{{
			Index:        0,
			Message:      openaiwire.Message{Role: openaiwire.ChatRoleAssistant, Content: "hello from chat"},
			FinishReason: openaiwire.FinishReasonStop,
		}},
	}
	if recorder := FlowRecorderFromContext(ctx); recorder != nil {
		upstreamReq := req
		upstreamReq.Model = target.RequestedModel
		requestPayload, _ := json.Marshal(upstreamReq)
		responsePayload, _ := json.Marshal(response)
		recorder.SetProviderRequestMode(false)
		recorder.SetTranslatedRequestBody(string(requestPayload))
		recorder.AddThirdPartyLog(ThirdPartyLog{
			ProviderID:          target.ProviderID,
			ProviderName:        target.ProviderName,
			ConnectionID:        "opena-1",
			ConnectionName:      "opena-1",
			AttemptIndex:        AttemptIndex(ctx),
			ProviderRequestMode: RequestModeSync,
			RequestMethod:       "POST",
			RequestURL:          "https://provider.example/v1/chat/completions",
			RequestBody:         string(requestPayload),
			ResponseStatusCode:  200,
			ResponseHeaders:     `{"Content-Type":["application/json"]}`,
			ResponseBody:        string(responsePayload),
			StartedAt:           time.Now().UTC(),
			CompletedAt:         time.Now().UTC(),
		})
	}
	return response, nil
}

func (c bridgeLoggingChatConnection) ChatCompletionsStream(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func (c recordingConnection) ChatCompletions(ctx context.Context, _ openaiwire.ChatCompletionsRequest, _ routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.onCallContext != nil {
		c.onCallContext(ctx)
	}
	if c.err != nil {
		return openaiwire.ChatCompletionsResponse{}, c.err
	}

	return c.response, nil
}

func (c recordingConnection) ChatCompletionsStream(ctx context.Context, _ openaiwire.ChatCompletionsRequest, _ routing.Target) (io.ReadCloser, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.onCallContext != nil {
		c.onCallContext(ctx)
	}
	if c.err != nil {
		return nil, c.err
	}

	if c.streamBody != "" {
		return io.NopCloser(strings.NewReader(c.streamBody)), nil
	}

	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func (c recordingConnection) Responses(ctx context.Context, _ openaiwire.ResponsesRequest, _ routing.Target) (openaiwire.ResponsesResponse, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.onCallContext != nil {
		c.onCallContext(ctx)
	}
	if c.err != nil {
		return openaiwire.ResponsesResponse{}, c.err
	}

	return c.responsesResponse, nil
}

func (c recordingConnection) ResponsesStream(ctx context.Context, _ openaiwire.ResponsesRequest, _ routing.Target) (io.ReadCloser, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.onCallContext != nil {
		c.onCallContext(ctx)
	}
	if c.err != nil {
		return nil, c.err
	}

	if c.responsesStream != "" {
		return io.NopCloser(strings.NewReader(c.responsesStream)), nil
	}

	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func loggerPtr(logger zerolog.Logger) *zerolog.Logger {
	return &logger
}

type recordingRuntimeStateStore struct {
	recordedID         string
	recordedMessage    string
	recordedCategory   string
	recordedAt         int64
	recordedRetryAfter int64
	clearedID          string
}

func (s *recordingRuntimeStateStore) RecordConnectionRuntimeError(id string, message string, category string, lastErrorAt int64, retryAfter int64) error {
	s.recordedID = id
	s.recordedMessage = message
	s.recordedCategory = category
	s.recordedAt = lastErrorAt
	s.recordedRetryAfter = retryAfter
	return nil
}

func (s *recordingRuntimeStateStore) ClearConnectionRuntimeError(id string) error {
	s.clearedID = id
	return nil
}
