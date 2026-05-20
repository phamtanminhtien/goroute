package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestAnthropicMessagesEndpointRoutesThroughResponses(t *testing.T) {
	provider := &testProvider{responsesResponse: openaiwire.ResponsesResponse{
		ID:        "resp_1",
		Object:    "response",
		CreatedAt: 123,
		Status:    openaiwire.ResponsesStatusCompleted,
		Model:     "gpt-5.4",
		Output: []openaiwire.OutputItem{{
			Type: openaiwire.OutputItemTypeMessage,
			Role: "assistant",
			Content: []openaiwire.OutputContent{{
				Type: openaiwire.OutputContentTypeOutputText,
				Text: "hello",
			}},
		}},
		Usage: &openaiwire.ResponseUsage{InputTokens: 3, OutputTokens: 2, TotalTokens: 5},
	}}
	handler := testServer(t, provider)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"cx/gpt-5.4","system":[{"type":"text","text":"be terse"},{"cache_control":{"type":"ephemeral"},"text":"prefer json"}],"messages":[{"role":"user","content":"hi"}],"max_tokens":32}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Type != "message" || response.Role != "assistant" || response.Model != "cx/gpt-5.4" || response.Content[0].Text != "hello" {
		t.Fatalf("unexpected anthropic response %#v", response)
	}
	if provider.lastResponsesReq.Instructions != "be terse\nprefer json" || len(provider.lastResponsesReq.Input) != 1 {
		t.Fatalf("unexpected translated request %#v", provider.lastResponsesReq)
	}
	if provider.lastResponsesTarget.RequestedModel != "gpt-5.4" {
		t.Fatalf("unexpected target %#v", provider.lastResponsesTarget)
	}
}

func TestAnthropicMessagesEndpointStreams(t *testing.T) {
	provider := streamingTestProvider{testProvider: &testProvider{}, body: "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"in_progress\",\"model\":\"gpt-5.4\",\"output\":[]}}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"completed\",\"model\":\"gpt-5.4\"}}\n\ndata: [DONE]\n\n"}
	handler := testServer(t, provider)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"cx/gpt-5.4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("expected event stream content type, got %q", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "event: message_start") || !strings.Contains(body, `"text":"hi"`) {
		t.Fatalf("unexpected stream body %q", body)
	}
}
