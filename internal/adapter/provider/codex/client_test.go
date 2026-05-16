package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
	"github.com/phamtanminhtien/goroute/internal/domain/routing"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	"github.com/phamtanminhtien/goroute/internal/usecase/chatcompletion"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientRequiresCredential(t *testing.T) {
	client := NewClient(connection.Record{ProviderID: "cx", Name: "codex-user"})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.4"})
	if err == nil {
		t.Fatal("expected credential error")
	}
	var configErr chatcompletion.ConnectionConfigurationError
	if !errors.As(err, &configErr) {
		t.Fatalf("expected connection configuration error, got %v", err)
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

	client := NewClientWithHTTPClient(httpClient, connection.Record{ProviderID: "cx", Name: "codex-user", AccessToken: "token"})

	response, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.3-codex"})
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

	client := NewClientWithHTTPClient(httpClient, connection.Record{ProviderID: "cx", Name: "codex-user", AccessToken: "token"})

	body, err := client.ChatCompletionsStream(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:    "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.3-codex"})
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

func TestClientRefreshesTokenProactivelyBeforeRequest(t *testing.T) {
	t.Cleanup(func() {
		oauthHTTPClient = http.DefaultClient
		timeNow = time.Now
	})

	timeNow = func() time.Time {
		return time.Unix(1710000000, 0)
	}

	var seenAuthHeaders []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seenAuthHeaders = append(seenAuthHeaders, r.Header.Get("Authorization"))
		if r.URL.Host == "auth.openai.com" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"fresh-token","refresh_token":"next-refresh","expires_in":3600,"token_type":"Bearer"}`)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[{"content":[{"type":"output_text","text":"hello back"}]}]}`)),
		}, nil
	})}
	oauthHTTPClient = httpClient

	client := NewClientWithHTTPClient(httpClient, connection.Record{
		ProviderID:           "cx",
		Name:                 "codex-user",
		AccessToken:          "stale-token",
		RefreshToken:         "refresh-token",
		AccessTokenExpiresAt: timeNow().Add(2 * time.Minute).Unix(),
	})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:    "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "Hello"}},
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.3-codex"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}

	if len(seenAuthHeaders) != 3 {
		t.Fatalf("expected refresh request and upstream request, got %#v", seenAuthHeaders)
	}
	if seenAuthHeaders[2] != "Bearer fresh-token" {
		t.Fatalf("expected refreshed bearer token, got %#v", seenAuthHeaders)
	}
}

func TestClientRefreshesTokenAfterUnauthorizedResponse(t *testing.T) {
	t.Cleanup(func() {
		oauthHTTPClient = http.DefaultClient
		timeNow = time.Now
	})

	timeNow = func() time.Time {
		return time.Unix(1710000000, 0)
	}

	var codexRequests int
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "auth.openai.com" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"fresh-token","refresh_token":"next-refresh","expires_in":3600,"token_type":"Bearer"}`)),
			}, nil
		}

		codexRequests++
		if codexRequests == 1 {
			if got := r.Header.Get("Authorization"); got != "Bearer stale-token" {
				t.Fatalf("expected first request to use stale token, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Status:     "401 Unauthorized",
				Body:       io.NopCloser(strings.NewReader("expired")),
			}, nil
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fresh-token" {
			t.Fatalf("expected retry to use refreshed token, got %q", got)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[{"content":[{"type":"output_text","text":"hello back"}]}]}`)),
		}, nil
	})}
	oauthHTTPClient = httpClient

	client := NewClientWithHTTPClient(httpClient, connection.Record{
		ProviderID:           "cx",
		Name:                 "codex-user",
		AccessToken:          "stale-token",
		RefreshToken:         "refresh-token",
		AccessTokenExpiresAt: timeNow().Add(10 * 24 * time.Hour).Unix(),
	})

	_, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model:    "cx/gpt-5.3-codex",
		Messages: []openaiwire.ChatMessage{{Role: "user", Content: "Hello"}},
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "gpt-5.3-codex"})
	if err != nil {
		t.Fatalf("chat completions: %v", err)
	}
	if codexRequests != 2 {
		t.Fatalf("expected 2 codex requests, got %d", codexRequests)
	}
}
