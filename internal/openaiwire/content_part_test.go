package openaiwire

import (
	"encoding/json"
	"testing"
)

func TestResponseInputTextPartMarshalKeepsEmptyText(t *testing.T) {
	payload, err := json.Marshal(ResponseInputContentPart{Type: "input_text"})
	if err != nil {
		t.Fatalf("marshal input text part: %v", err)
	}

	if got, want := string(payload), `{"type":"input_text","text":""}`; got != want {
		t.Fatalf("unexpected payload %s, want %s", got, want)
	}
}

func TestResponseInputImagePartMarshalOmitsText(t *testing.T) {
	payload, err := json.Marshal(ResponseInputContentPart{Type: "input_image", ImageURL: "https://example.com/image.png", Detail: "auto"})
	if err != nil {
		t.Fatalf("marshal input image part: %v", err)
	}

	if got, want := string(payload), `{"type":"input_image","image_url":"https://example.com/image.png","detail":"auto"}`; got != want {
		t.Fatalf("unexpected payload %s, want %s", got, want)
	}
}

func TestChatTextPartMarshalKeepsEmptyText(t *testing.T) {
	payload, err := json.Marshal(ChatMessageContentPart{Type: "text"})
	if err != nil {
		t.Fatalf("marshal chat text part: %v", err)
	}

	if got, want := string(payload), `{"type":"text","text":""}`; got != want {
		t.Fatalf("unexpected payload %s, want %s", got, want)
	}
}
