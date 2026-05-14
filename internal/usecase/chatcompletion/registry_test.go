package chatcompletion

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestProviderRegistryDispatchesByTargetProviderType(t *testing.T) {
	codexProvider := recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "codex-response"}}
	openaiProvider := recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "openai-response"}}
	registry := newTestRegistry(map[string][]Provider{
		"codex":  {codexProvider},
		"openai": {openaiProvider},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderType: "codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "codex-response" {
		t.Fatalf("expected codex response, got %q", response.ID)
	}
}

func TestProviderRegistryFallsBackAcrossProvidersOfSameType(t *testing.T) {
	registry := newTestRegistry(map[string][]Provider{
		"codex": {
			recordingProvider{err: UpstreamError{StatusCode: 503, Message: "first failed"}},
			recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "second-response"}},
		},
	})

	response, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderType: "codex"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if response.ID != "second-response" {
		t.Fatalf("expected second provider response, got %q", response.ID)
	}
}

func TestProviderRegistryStopsFallbackOnTerminalErrors(t *testing.T) {
	secondCalled := false
	registry := newTestRegistry(map[string][]Provider{
		"codex": {
			recordingProvider{err: ProviderConfigurationError{ProviderID: "codex-1", Message: "missing token"}},
			recordingProvider{
				response: openaiwire.ChatCompletionsResponse{ID: "should-not-run"},
				onCall: func() {
					secondCalled = true
				},
			},
		},
	})

	_, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{Model: "cx/gpt-5.4"}, routing.Target{Prefix: "cx", RequestedModel: "gpt-5.4", ProviderType: "codex"})
	if err == nil {
		t.Fatal("expected ChatCompletions to return an error")
	}
	if secondCalled {
		t.Fatal("expected fallback to stop on terminal error")
	}
}

func TestProviderRegistryLogsAttemptsAndFinalCategory(t *testing.T) {
	var logs bytes.Buffer
	registry := NewProviderRegistryWithEntries(map[string][]ProviderEntry{
		"codex": {{
			ID:       "codex-primary",
			Name:     "Codex Primary",
			Provider: recordingProvider{err: UpstreamError{StatusCode: 429, Message: "rate limited"}},
		}, {
			ID:       "codex-secondary",
			Name:     "Codex Secondary",
			Provider: recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "ok"}},
		}},
	}, log.New(&logs, "", 0))

	_, err := registry.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{Model: "cx/gpt-5.4"}, routing.Target{
		Prefix:         "cx",
		RequestedModel: "gpt-5.4",
		ProviderType:   "codex",
	})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `provider_id="codex-primary"`) {
		t.Fatalf("expected logs to include first provider metadata, got %s", output)
	}
	if !strings.Contains(output, `outcome="retryable"`) {
		t.Fatalf("expected logs to include retryable outcome, got %s", output)
	}
	if !strings.Contains(output, `provider_id="codex-secondary"`) {
		t.Fatalf("expected logs to include second provider metadata, got %s", output)
	}
	if !strings.Contains(output, `outcome="success"`) {
		t.Fatalf("expected logs to include success outcome, got %s", output)
	}
}

func newTestRegistry(providers map[string][]Provider) ProviderRegistry {
	return NewProviderRegistryWithEntries(wrapProviders(providers), log.New(&bytes.Buffer{}, "", 0))
}

func wrapProviders(providers map[string][]Provider) map[string][]ProviderEntry {
	entries := make(map[string][]ProviderEntry, len(providers))
	for providerType, configuredProviders := range providers {
		for i, provider := range configuredProviders {
			entries[providerType] = append(entries[providerType], ProviderEntry{
				ID:       fmt.Sprintf("%s-%d", providerType, i+1),
				Name:     fmt.Sprintf("%s-%d", providerType, i+1),
				Provider: provider,
			})
		}
	}

	return entries
}

type recordingProvider struct {
	response openaiwire.ChatCompletionsResponse
	err      error
	onCall   func()
}

func (p recordingProvider) ChatCompletions(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if p.onCall != nil {
		p.onCall()
	}
	if p.err != nil {
		return openaiwire.ChatCompletionsResponse{}, p.err
	}

	return p.response, nil
}
