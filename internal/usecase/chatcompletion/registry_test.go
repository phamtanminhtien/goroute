package chatcompletion

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/logging"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/rs/zerolog"
)

func TestConnectionRegistryDispatchesByTargetProviderID(t *testing.T) {
	codexConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "codex-response"}}
	openaiConnection := recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "openai-response"}}
	registry := newTestRegistry(map[string][]Connection{
		"cx":    {codexConnection},
		"opena": {openaiConnection},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "codex-response" {
		t.Fatalf("expected codex response, got %q", response.ID)
	}
}

func TestConnectionRegistryFallsBackAcrossConnectionsOfSameProvider(t *testing.T) {
	registry := newTestRegistry(map[string][]Connection{
		"cx": {
			recordingConnection{err: UpstreamError{StatusCode: 503, Message: "first failed"}},
			recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "second-response"}},
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
	registry := newTestRegistry(map[string][]Connection{
		"cx": {
			recordingConnection{err: ConnectionConfigurationError{ConnectionID: "codex-1", Message: "missing token"}},
			recordingConnection{
				response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
				onCall: func() {
					secondCalled = true
				},
			},
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
			Connection: recordingConnection{err: UpstreamError{StatusCode: 429, Message: "rate limited"}},
		}, {
			ID:         "codex-secondary",
			Name:       "Codex Secondary",
			ProviderID: "cx",
			Connection: recordingConnection{response: openaiwire.ChatCompletionsResponse{ID: "ok"}},
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

func newTestRegistry(connections map[string][]Connection) ConnectionRegistry {
	return NewConnectionRegistryWithEntries(wrapConnections(connections), loggerPtr(logging.NewWithWriter("prod", &bytes.Buffer{})))
}

func wrapConnections(connections map[string][]Connection) map[string][]ConnectionEntry {
	entries := make(map[string][]ConnectionEntry, len(connections))
	for providerID, configuredConnections := range connections {
		for i, connection := range configuredConnections {
			entries[providerID] = append(entries[providerID], ConnectionEntry{
				ID:         fmt.Sprintf("%s-%d", providerID, i+1),
				Name:       fmt.Sprintf("%s-%d", providerID, i+1),
				ProviderID: providerID,
				Connection: connection,
			})
		}
	}

	return entries
}

type recordingConnection struct {
	response openaiwire.ChatCompletionsResponse
	err      error
	onCall   func()
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

func loggerPtr(logger zerolog.Logger) *zerolog.Logger {
	return &logger
}
