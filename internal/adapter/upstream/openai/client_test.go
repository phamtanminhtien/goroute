package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
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
		Messages:    []openaiwire.ChatMessage{{Role: "user", Content: openaiwire.TextContent("hello")}},
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
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: openaiwire.TextContent("hello")}},
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

func TestClientChatCompletionsPassesThroughImageContent(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
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
				`{"id":"chatcmpl-1","object":"chat.completion","created":123,"model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`,
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})
	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model: "opena/gpt-4.1",
		Messages: []openaiwire.ChatMessage{{
			Role: "user",
			Content: openaiwire.PartsContent(
				openaiwire.ChatMessageContentPart{Type: "text", Text: "describe this"},
				openaiwire.ChatMessageContentPart{
					Type: "image_url",
					ImageURL: &openaiwire.ChatMessageImageURL{
						URL:    "https://example.com/cat.png",
						Detail: "high",
					},
				},
			),
		}},
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}

	messages, ok := upstreamBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages payload %#v", upstreamBody["messages"])
	}
	message, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected message payload %#v", messages[0])
	}
	content, ok := message["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("unexpected content payload %#v", message["content"])
	}
	imagePart, ok := content[1].(map[string]any)
	if !ok || imagePart["type"] != "image_url" {
		t.Fatalf("unexpected image part %#v", content[1])
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok || imageURL["url"] != "https://example.com/cat.png" || imageURL["detail"] != "high" {
		t.Fatalf("unexpected image url payload %#v", imagePart["image_url"])
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

func TestClientResponsesPassesThroughRawBody(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected responses path, got %q", r.URL.Path)
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
				`{"id":"resp_1","object":"response","created_at":123,"model":"gpt-4.1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}],"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}`,
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})
	response, err := client.Responses(context.Background(), openaiwire.ResponsesRequest{
		Model:   "opena/gpt-4.1",
		RawBody: json.RawMessage(`{"model":"opena/gpt-4.1","input":"hello","temperature":0.3}`),
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("responses: %v", err)
	}

	if upstreamBody["model"] != "gpt-4.1" {
		t.Fatalf("unexpected upstream model %#v", upstreamBody["model"])
	}
	if upstreamBody["temperature"] != 0.3 {
		t.Fatalf("expected passthrough field, got %#v", upstreamBody["temperature"])
	}
	if response.ID != "resp_1" || response.Usage == nil || response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected response %#v", response)
	}
}

func TestClientChatCompletionsAppliesRTKCompressionWhenEnabled(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
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
				`{"id":"chatcmpl-1","object":"chat.completion","created":123,"model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`,
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})
	settingsManager := config.NewSettingsManager("", config.Config{RTK: config.NewRTKConfig(true)})
	ctx := config.WithSettingsManager(context.Background(), settingsManager)

	_, err := client.ChatCompletions(ctx, openaiwire.ChatCompletionsRequest{
		Model: "opena/gpt-4.1",
		Messages: []openaiwire.ChatMessage{{
			Role:    "user",
			Content: openaiwire.TextContent(strings.Repeat("same line\n", 120)),
		}},
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}

	messages := upstreamBody["messages"].([]any)
	message := messages[0].(map[string]any)
	content := message["content"].(string)
	if !strings.Contains(content, "... (119 duplicate lines)") {
		t.Fatalf("expected compressed content, got %q", content)
	}
}

func TestClientResponsesCompressesRawBodyWhenRTKEnabled(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
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
				`{"id":"resp_1","object":"response","created_at":123,"status":"completed","model":"gpt-4.1","output":[]}`,
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})
	settingsManager := config.NewSettingsManager("", config.Config{RTK: config.NewRTKConfig(true)})
	ctx := config.WithSettingsManager(context.Background(), settingsManager)

	_, err := client.Responses(ctx, openaiwire.ResponsesRequest{
		Model: "opena/gpt-4.1",
		RawBody: json.RawMessage([]byte(`{
			"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"` + strings.Repeat("same line\\n", 120) + `"}]}],
			"metadata":{"keep":"me"}
		}`)),
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("responses: %v", err)
	}

	if upstreamBody["metadata"].(map[string]any)["keep"] != "me" {
		t.Fatalf("expected raw body fields to survive, got %#v", upstreamBody["metadata"])
	}
	input := upstreamBody["input"].([]any)
	item := input[0].(map[string]any)
	content := item["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(content, "... (119 duplicate lines)") {
		t.Fatalf("expected compressed responses content, got %q", content)
	}
}

func TestClientStreamsResponses(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected responses path, got %q", r.URL.Path)
		}
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
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"created_at\":123}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"created_at\":123,\"status\":\"completed\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5,\"total_tokens\":15}}}\n\n" +
					"data: [DONE]\n\n",
			)),
		}, nil
	})}

	client := NewClient(httpClient, connection.Record{ProviderID: "openai", Name: "openai-user", APIKey: "token"})
	body, err := client.ResponsesStream(context.Background(), openaiwire.ResponsesRequest{
		Model:   "opena/gpt-4.1",
		RawBody: json.RawMessage(`{"model":"opena/gpt-4.1","input":"hello"}`),
	}, routing.Target{ProviderID: "openai", ProviderName: "OpenAI", RequestedModel: "gpt-4.1"})
	if err != nil {
		t.Fatalf("responses stream: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if !strings.Contains(string(data), `"type":"response.completed"`) {
		t.Fatalf("unexpected stream body %q", data)
	}
	if upstreamBody["stream"] != true {
		t.Fatalf("expected stream request, got %#v", upstreamBody["stream"])
	}
}
