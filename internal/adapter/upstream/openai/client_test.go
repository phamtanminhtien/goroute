package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientChatCompletionsPassesCommonOpenAIFields(t *testing.T) {
	temperature := 0.7
	maxTokens := 128
	toolSchema := json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`)

	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", got)
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &upstreamBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"id":"chatcmpl-1","object":"chat.completion","created":123,"model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})

	response, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:       "opena/gpt-4.1",
		Messages:    []openaiwire.ChatMessage{{Role: "user", Content: "hello"}},
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Tools: []openaiwire.Tool{{
			Type: "function",
			Function: openaiwire.ToolFunction{
				Name:       "lookup_weather",
				Parameters: toolSchema,
			},
		}},
		ToolChoice: map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "lookup_weather",
			},
		},
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}

	if upstreamBody["model"] != "gpt-4.1" {
		t.Fatalf("unexpected upstream model %#v", upstreamBody["model"])
	}
	if upstreamBody["temperature"] != temperature {
		t.Fatalf("unexpected temperature %#v", upstreamBody["temperature"])
	}
	if upstreamBody["max_tokens"] != float64(maxTokens) {
		t.Fatalf("unexpected max_tokens %#v", upstreamBody["max_tokens"])
	}
	if response.Created != 123 {
		t.Fatalf("unexpected created timestamp %d", response.Created)
	}
	if response.Choices[0].FinishReason != "stop" {
		t.Fatalf("unexpected finish reason %q", response.Choices[0].FinishReason)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage %#v", response.Usage)
	}
}

func TestClientStreamsOpenAIResponses(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("expected event stream accept header, got %q", got)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &upstreamBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("data: {\"id\":\"chunk_1\"}\n\n")),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})

	body, err := client.ChatCompletionsStream(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:    "opena/gpt-4.1",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "hello"}},
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("stream completions: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(data) != "data: {\"id\":\"chunk_1\"}\n\n" {
		t.Fatalf("unexpected stream body %q", data)
	}
	if upstreamBody["stream"] != true {
		t.Fatalf("expected stream request, got %#v", upstreamBody["stream"])
	}
}

func TestClientRequiresCredential(t *testing.T) {
	client := NewClient(nil, connection.Record{ProviderID: "openai", Name: "openai-user"})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err == nil {
		t.Fatal("expected credential error")
	}
	var configErr chatcompletion.ConnectionConfigurationError
	if !errors.As(err, &configErr) {
		t.Fatalf("expected connection configuration error, got %v", err)
	}
}
