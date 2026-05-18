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

func TestTranslateChatCompletionsRequestIncludesToolsAndToolOutputs(t *testing.T) {
	req := openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleSystem, Content: openaiwire.TextContent("be terse")},
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.PartsContent(
				openaiwire.ChatMessageContentPart{Type: "text", Text: "inspect"},
				openaiwire.ChatMessageContentPart{Type: "image_url", ImageURL: &openaiwire.ChatMessageImageURL{URL: "https://example.com/cat.png", Detail: "high"}},
			)},
			{Role: openaiwire.ChatRoleAssistant, Content: openaiwire.TextContent(""), ToolCalls: []openaiwire.ToolCall{{
				ID:   "call_1",
				Type: openaiwire.ToolTypeFunction,
				Function: openaiwire.ToolCallFunction{
					Name:      "lookup_weather",
					Arguments: `{"city":"Saigon"}`,
				},
			}}},
			{Role: openaiwire.ChatRoleTool, ToolCallID: "call_1", Content: openaiwire.TextContent("sunny")},
		},
		Tools: []openaiwire.Tool{{
			Type: openaiwire.ToolTypeFunction,
			Function: openaiwire.ToolFunction{
				Name:        "lookup_weather",
				Description: "weather lookup",
				Parameters:  json.RawMessage(`{"type":"object"}`),
				Strict:      true,
			},
		}},
		ToolChoice:      map[string]any{"type": "function", "function": map[string]any{"name": "lookup_weather"}},
		ReasoningEffort: "high",
	}

	translated, err := translateChatCompletionsRequest(req)
	if err != nil {
		t.Fatalf("translateChatCompletionsRequest returned error: %v", err)
	}

	if translated.Instructions != "be terse" {
		t.Fatalf("expected system message folded into instructions, got %q", translated.Instructions)
	}
	if len(translated.Input) != 3 {
		t.Fatalf("expected three input items, got %#v", translated.Input)
	}

	var payload map[string]any
	if err := json.Unmarshal(translated.RawBody, &payload); err != nil {
		t.Fatalf("decode raw body: %v", err)
	}

	if payload["instructions"] != "be terse" {
		t.Fatalf("expected instructions in payload, got %#v", payload["instructions"])
	}
	if payload["tool_choice"] == nil {
		t.Fatalf("expected tool_choice to be preserved in payload, got %#v", payload)
	}
	if payload["tools"] == nil {
		t.Fatalf("expected tools to be preserved in payload, got %#v", payload)
	}

	inputItems, ok := payload["input"].([]any)
	if !ok || len(inputItems) != 3 {
		t.Fatalf("expected translated input array, got %#v", payload["input"])
	}
	lastItem, ok := inputItems[2].(map[string]any)
	if !ok || lastItem["type"] != "function_call_output" || lastItem["call_id"] != "call_1" || lastItem["output"] != "sunny" {
		t.Fatalf("unexpected tool output item %#v", inputItems[2])
	}
}

func TestClientChatCompletionsMapsResponsesBackToChatCompletion(t *testing.T) {
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
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.5\",\"output\":[]}}\n\n" +
					"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"lookup_weather\",\"arguments\":\"{\\\"city\\\":\\\"Saigon\\\"}\"}}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":123,\"status\":\"completed\",\"model\":\"cx/gpt-5.5\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5,\"total_tokens\":15}}}\n\n" +
					"data: [DONE]\n\n",
			)),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	response, err := client.ChatCompletions(context.Background(), openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.TextContent("hello")},
		},
		Tools: []openaiwire.Tool{{
			Type: openaiwire.ToolTypeFunction,
			Function: openaiwire.ToolFunction{
				Name:       "lookup_weather",
				Parameters: json.RawMessage(`{"type":"object"}`),
			},
		}},
		ToolChoice: "auto",
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.5"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if upstreamBody["stream"] != true {
		t.Fatalf("expected upstream requests to force stream, got %#v", upstreamBody["stream"])
	}
	if upstreamBody["store"] != false {
		t.Fatalf("expected upstream requests to force store=false, got %#v", upstreamBody["store"])
	}
	if response.Object != "chat.completion" || response.Model != "cx/gpt-5.5" {
		t.Fatalf("unexpected chat completion metadata %#v", response)
	}
	if len(response.Choices) != 1 || response.Choices[0].FinishReason != openaiwire.FinishReasonToolCalls {
		t.Fatalf("unexpected chat completion choices %#v", response.Choices)
	}
	if len(response.Choices[0].Message.ToolCalls) != 1 || response.Choices[0].Message.ToolCalls[0].Function.Name != "lookup_weather" {
		t.Fatalf("unexpected tool call mapping %#v", response.Choices[0].Message.ToolCalls)
	}
}

