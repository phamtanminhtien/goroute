package chatcompletion

import (
	"context"
	"fmt"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestProviderRegistryDispatchesByTargetProviderType(t *testing.T) {
	codexProvider := recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "codex-response"}}
	openaiProvider := recordingProvider{response: openaiwire.ChatCompletionsResponse{ID: "openai-response"}}
	registry := NewProviderRegistry(map[string][]Provider{
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
	registry := NewProviderRegistry(map[string][]Provider{
		"codex": {
			recordingProvider{err: fmt.Errorf("first failed")},
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

type recordingProvider struct {
	response openaiwire.ChatCompletionsResponse
	err      error
}

func (p recordingProvider) ChatCompletions(context.Context, openaiwire.ChatCompletionsRequest, routing.Target) (openaiwire.ChatCompletionsResponse, error) {
	if p.err != nil {
		return openaiwire.ChatCompletionsResponse{}, p.err
	}

	return p.response, nil
}
