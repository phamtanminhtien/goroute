package openaiwire

import (
	"encoding/json"
	"strings"
	"testing"
)

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
