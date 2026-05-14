package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/config"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientRequiresCredential(t *testing.T) {
	client := NewClient(config.ProviderConfig{Type: "codex", Name: "codex-user"})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderType: "codex", RequestedModel: "gpt-5.4"})
	if err == nil {
		t.Fatal("expected credential error")
	}
	if !strings.Contains(err.Error(), "no usable credential") {
		t.Fatalf("expected credential error, got %v", err)
	}
}

func TestClientConvertsChatCompletionsToCodexResponses(t *testing.T) {
	t.Setenv("MACHINE_ID", "test-machine")

	var upstreamBody map[string]any
	var sessionID string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/backend-api/codex/responses" {
			t.Fatalf("expected codex responses path, got %q", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("expected SSE accept header, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		if got := r.Header.Get("originator"); got != "codex-cli" {
			t.Fatalf("expected codex originator, got %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != defaultUserAgent {
			t.Fatalf("expected codex user agent, got %q", got)
		}
		sessionID = r.Header.Get("session_id")

		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &upstreamBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[{"content":[{"type":"output_text","text":"hello back"}]}]}`)),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, config.ProviderConfig{Type: "codex", Name: "codex-user", AccessToken: "token"})

	response, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}, routing.Target{ProviderType: "codex", RequestedModel: "gpt-5.3-codex"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}
	if response.Choices[0].Message.Content != "hello back" {
		t.Fatalf("unexpected response content: %#v", response.Choices[0].Message.Content)
	}

	if sessionID != "sess_"+hashContent("test-machine") {
		t.Fatalf("unexpected session id %q", sessionID)
	}
	if upstreamBody["model"] != "gpt-5.3-codex" {
		t.Fatalf("unexpected upstream model %#v", upstreamBody["model"])
	}
	if upstreamBody["instructions"] != "You are helpful." {
		t.Fatalf("unexpected instructions %#v", upstreamBody["instructions"])
	}
	if upstreamBody["stream"] != false {
		t.Fatalf("non-stream executor should disable upstream stream, got %#v", upstreamBody["stream"])
	}
	if upstreamBody["store"] != false {
		t.Fatalf("expected store=false, got %#v", upstreamBody["store"])
	}
	if _, ok := upstreamBody["include"].([]any); !ok {
		t.Fatalf("expected include reasoning encrypted content, got %#v", upstreamBody["include"])
	}
}

func TestClientStreamsCodexResponsesBody(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: first\n\n")),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, config.ProviderConfig{Type: "codex", Name: "codex-user", AccessToken: "token"})

	body, err := client.ChatCompletionsStream(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:    "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}, routing.Target{ProviderType: "codex", RequestedModel: "gpt-5.3-codex"})
	if err != nil {
		t.Fatalf("stream completions: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(data) != "data: first\n\n" {
		t.Fatalf("unexpected stream body %q", data)
	}
}
