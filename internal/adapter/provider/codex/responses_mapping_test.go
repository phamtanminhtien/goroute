package codex

import (
	"io"
	"strings"
	"testing"
)

func TestParseResponsesSSEBuildsChatCompletionFromFinalAssistantMessage(t *testing.T) {
	data := []byte(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_123\",\"created_at\":1712345678,\"status\":\"in_progress\"}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"first\"}]}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"reasoning\",\"summary\":[{\"type\":\"summary_text\",\"text\":\"thinking\"}]}}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":2,\"item\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"final\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_123\",\"created_at\":1712345678,\"status\":\"completed\",\"usage\":{\"input_tokens\":10,\"output_tokens\":2,\"total_tokens\":12,\"input_tokens_details\":{\"cached_tokens\":4}}}}\n\n",
	)

	response, err := parseResponsesSSE(data)
	if err != nil {
		t.Fatalf("parse responses SSE: %v", err)
	}

	completion := responseToChatCompletion(response, "gpt-5.3-codex")
	if completion.ID != "chatcmpl-123" {
		t.Fatalf("unexpected completion id %q", completion.ID)
	}
	if got := completion.Choices[0].Message.Content; got != "final" {
		t.Fatalf("unexpected final content %q", got)
	}
	if got := completion.Choices[0].FinishReason; got != "stop" {
		t.Fatalf("unexpected finish reason %q", got)
	}
	if completion.Usage == nil || completion.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage %#v", completion.Usage)
	}
	if completion.Usage.PromptTokensDetails == nil || completion.Usage.PromptTokensDetails.CachedTokens != 4 {
		t.Fatalf("unexpected usage details %#v", completion.Usage)
	}
}

func TestResponseToChatCompletionMapsToolCalls(t *testing.T) {
	data := []byte(
		"data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"type\":\"function_call\",\"call_id\":\"call_123\",\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"a.txt\\\"}\"}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_123\",\"created_at\":1712345678,\"status\":\"completed\"}}\n\n",
	)

	response, err := parseResponsesSSE(data)
	if err != nil {
		t.Fatalf("parse responses SSE: %v", err)
	}

	completion := responseToChatCompletion(response, "gpt-5.3-codex")
	if got := completion.Choices[0].FinishReason; got != "tool_calls" {
		t.Fatalf("unexpected finish reason %q", got)
	}
	if got := len(completion.Choices[0].Message.ToolCalls); got != 1 {
		t.Fatalf("unexpected tool call count %d", got)
	}
	if got := completion.Choices[0].Message.ToolCalls[0].Function.Name; got != "read_file" {
		t.Fatalf("unexpected tool name %q", got)
	}
}

func TestTransformResponsesToChatCompletionsStream(t *testing.T) {
	body := io.NopCloser(strings.NewReader(
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_123\",\"created_at\":1712345678}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":{\"type\":\"function_call\",\"call_id\":\"call_123\",\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"a.txt\\\"}\"}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_123\",\"created_at\":1712345678,\"usage\":{\"input_tokens\":10,\"output_tokens\":2,\"total_tokens\":12,\"input_tokens_details\":{\"cached_tokens\":4}}}}\n\n",
	))

	stream := transformResponsesToChatCompletionsStream(body, "cx/gpt-5.3-codex")
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read transformed stream: %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "\"content\":\"Hello\"") {
		t.Fatalf("expected content delta, got %q", text)
	}
	if !strings.Contains(text, "\"tool_calls\":[") {
		t.Fatalf("expected tool call delta, got %q", text)
	}
	if !strings.Contains(text, "\"finish_reason\":\"tool_calls\"") {
		t.Fatalf("expected tool_calls finish reason, got %q", text)
	}
	if !strings.Contains(text, "\"prompt_tokens_details\":{\"cached_tokens\":4}") {
		t.Fatalf("expected usage details, got %q", text)
	}
	if !strings.Contains(text, "data: [DONE]") {
		t.Fatalf("expected done marker, got %q", text)
	}
}
