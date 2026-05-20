package protocoltranslator

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestAnthropicToResponsesMapsToolsAndToolResults(t *testing.T) {
	req := anthropicwire.MessagesRequest{
		Model:     "anthropic/claude-sonnet-4-5",
		System:    "be terse",
		MaxTokens: 128,
		Messages: []anthropicwire.MessageParam{
			{Role: "user", Content: anthropicwire.BlocksContent(anthropicwire.ContentBlock{Type: "text", Text: "inspect"})},
			{Role: "assistant", Content: anthropicwire.BlocksContent(anthropicwire.ContentBlock{Type: "tool_use", ID: "toolu_1", Name: "lookup_weather", Input: json.RawMessage(`{"city":"Saigon"}`)})},
			{Role: "user", Content: anthropicwire.BlocksContent(anthropicwire.ContentBlock{Type: "tool_result", ToolUseID: "toolu_1", Content: "sunny"})},
		},
		Tools: []anthropicwire.Tool{{Name: "lookup_weather", InputSchema: json.RawMessage(`{"type":"object"}`)}},
	}

	translated, err := AnthropicToResponses(req)
	if err != nil {
		t.Fatalf("AnthropicToResponses returned error: %v", err)
	}
	if translated.Instructions != "be terse" {
		t.Fatalf("unexpected instructions %q", translated.Instructions)
	}
	if len(translated.Input) != 3 {
		t.Fatalf("unexpected input %#v", translated.Input)
	}
	if translated.Input[1].Type != "function_call" || translated.Input[1].CallID != "toolu_1" {
		t.Fatalf("unexpected tool call %#v", translated.Input[1])
	}
	if translated.Input[2].Type != "function_call_output" || translated.Input[2].Output != "sunny" {
		t.Fatalf("unexpected tool result %#v", translated.Input[2])
	}
	var payload map[string]any
	if err := json.Unmarshal(translated.RawBody, &payload); err != nil {
		t.Fatalf("decode raw body: %v", err)
	}
	if payload["max_output_tokens"] != float64(128) || payload["tools"] == nil {
		t.Fatalf("unexpected raw payload %#v", payload)
	}
}

func TestResponsesToAnthropicMapsFunctionCalls(t *testing.T) {
	response := ResponsesToAnthropic(openaiwire.ResponsesResponse{
		ID:        "resp_1",
		Model:     "cx/gpt-5.4",
		CreatedAt: 123,
		Output: []openaiwire.OutputItem{{
			Type: openaiwire.OutputItemTypeFunctionCall,
			ID:   "fc_1", CallID: "call_1", Name: "lookup_weather", Arguments: `{"city":"Saigon"}`,
		}},
		Usage: &openaiwire.ResponseUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
	})

	if response.ID != "msg_1" || response.StopReason != anthropicwire.StopReasonToolUse {
		t.Fatalf("unexpected response metadata %#v", response)
	}
	if len(response.Content) != 1 || response.Content[0].Type != anthropicwire.ContentTypeToolUse || response.Content[0].Name != "lookup_weather" {
		t.Fatalf("unexpected content %#v", response.Content)
	}
	if response.Usage == nil || response.Usage.InputTokens != 10 || response.Usage.OutputTokens != 5 {
		t.Fatalf("unexpected usage %#v", response.Usage)
	}
}

func TestResponsesStreamToAnthropicTranslatesTextDeltas(t *testing.T) {
	body := ResponsesStreamToAnthropic(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\",\"usage\":{\"input_tokens\":3,\"output_tokens\":1,\"total_tokens\":4}}}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	stream := string(data)
	if !strings.Contains(stream, "event: message_start") || !strings.Contains(stream, `"text":"hello"`) || !strings.Contains(stream, "event: message_stop") {
		t.Fatalf("unexpected anthropic stream %q", stream)
	}
	messageStartIndex := strings.Index(stream, "event: message_start")
	pingIndex := strings.Index(stream, "event: ping")
	contentBlockStartIndex := strings.Index(stream, "event: content_block_start")
	if pingIndex == -1 || !(messageStartIndex < pingIndex && pingIndex < contentBlockStartIndex) {
		t.Fatalf("expected ping between message_start and content_block_start, got %q", stream)
	}
	if !strings.Contains(stream, "\n\n\nevent: ping") || !strings.Contains(stream, "\n\n\nevent: content_block_start") {
		t.Fatalf("expected triple-newline Anthropic SSE frame separators, got %q", stream)
	}
	if !strings.Contains(stream, `"id":"msg_2"`) || !strings.Contains(stream, `"stop_reason":null`) || strings.Contains(stream, `"type":"message_delta","stop_reason"`) {
		t.Fatalf("unexpected anthropic lifecycle payload %q", stream)
	}
	if !strings.Contains(stream, `"content_block":{"text":"","type":"text"}`) {
		t.Fatalf("expected explicit empty text content block, got %q", stream)
	}
	if !strings.Contains(stream, `"index":0`) || !strings.Contains(stream, `"index":0,"type":"content_block_stop"`) {
		t.Fatalf("expected content block events to include index zero, got %q", stream)
	}
}

func TestResponsesStreamToAnthropicAcceptsTextDeltaField(t *testing.T) {
	body := ResponsesStreamToAnthropic(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_3\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"text\":\"hello\"}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if stream := string(data); !strings.Contains(stream, `"text":"hello"`) {
		t.Fatalf("unexpected anthropic stream %q", stream)
	}
}

func TestResponsesStreamToAnthropicStreamsToolUse(t *testing.T) {
	body := ResponsesStreamToAnthropic(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_tool\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"create_transaction\",\"arguments\":\"\"}}\n\n" +
			"data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"transactions\\\":[\"}\n\n" +
			"data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"amount\\\":10000}]}\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"create_transaction\",\"arguments\":\"{\\\"transactions\\\":[{\\\"amount\\\":10000}]}\"}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_tool\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\"}}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	stream := string(data)
	for _, expected := range []string{
		`"type":"tool_use"`,
		`"id":"call_1"`,
		`"name":"create_transaction"`,
		`"type":"input_json_delta"`,
		`"partial_json":"{\"transactions\":["`,
		`"partial_json":"{\"amount\":10000}]}"`,
		`"stop_reason":"tool_use"`,
	} {
		if !strings.Contains(stream, expected) {
			t.Fatalf("expected %s in %q", expected, stream)
		}
	}
	if strings.Contains(stream, `"type":"text","text":""`) {
		t.Fatalf("tool-only stream should not open a text block: %q", stream)
	}
}
