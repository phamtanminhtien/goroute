package chatcompletion

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

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
	registry := newTestRegistry(map[string][]ConnectionEntry{
		"cx": {
			newConnectionEntry("cx", 1, recordingConnection{err: UpstreamError{StatusCode: 503, Message: "first failed"}}, nil),
			newConnectionEntry("cx", 2, recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "second-response"}}, nil),
		},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "second-response" {
		t.Fatalf("expected second connection response, got %q", response.ID)
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

func TestConnectionRegistryFallsBackWhenProtocolUnsupported(t *testing.T) {
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
	if response.ID != "chat" {
		t.Fatalf("expected fallback chat response, got %q", response.ID)
	}
}

func newTestRegistry(connections map[string][]ConnectionEntry) ConnectionRegistry {
	return NewConnectionRegistryWithEntries(connections, loggerPtr(logging.NewWithWriter("prod", &bytes.Buffer{})))
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
	err               error
	onCall            func()
}

func (c recordingConnection) ChatCompletions(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.err != nil {
		return openaiwire.ChatCompletionsResponse{}, c.err
	}

	return c.response, nil
}

func (c recordingConnection) ChatCompletionsStream(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (io.ReadCloser, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.err != nil {
		return nil, c.err
	}

	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func (c recordingConnection) Responses(context.Context, openaiwire.ResponsesRequest, routing.Target) (openaiwire.ResponsesResponse, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.err != nil {
		return openaiwire.ResponsesResponse{}, c.err
	}

	return c.responsesResponse, nil
}

func (c recordingConnection) ResponsesStream(context.Context, openaiwire.ResponsesRequest, routing.Target) (io.ReadCloser, error) {
	if c.onCall != nil {
		c.onCall()
	}
	if c.err != nil {
		return nil, c.err
	}

	return io.NopCloser(strings.NewReader("data: [DONE]\n\n")), nil
}

func loggerPtr(logger zerolog.Logger) *zerolog.Logger {
	return &logger
}
