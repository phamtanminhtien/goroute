package codex

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

const (
	codexInputTypeMessage          = "message"
	codexInputContentTypeInputText = "input_text"
	reasoningEncryptedContent      = "reasoning.encrypted_content"
)

var codexUnsupportedPayloadParams = map[string]struct{}{
	"frequency_penalty":     {},
	"logit_bias":            {},
	"logprobs":              {},
	"max_completion_tokens": {},
	"max_output_tokens":     {},
	"max_tokens":            {},
	"metadata":              {},
	"n":                     {},
	"presence_penalty":      {},
	"response_format":       {},
	"seed":                  {},
	"service_tier":          {},
	"stream_options":        {},
	"temperature":           {},
	"top_logprobs":          {},
	"top_p":                 {},
	"user":                  {},
}

var codexReasoningSuffixes = map[string]string{
	"-minimal": "minimal",
	"-low":     "low",
	"-medium":  "medium",
	"-high":    "high",
	"-xhigh":   "xhigh",
	"-none":    "none",
}

func transformCodexResponsesPayload(payload map[string]any, requestedModel string) ([]openaiwire.ResponseInputItem, error) {
	for key := range codexUnsupportedPayloadParams {
		delete(payload, key)
	}

	input, err := normalizeCodexInput(payload["input"])
	if err != nil {
		return nil, err
	}
	if len(input) == 0 {
		input = []openaiwire.ResponseInputItem{{
			Type: codexInputTypeMessage,
			Role: string(openaiwire.ChatRoleUser),
			Content: []openaiwire.ResponseInputContentPart{{
				Type: codexInputContentTypeInputText,
				Text: "...",
			}},
		}}
	}

	inputPayload, err := jsonCompatible(input)
	if err != nil {
		return nil, fmt.Errorf("encode normalized input: %w", err)
	}
	payload["input"] = inputPayload

	model, effort := codexUpstreamModelAndReasoning(requestedModel)
	payload["model"] = model
	if effort != "" && effort != "none" {
		payload["reasoning"] = map[string]any{
			"effort":  effort,
			"summary": "auto",
		}
		payload["include"] = appendUniqueInclude(payload["include"], reasoningEncryptedContent)
	} else if effort == "none" {
		delete(payload, "reasoning")
	}

	return input, nil
}

func normalizeCodexInput(raw any) ([]openaiwire.ResponseInputItem, error) {
	if raw == nil {
		return nil, nil
	}

	switch typed := raw.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, nil
		}
		return []openaiwire.ResponseInputItem{{
			Type: codexInputTypeMessage,
			Role: string(openaiwire.ChatRoleUser),
			Content: []openaiwire.ResponseInputContentPart{{
				Type: codexInputContentTypeInputText,
				Text: typed,
			}},
		}}, nil
	case []openaiwire.ResponseInputItem:
		return typed, nil
	case []any:
		if len(typed) == 0 {
			return nil, nil
		}
		items := make([]openaiwire.ResponseInputItem, 0, len(typed))
		for index, value := range typed {
			item, err := responseInputItemFromAny(value)
			if err != nil {
				return nil, fmt.Errorf("input[%d]: %w", index, err)
			}
			items = append(items, item)
		}
		return items, nil
	case map[string]any:
		item, err := responseInputItemFromAny(typed)
		if err != nil {
			return nil, fmt.Errorf("input: %w", err)
		}
		return []openaiwire.ResponseInputItem{item}, nil
	default:
		item, err := responseInputItemFromAny(typed)
		if err != nil {
			return nil, fmt.Errorf("input: %w", err)
		}
		return []openaiwire.ResponseInputItem{item}, nil
	}
}

func responseInputItemFromAny(value any) (openaiwire.ResponseInputItem, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return openaiwire.ResponseInputItem{}, err
	}
	var item openaiwire.ResponseInputItem
	if err := json.Unmarshal(data, &item); err != nil {
		return openaiwire.ResponseInputItem{}, err
	}
	return item, nil
}

func jsonCompatible(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func codexUpstreamModelAndReasoning(model string) (string, string) {
	model = strings.TrimSpace(model)
	for suffix, effort := range codexReasoningSuffixes {
		if strings.HasSuffix(model, suffix) {
			return strings.TrimSuffix(model, suffix), effort
		}
	}
	return model, ""
}

func appendUniqueInclude(raw any, value string) []string {
	includes := make([]string, 0)
	switch typed := raw.(type) {
	case []string:
		includes = append(includes, typed...)
	case []any:
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				includes = append(includes, text)
			}
		}
	}
	for _, include := range includes {
		if include == value {
			return includes
		}
	}
	return append(includes, value)
}
