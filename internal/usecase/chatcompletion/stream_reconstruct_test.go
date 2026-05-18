package chatcompletion

import (
	"io"
	"strings"
	"testing"
)

func TestReconstructOpenAIResponseFromSSE(t *testing.T) {
	data := []byte("data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":123,\"model\":\"gpt-4.1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hello \"}}]}\n\ndata: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":123,\"model\":\"gpt-4.1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"world\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12}}\n\ndata: [DONE]\n\n")

	response, ok := ReconstructOpenAIResponseFromSSE(data, "fallback-model")
	if !ok {
		t.Fatal("expected reconstruct success")
	}
	if response.ID != "chatcmpl-1" {
		t.Fatalf("unexpected id %q", response.ID)
	}
	if response.Object != "chat.completion.chunk" {
		t.Fatalf("unexpected object %q", response.Object)
	}
	if response.Model != "gpt-4.1" {
		t.Fatalf("unexpected model %q", response.Model)
	}
	if len(response.Choices) != 1 {
		t.Fatalf("unexpected choices %#v", response.Choices)
	}
	if response.Choices[0].Message.Role != "assistant" {
		t.Fatalf("unexpected role %#v", response.Choices[0].Message.Role)
	}
	if response.Choices[0].Message.Content != "Hello world" {
		t.Fatalf("unexpected content %#v", response.Choices[0].Message.Content)
	}
	if response.Choices[0].FinishReason != "stop" {
		t.Fatalf("unexpected finish reason %q", response.Choices[0].FinishReason)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage %#v", response.Usage)
	}
}

func TestExtractTextFromSSEForResponsesFormats(t *testing.T) {
	data := []byte("data: {\"text\":\"Hello \"}\n\ndata: {\"delta\":\"there\"}\n\ndata: {\"content\":[{\"type\":\"output_text\",\"text\":\"!\"}]}\n\ndata: {\"response\":{\"output\":[{\"content\":[{\"type\":\"output_text\",\"text\":\" Done\"}]}]}}\n\ndata: [DONE]\n\n")

	text := ExtractTextFromSSE(data)
	if text != "Hello there! Done" {
		t.Fatalf("unexpected text %q", text)
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
