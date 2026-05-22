package openaiwire

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponsesRequestUnmarshalAcceptsInputText(t *testing.T) {
	var request ResponsesRequest
	if err := json.Unmarshal([]byte(`{"model":"gpt-4.1","input":"hello"}`), &request); err != nil {
		t.Fatalf("unmarshal responses request: %v", err)
	}

	if request.InputText != "hello" {
		t.Fatalf("expected input text, got %q", request.InputText)
	}
	if len(request.Input) != 0 {
		t.Fatalf("expected no structured input items, got %#v", request.Input)
	}
}

func TestResponsesRequestUnmarshalAcceptsInputItems(t *testing.T) {
	var request ResponsesRequest
	if err := json.Unmarshal([]byte(`{
		"model":"gpt-4.1",
		"input":[{
			"type":"message",
			"role":"user",
			"content":[{"type":"input_text","text":"hello"}]
		}]
	}`), &request); err != nil {
		t.Fatalf("unmarshal responses request: %v", err)
	}

	if len(request.Input) != 1 {
		t.Fatalf("expected one input item, got %#v", request.Input)
	}
	if request.Input[0].Role != "user" || request.Input[0].Content[0].Text != "hello" {
		t.Fatalf("unexpected input item %#v", request.Input[0])
	}
}

func TestResponsesRequestUnmarshalRejectsInputObjectWithClearError(t *testing.T) {
	var request ResponsesRequest
	err := json.Unmarshal([]byte(`{
		"model":"gpt-4.1",
		"input":{
			"type":"message",
			"role":"user",
			"content":[{"type":"input_text","text":"hello"}]
		}
	}`), &request)
	if err == nil {
		t.Fatal("expected input object to be rejected")
	}
	if got, want := err.Error(), "input must be a string or an array of response input items"; got != want {
		t.Fatalf("unexpected error %q, want %q", got, want)
	}
}

func TestResponsesRequestUnmarshalAcceptsFunctionCallOutputParts(t *testing.T) {
	var request ResponsesRequest
	if err := json.Unmarshal([]byte(`{
		"model":"gpt-4.1",
		"input":[{
			"type":"function_call_output",
			"call_id":"call_1",
			"output":[
				{"type":"input_text","text":"screenshot"},
				{"type":"input_image","image_url":"data:image/png;base64,abc","detail":"high"}
			]
		}]
	}`), &request); err != nil {
		t.Fatalf("unmarshal responses request: %v", err)
	}

	if len(request.Input) != 1 {
		t.Fatalf("expected one input item, got %#v", request.Input)
	}
	item := request.Input[0]
	if item.Output != "" {
		t.Fatalf("expected string output to stay empty for part array, got %q", item.Output)
	}
	if len(item.OutputParts) != 2 {
		t.Fatalf("expected output parts, got %#v", item.OutputParts)
	}
	if item.OutputParts[1].Type != "input_image" || item.OutputParts[1].ImageURL != "data:image/png;base64,abc" {
		t.Fatalf("unexpected image output part %#v", item.OutputParts[1])
	}

	payload, err := json.Marshal(request.Input[0])
	if err != nil {
		t.Fatalf("marshal input item: %v", err)
	}
	text := string(payload)
	for _, snippet := range []string{
		`"type":"function_call_output"`,
		`"output":[{"type":"input_text","text":"screenshot"},{"type":"input_image","image_url":"data:image/png;base64,abc","detail":"high"}]`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected %s in %s", snippet, text)
		}
	}
}

func TestResponseInputItemMarshalJSONPreservesUnknownFields(t *testing.T) {
	var item ResponseInputItem
	if err := json.Unmarshal([]byte(`{
		"type":"reasoning",
		"id":"rs_123",
		"summary":[{"type":"summary_text","text":"thinking"}]
	}`), &item); err != nil {
		t.Fatalf("unmarshal input item: %v", err)
	}

	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal input item: %v", err)
	}

	text := string(payload)
	for _, snippet := range []string{
		`"type":"reasoning"`,
		`"id":"rs_123"`,
		`"summary":[{"type":"summary_text","text":"thinking"}]`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected %s in %s", snippet, text)
		}
	}
}
