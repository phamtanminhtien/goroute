package codex

import (
	"context"
	"encoding/json"
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

func TestClientResponsesForcesStreamingAndReconstructsSSE(t *testing.T) {
	var upstreamBody map[string]any
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("expected event stream accept header, got %q", got)
		}
		if got := r.Header.Get("originator"); got != "codex-cli" {
			t.Fatalf("expected codex originator header, got %q", got)
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
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.5\",\"output\":[]}}\n\n" +
					"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello from codex\"}]}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"completed\",\"model\":\"cx/gpt-5.5\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5,\"total_tokens\":15}}}\n\n" +
					"data: [DONE]\n\n",
			)),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	response, err := client.Responses(context.Background(), openaiwire.ResponsesRequest{
		Model:   "cx/gpt-5.4",
		RawBody: json.RawMessage(`{"model":"cx/gpt-5.4","input":"hello","store":true}`),
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.5"})
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}

	if upstreamBody["model"] != "cx/gpt-5.5" {
		t.Fatalf("expected target model override, got %#v", upstreamBody["model"])
	}
	if upstreamBody["stream"] != true {
		t.Fatalf("expected responses request to force stream, got %#v", upstreamBody["stream"])
	}
	if upstreamBody["store"] != false {
		t.Fatalf("expected responses request to force store=false, got %#v", upstreamBody["store"])
	}
	if response.ID != "resp_1" || response.Status != openaiwire.ResponsesStatusCompleted {
		t.Fatalf("unexpected response metadata %#v", response)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage %#v", response.Usage)
	}
	if len(response.Output) != 1 {
		t.Fatalf("expected a reconstructed output item, got %#v", response.Output)
	}
	content := response.Output[0].Content
	if len(content) != 1 || content[0].Text != "hello from codex" {
		t.Fatalf("unexpected reconstructed content %#v", content)
	}
}

func TestClientResponsesForwardsOnlyWhitelistedInboundHeaders(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("expected upstream bearer token, got %q", got)
		}
		if got := r.Header.Get("Originator"); got != "Codex Desktop" {
			t.Fatalf("expected forwarded originator header, got %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "Codex Desktop/0.131.0-alpha.9" {
			t.Fatalf("expected forwarded user-agent, got %q", got)
		}
		if got := r.Header.Get("Session-Id"); got != "session-123" {
			t.Fatalf("expected forwarded session-id, got %q", got)
		}
		if got := r.Header.Get("X-Codex-Beta-Features"); got != "terminal_resize_reflow" {
			t.Fatalf("expected forwarded beta feature header, got %q", got)
		}
		if got := r.Header.Get("X-Codex-Turn-Metadata"); got != "{\"thread_id\":\"thread-123\"}" {
			t.Fatalf("expected forwarded turn metadata header, got %q", got)
		}
		if got := r.Header.Get("X-Unrelated-Header"); got != "" {
			t.Fatalf("expected non-whitelisted header to be dropped, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("expected stream accept header, got %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"completed\",\"model\":\"cx/gpt-5.5\",\"output\":[]}}\n\ndata: [DONE]\n\n")),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	ctx := chatcompletion.WithInboundHeaders(context.Background(), http.Header{
		"Authorization":         []string{"Bearer client-token"},
		"Originator":            []string{"Codex Desktop"},
		"User-Agent":            []string{"Codex Desktop/0.131.0-alpha.9"},
		"Session-Id":            []string{"session-123"},
		"X-Codex-Beta-Features": []string{"terminal_resize_reflow"},
		"X-Codex-Turn-Metadata": []string{"{\"thread_id\":\"thread-123\"}"},
		"X-Unrelated-Header":    []string{"should-not-pass"},
	})

	if _, err := client.Responses(ctx, openaiwire.ResponsesRequest{
		Model:   "cx/gpt-5.4",
		RawBody: json.RawMessage(`{"input":"hello"}`),
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.5"}); err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}
}

func TestClientResponsesStreamRecordsReconstructedResponse(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\",\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}}\n\n" +
					"data: [DONE]\n\n",
			)),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ID: "cx-1", ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	recorder := chatcompletion.NewFlowRecorder("req-1", time.Unix(0, 0).UTC())
	recorder.SetResolvedTarget(routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "combo/fast"})
	ctx := chatcompletion.WithFlowRecorder(context.Background(), recorder)

	body, err := client.ResponsesStream(ctx, openaiwire.ResponsesRequest{
		Model:   "cx/gpt-5.4",
		RawBody: json.RawMessage(`{"input":"hello"}`),
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.4"})
	if err != nil {
		t.Fatalf("ResponsesStream returned error: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if !strings.Contains(string(data), `"type":"response.completed"`) {
		t.Fatalf("unexpected stream body %q", data)
	}

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if len(thirdPartyLogs) != 1 {
		t.Fatalf("expected a single third-party log, got %#v", thirdPartyLogs)
	}
	if thirdPartyLogs[0].ResponseStatusCode != http.StatusOK {
		t.Fatalf("expected third-party log to capture status code, got %#v", thirdPartyLogs[0])
	}
	if !strings.Contains(flow.TranslatedResponseBody, `"id":"resp_2"`) {
		t.Fatalf("expected reconstructed response in flow log, got %q", flow.TranslatedResponseBody)
	}
	if !strings.Contains(flow.TranslatedResponseBody, `"model":"combo/fast"`) {
		t.Fatalf("expected normalized model in flow log, got %q", flow.TranslatedResponseBody)
	}
	if flow.ProviderRequestMode != chatcompletion.RequestModeStream {
		t.Fatalf("expected provider request mode stream, got %q", flow.ProviderRequestMode)
	}
}

func TestClientResponsesAppliesDefaultInstructionWhenMissingAndLogsTranslatedPayload(t *testing.T) {
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
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_3\",\"object\":\"response\",\"created_at\":789,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\ndata: [DONE]\n\n")),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ID: "cx-1", ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	recorder := chatcompletion.NewFlowRecorder("req-2", time.Unix(0, 0).UTC())
	ctx := chatcompletion.WithFlowRecorder(context.Background(), recorder)

	_, err := client.Responses(ctx, openaiwire.ResponsesRequest{
		Model:   "cx/gpt-5.4",
		RawBody: json.RawMessage(`{"input":"hello","instructions":"   "}`),
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.4"})
	if err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}

	if upstreamBody["instructions"] != defaultInstruction {
		t.Fatalf("expected default instruction fallback, got %#v", upstreamBody["instructions"])
	}

	flow, _ := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if !strings.Contains(flow.TranslatedRequestBody, `"instructions":`) || !strings.Contains(flow.TranslatedRequestBody, `You are Codex, based on GPT-5.`) {
		t.Fatalf("expected translated request log to include default instruction, got %q", flow.TranslatedRequestBody)
	}
}
