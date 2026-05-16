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

func TestConnectionRegistryPersistsLifecycleUpdates(t *testing.T) {
	historyStore := &recordingHistoryStore{}
	registry := NewConnectionRegistryWithEntriesAndHistory(map[string][]ConnectionEntry{
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
	}, loggerPtr(logging.NewWithWriter("prod", &bytes.Buffer{})), historyStore)

	ctx := WithRequestID(context.Background(), "req-42")
	_, err := registry.ChatCompletions(ctx, openaiwire.ChatCompletionsRequest{
		Model:    "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "hello"}},
		Tools:    []openaiwire.Tool{{Type: "function", Function: openaiwire.ToolFunction{Name: "lookup"}}},
	}, routing.Target{
		Prefix:         "cx",
		RequestedModel: "gpt-5.4",
		ProviderID:     "cx",
		ProviderName:   "Codex",
	})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if historyStore.created.HistoryID == 0 {
		t.Fatalf("expected created history id, got %#v", historyStore.created)
	}
	if historyStore.created.Status != RequestStatusStarted {
		t.Fatalf("expected started status on create, got %#v", historyStore.created)
	}
	if historyStore.created.MessageCount != 1 || historyStore.created.ToolCount != 1 {
		t.Fatalf("expected request metadata on create, got %#v", historyStore.created)
	}
	if len(historyStore.updates) != 2 {
		t.Fatalf("expected two updates (retry + success), got %#v", historyStore.updates)
	}
	if historyStore.updates[0].Status != RequestStatusRetrying || historyStore.updates[0].AttemptCount != 1 {
		t.Fatalf("expected retrying update after first attempt, got %#v", historyStore.updates[0])
	}
	if historyStore.updates[0].LastErrorCategory != "upstream_retryable_error" {
		t.Fatalf("expected rate_limited on retry update, got %#v", historyStore.updates[0])
	}
	last := historyStore.updates[1]
	if last.Status != RequestStatusSuccess || last.FinalStatus != RequestStatusSuccess {
		t.Fatalf("expected success final update, got %#v", last)
	}
	if last.AttemptCount != 2 || last.LastConnectionID != "codex-secondary" {
		t.Fatalf("expected second attempt metadata, got %#v", last)
	}
	if len(last.Attempts) != 2 || !last.Attempts[0].WillFallback || last.Attempts[1].Outcome != "success" {
		t.Fatalf("unexpected attempts timeline: %#v", last.Attempts)
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

type recordingHistoryStore struct {
	created RequestAttemptHistory
	updates []RequestAttemptHistory
	nextID  int64
}

func (s *recordingHistoryStore) CreateRequestAttemptHistory(record RequestAttemptHistory) (RequestAttemptHistory, error) {
	s.nextID++
	record.HistoryID = s.nextID
	s.created = cloneHistoryRecord(record)
	return record, nil
}

func (s *recordingHistoryStore) UpdateRequestAttemptHistory(record RequestAttemptHistory) error {
	s.updates = append(s.updates, cloneHistoryRecord(record))
	return nil
}

func (s *recordingHistoryStore) RecentRequestAttempts(limit int) ([]RequestAttemptHistory, error) {
	return nil, nil
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
