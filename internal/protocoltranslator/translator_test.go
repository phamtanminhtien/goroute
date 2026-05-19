package protocoltranslator

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func TestChatCompletionsToResponsesIncludesToolsAndToolOutputs(t *testing.T) {
	req := openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleSystem, Content: openaiwire.TextContent("be terse")},
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.PartsContent(
				openaiwire.ChatMessageContentPart{Type: "text", Text: "inspect"},
				openaiwire.ChatMessageContentPart{Type: "image_url", ImageURL: &openaiwire.ChatMessageImageURL{URL: "https://example.com/image.png", Detail: "high"}},
			)},
			{Role: openaiwire.ChatRoleAssistant, Content: openaiwire.TextContent(""), ToolCalls: []openaiwire.ToolCall{{
				ID: "call_1", Type: openaiwire.ToolTypeFunction,
				Function: openaiwire.ToolCallFunction{Name: "lookup_weather", Arguments: `{"city":"Saigon"}`},
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

	translated, err := ChatCompletionsToResponses(req)
	if err != nil {
		t.Fatalf("ChatCompletionsToResponses returned error: %v", err)
	}

	if translated.Instructions != "be terse" {
		t.Fatalf("expected system message folded into instructions, got %q", translated.Instructions)
	}
	var payload map[string]any
	if err := json.Unmarshal(translated.RawBody, &payload); err != nil {
		t.Fatalf("decode raw body: %v", err)
	}
	if payload["tool_choice"] == nil || payload["tools"] == nil {
		t.Fatalf("expected tools and tool_choice in payload, got %#v", payload)
	}
	inputItems := payload["input"].([]any)
	lastItem := inputItems[2].(map[string]any)
	if lastItem["type"] != "function_call_output" || lastItem["call_id"] != "call_1" || lastItem["output"] != "sunny" {
		t.Fatalf("unexpected tool output item %#v", inputItems[2])
	}
}

func TestChatCompletionsToResponsesUsesOutputTextForAssistantHistory(t *testing.T) {
	translated, err := ChatCompletionsToResponses(openaiwire.ChatCompletionsRequest{
		Model: "cx/gpt-5.4-mini",
		Messages: []openaiwire.ChatMessage{
			{Role: openaiwire.ChatRoleAssistant, Content: openaiwire.TextContent("previous answer")},
			{Role: openaiwire.ChatRoleUser, Content: openaiwire.TextContent("next question")},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletionsToResponses returned error: %v", err)
	}

	var payload struct {
		Input []struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(translated.RawBody, &payload); err != nil {
		t.Fatalf("decode raw body: %v", err)
	}

	if got := payload.Input[0].Content[0].Type; got != "output_text" {
		t.Fatalf("expected assistant history to use output_text, got %q", got)
	}
	if got := payload.Input[1].Content[0].Type; got != "input_text" {
		t.Fatalf("expected user input to use input_text, got %q", got)
	}
}

func TestResponsesToChatCompletionsMapsInputAndTools(t *testing.T) {
	translated, err := ResponsesToChatCompletions(openaiwire.ResponsesRequest{
		RawBody: json.RawMessage(`{
			"model":"gpt-5.4",
			"instructions":"be terse",
			"input":[
				{"type":"message","role":"user","content":[{"type":"input_text","text":"inspect"},{"type":"input_image","image_url":"https://example.com/image.png","detail":"high"}]},
				{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]},
				{"type":"function_call_output","call_id":"call_1","output":"sunny"}
			],
			"tools":[{"type":"function","name":"lookup_weather","parameters":{"type":"object"}}],
			"tool_choice":{"type":"function","name":"lookup_weather"}
		}`),
	})
	if err != nil {
		t.Fatalf("ResponsesToChatCompletions returned error: %v", err)
	}

	if len(translated.Messages) != 4 || translated.Messages[0].Role != openaiwire.ChatRoleSystem {
		t.Fatalf("unexpected translated messages %#v", translated.Messages)
	}
	if !translated.Messages[1].Content.IsParts() {
		t.Fatalf("expected image input to translate to chat content parts")
	}
	if translated.Messages[2].Role != openaiwire.ChatRoleAssistant || translated.Messages[2].Content.Text() != "done" {
		t.Fatalf("unexpected assistant message %#v", translated.Messages[2])
	}
	if translated.Messages[3].Role != openaiwire.ChatRoleTool || translated.Messages[3].ToolCallID != "call_1" {
		t.Fatalf("unexpected tool output message %#v", translated.Messages[3])
	}
	if len(translated.Tools) != 1 || translated.Tools[0].Function.Name != "lookup_weather" {
		t.Fatalf("unexpected tools %#v", translated.Tools)
	}
}

func TestResponsesStreamToChatCompletionsTranslatesSSE(t *testing.T) {
	body := ResponsesStreamToChatCompletions(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello \"}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"world\"}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\",\"usage\":{\"input_tokens\":3,\"output_tokens\":2,\"total_tokens\":5}}}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	streamText := string(data)
	if !strings.Contains(streamText, `"object":"chat.completion.chunk"`) || !strings.Contains(streamText, `"content":"hello "`) || !strings.Contains(streamText, `"content":"world"`) {
		t.Fatalf("expected translated chat chunks, got %q", streamText)
	}
	if !strings.Contains(streamText, `"finish_reason":"stop"`) || !strings.Contains(streamText, `data: [DONE]`) {
		t.Fatalf("expected translated terminal events, got %q", streamText)
	}
}

func TestResponsesStreamToChatCompletionsAddsDoneWhenResponsesStreamOmitsIt(t *testing.T) {
	body := ResponsesStreamToChatCompletions(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_2\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\"}}\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	streamText := string(data)
	if !strings.HasSuffix(streamText, "data: [DONE]\n\n") {
		t.Fatalf("expected translated chat stream to end with [DONE], got %q", streamText)
	}
	if strings.Count(streamText, "data: [DONE]") != 1 {
		t.Fatalf("expected exactly one [DONE], got %q", streamText)
	}
}

func TestResponsesStreamToChatCompletionsDoesNotReplayFullToolArgumentsOnDone(t *testing.T) {
	body := ResponsesStreamToChatCompletions(io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_tool\",\"object\":\"response\",\"created_at\":456,\"status\":\"in_progress\",\"model\":\"cx/gpt-5.4\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"create_transaction\",\"arguments\":\"\"}}\n\n" +
			"data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"transactions\\\":[\"}\n\n" +
			"data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"amount\\\":10000}]}\"}\n\n" +
			"data: {\"type\":\"response.function_call_arguments.done\",\"output_index\":0,\"arguments\":\"{\\\"transactions\\\":[{\\\"amount\\\":10000}]}\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"id\":\"fc_1\",\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"create_transaction\",\"arguments\":\"{\\\"transactions\\\":[{\\\"amount\\\":10000}]}\"}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_tool\",\"object\":\"response\",\"created_at\":456,\"status\":\"completed\",\"model\":\"cx/gpt-5.4\"}}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	var argumentDeltas []string
	var finishReason openaiwire.FinishReason
	for _, event := range sseDataEvents(data) {
		if event == "[DONE]" {
			continue
		}
		var chunk openaiwire.ChatCompletionsStreamChunk
		if err := json.Unmarshal([]byte(event), &chunk); err != nil {
			t.Fatalf("decode chunk %q: %v", event, err)
		}
		for _, choice := range chunk.Choices {
			for _, call := range choice.Delta.ToolCalls {
				argumentDeltas = append(argumentDeltas, call.Function.Arguments)
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}

	expected := []string{"", `{"transactions":[`, `{"amount":10000}]}`}
	if strings.Join(argumentDeltas, "|") != strings.Join(expected, "|") {
		t.Fatalf("unexpected argument deltas %#v", argumentDeltas)
	}
	if finishReason != openaiwire.FinishReasonToolCalls {
		t.Fatalf("unexpected finish reason %q", finishReason)
	}
}

func TestChatCompletionsStreamToResponsesTranslatesSSE(t *testing.T) {
	body := ChatCompletionsStreamToResponses(io.NopCloser(strings.NewReader(
		"data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":456,\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hello \"}}]}\n\n" +
			"data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":456,\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"world\"}}]}\n\n" +
			"data: {\"id\":\"chat_1\",\"object\":\"chat.completion.chunk\",\"created\":456,\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n",
	)))
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	streamText := string(data)
	if !strings.Contains(streamText, `"type":"response.output_text.delta"`) || !strings.Contains(streamText, `"delta":"hello "`) {
		t.Fatalf("expected responses deltas, got %q", streamText)
	}
	if !strings.Contains(streamText, `"type":"response.completed"`) || !strings.Contains(streamText, `data: [DONE]`) {
		t.Fatalf("expected completed response stream, got %q", streamText)
	}

	reconstructed, err := ParseResponsesSSE(data)
	if err != nil {
		t.Fatalf("parse translated stream: %v", err)
	}
	if reconstructed.TextValue() != "hello world" {
		t.Fatalf("expected reconstructed text, got %#v", reconstructed.Output)
	}
}

func TestRewriteResponsesStreamModelPreservesEventShape(t *testing.T) {
	body := io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.completed\",\"sequence_number\":88,\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":1779098607,\"status\":\"completed\",\"background\":false,\"completed_at\":1779098610,\"error\":null,\"model\":\"gpt-5.3\",\"tool_usage\":{\"web_search\":{\"num_requests\":0}},\"usage\":{\"input_tokens\":2490,\"input_tokens_details\":{\"cached_tokens\":2304},\"output_tokens\":122,\"output_tokens_details\":{\"reasoning_tokens\":42},\"total_tokens\":2612}}}\n\n" +
			"data: [DONE]\n\n",
	))

	stream := RewriteResponsesStreamModel(body, "cx/gpt-5.4")
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	text := string(data)
	for _, snippet := range []string{
		`"sequence_number":88`,
		`"background":false`,
		`"completed_at":1779098610`,
		`"error":null`,
		`"tool_usage":{"web_search":{"num_requests":0}}`,
		`"output_tokens_details":{"reasoning_tokens":42}`,
		`"model":"cx/gpt-5.4"`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected %s in %s", snippet, text)
		}
	}
}
