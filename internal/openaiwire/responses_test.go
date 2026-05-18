package openaiwire

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponseUsageMarshalJSONIncludesReasoningTokensWhenMissing(t *testing.T) {
	payload, err := json.Marshal(ResponseUsage{
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	})
	if err != nil {
		t.Fatalf("marshal usage: %v", err)
	}

	text := string(payload)
	if !strings.Contains(text, `"output_tokens_details":{"reasoning_tokens":0}`) {
		t.Fatalf("expected default output token details, got %s", text)
	}
}

func TestResponseUsageMarshalJSONPreservesReasoningTokens(t *testing.T) {
	payload, err := json.Marshal(ResponseUsage{
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
		OutputTokensDetails: &ResponseOutputTokenDetails{
			ReasoningTokens: 3,
		},
	})
	if err != nil {
		t.Fatalf("marshal usage: %v", err)
	}

	text := string(payload)
	if !strings.Contains(text, `"output_tokens_details":{"reasoning_tokens":3}`) {
		t.Fatalf("expected preserved reasoning tokens, got %s", text)
	}
}

func TestResponsesResponseMarshalJSONPreservesUnknownFields(t *testing.T) {
	var response ResponsesResponse
	if err := json.Unmarshal([]byte(`{
		"id":"resp_1",
		"object":"response",
		"created_at":1779098607,
		"status":"completed",
		"background":false,
		"completed_at":1779098610,
		"error":null,
		"tool_usage":{"web_search":{"num_requests":0}},
		"usage":{
			"input_tokens":2490,
			"input_tokens_details":{"cached_tokens":2304},
			"output_tokens":122,
			"output_tokens_details":{"reasoning_tokens":42},
			"total_tokens":2612
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	response.Model = "gpt-5.4"

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	text := string(payload)
	for _, snippet := range []string{
		`"background":false`,
		`"completed_at":1779098610`,
		`"error":null`,
		`"tool_usage":{"web_search":{"num_requests":0}}`,
		`"model":"gpt-5.4"`,
		`"output_tokens_details":{"reasoning_tokens":42}`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected %s in %s", snippet, text)
		}
	}
}