func TestClientChatCompletionsStreamTranslatesResponsesSSE(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
					"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello \"}\n\n" +
					"data: {\"type\":\"response.output_text.delta\",\"delta\":\"world\"}\n\n" +
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

	body, err := client.ChatCompletionsStream(ctx, openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.TextContent("hello")},
		},
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.4"})
	if err != nil {
		t.Fatalf("ChatCompletionsStream returned error: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	streamText := string(data)
	if !strings.Contains(streamText, `"object":"chat.completion.chunk"`) {
		t.Fatalf("expected chat completion chunks, got %q", streamText)
	}
	if !strings.Contains(streamText, `"content":"hello "`) || !strings.Contains(streamText, `"content":"world"`) {
		t.Fatalf("expected translated text deltas, got %q", streamText)
	}
	if !strings.Contains(streamText, `"finish_reason":"stop"`) || !strings.Contains(streamText, `data: [DONE]`) {
		t.Fatalf("expected translated terminal events, got %q", streamText)
	}

	flow, thirdPartyLogs := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if len(thirdPartyLogs) != 1 {
		t.Fatalf("expected a single third-party log, got %#v", thirdPartyLogs)
	}
	if !strings.Contains(flow.TranslatedResponseBody, `"object":"chat.completion"`) {
		t.Fatalf("expected chat completion flow response, got %q", flow.TranslatedResponseBody)
	}
	if !strings.Contains(thirdPartyLogs[0].RequestURL, "/responses") {
		t.Fatalf("expected third-party logs to point at responses upstream, got %#v", thirdPartyLogs[0].RequestURL)
	}
}

func TestClientChatCompletionsAppliesDefaultInstructionWhenSystemMessagesAreMissing(t *testing.T) {
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
			Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_4\",\"object\":\"response\",\"created_at\":790,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\ndata: [DONE]\n\n")),
		}, nil
	})}

	client := NewClientWithHTTPClient(httpClient, connection.Record{ID: "cx-2", ProviderID: "cx", Name: "codex-user", APIKey: "token"})
	client.baseURL = "https://example.com/backend-api/codex"

	recorder := chatcompletion.NewFlowRecorder("req-3", time.Unix(0, 0).UTC())
	ctx := chatcompletion.WithFlowRecorder(context.Background(), recorder)

	_, err := client.ChatCompletions(ctx, openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.TextContent("hello")},
		},
	}, routing.Target{ProviderID: "cx", ProviderName: "Codex", RequestedModel: "cx/gpt-5.4"})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}

	if upstreamBody["instructions"] != defaultInstruction {
		t.Fatalf("expected default instruction fallback, got %#v", upstreamBody["instructions"])
	}

	flow, _ := recorder.SnapshotDetails(time.Unix(1, 0).UTC(), 1)
	if !strings.Contains(flow.TranslatedRequestBody, `"instructions":`) || !strings.Contains(flow.TranslatedRequestBody, `You are Codex, based on GPT-5.`) {
		t.Fatalf("expected translated request log to include default instruction, got %q", flow.TranslatedRequestBody)
	}
}
