package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientAnthropicMessagesSendsAnthropicHeadersAndModel(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("expected messages path, got %q", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "token" {
			t.Fatalf("expected x-api-key token, got %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("expected anthropic-version header")
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &upstreamBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}`))}, nil
	})}
	client := NewClientWithBaseURL(httpClient, connection.Record{ID: "anthropic-1", ProviderID: "anthropic", Name: "Anthropic", APIKey: "token"}, "https://api.anthropic.test")

	response, err := client.AnthropicMessages(context.Background(), anthropicwire.MessagesRequest{
		Model:     "anthropic/claude-sonnet-4-5",
		MaxTokens: 32,
		Messages:  []anthropicwire.MessageParam{{Role: "user", Content: anthropicwire.TextContent("hello")}},
	}, routing.Target{ProviderID: "anthropic", ProviderName: "Anthropic", RequestedModel: "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("AnthropicMessages returned error: %v", err)
	}
	if upstreamBody["model"] != "claude-sonnet-4-5" || upstreamBody["max_tokens"] != float64(32) {
		t.Fatalf("unexpected upstream body %#v", upstreamBody)
	}
	if response.ID != "msg_1" || response.Content[0].Text != "hi" {
		t.Fatalf("unexpected response %#v", response)
	}
}

func TestClientAnthropicMessagesStreamForcesStream(t *testing.T) {
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
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))}, nil
	})}
	client := NewClientWithBaseURL(httpClient, connection.Record{ID: "anthropic-1", ProviderID: "anthropic", Name: "Anthropic", APIKey: "token"}, "https://api.anthropic.test")

	body, err := client.AnthropicMessagesStream(context.Background(), anthropicwire.MessagesRequest{
		Model:    "anthropic/claude-sonnet-4-5",
		Messages: []anthropicwire.MessageParam{{Role: "user", Content: anthropicwire.TextContent("hello")}},
	}, routing.Target{ProviderID: "anthropic", ProviderName: "Anthropic", RequestedModel: "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("AnthropicMessagesStream returned error: %v", err)
	}
	defer body.Close()
	if _, err := io.ReadAll(body); err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if upstreamBody["stream"] != true {
		t.Fatalf("expected stream true, got %#v", upstreamBody)
	}
}

func TestClientAnthropicMessagesReconstructsUnexpectedStreamResponse(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-5\",\"content\":[]}}\n\n" +
					"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n" +
					"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
			)),
		}, nil
	})}
	client := NewClientWithBaseURL(httpClient, connection.Record{ID: "anthropic-1", ProviderID: "anthropic", Name: "Anthropic", APIKey: "token"}, "https://api.anthropic.test")

	response, err := client.AnthropicMessages(context.Background(), anthropicwire.MessagesRequest{
		Model:    "anthropic/claude-sonnet-4-5",
		Messages: []anthropicwire.MessageParam{{Role: "user", Content: anthropicwire.TextContent("hello")}},
	}, routing.Target{ProviderID: "anthropic", ProviderName: "Anthropic", RequestedModel: "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("AnthropicMessages returned error: %v", err)
	}
	if response.ID != "msg_1" || response.Content[0].Text != "hi" {
		t.Fatalf("unexpected reconstructed response %#v", response)
	}
}
