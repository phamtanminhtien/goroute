package chatcompletion

import "testing"

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
