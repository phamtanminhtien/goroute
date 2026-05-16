package chatcompletion

import (
	"bufio"
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type openAIStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int                    `json:"index"`
		Delta        openaiwire.ChatMessage `json:"delta"`
		FinishReason string                 `json:"finish_reason"`
	} `json:"choices"`
	Usage *openaiwire.Usage `json:"usage,omitempty"`
}

func ReconstructOpenAIResponseFromSSE(data []byte, fallbackModel string) (openaiwire.ChatCompletionsResponse, bool) {
	events := sseDataEvents(data)
	if len(events) == 0 {
		return openaiwire.ChatCompletionsResponse{}, false
	}

	response := openaiwire.ChatCompletionsResponse{
		Object: "chat.completion",
		Model:  fallbackModel,
	}
	choices := make(map[int]*openaiwire.ChatChoice)

	for _, event := range events {
		if event == "[DONE]" {
			continue
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(event), &chunk); err != nil {
			return openaiwire.ChatCompletionsResponse{}, false
		}

		if chunk.ID != "" {
			response.ID = chunk.ID
		}
		if chunk.Object != "" {
			response.Object = chunk.Object
		}
		if chunk.Created != 0 {
			response.Created = chunk.Created
		}
		if chunk.Model != "" {
			response.Model = chunk.Model
		}
		if chunk.Usage != nil {
			copied := *chunk.Usage
			response.Usage = &copied
		}

		for _, choice := range chunk.Choices {
			current := choices[choice.Index]
			if current == nil {
				current = &openaiwire.ChatChoice{
					Index:   choice.Index,
					Message: openaiwire.ChatMessage{},
				}
				choices[choice.Index] = current
			}
			if choice.Delta.Role != "" {
				current.Message.Role = choice.Delta.Role
			}
			current.Message.Content = mergeContent(current.Message.Content, choice.Delta.Content)
			current.Message.ToolCallID = choice.Delta.ToolCallID
			current.Message.ToolCalls = mergeToolCalls(current.Message.ToolCalls, choice.Delta.ToolCalls)
			if choice.FinishReason != "" {
				current.FinishReason = choice.FinishReason
			}
		}
	}

	if len(choices) == 0 {
		return openaiwire.ChatCompletionsResponse{}, false
	}

	indexes := make([]int, 0, len(choices))
	for index := range choices {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	response.Choices = make([]openaiwire.ChatChoice, 0, len(indexes))
	for _, index := range indexes {
		response.Choices = append(response.Choices, *choices[index])
	}

	return response, true
}

func BuildAssistantResponse(model string, content string) openaiwire.ChatCompletionsResponse {
	return openaiwire.ChatCompletionsResponse{
		Object: "chat.completion",
		Model:  model,
		Choices: []openaiwire.ChatChoice{{
			Index: 0,
			Message: openaiwire.ChatMessage{
				Role:    "assistant",
				Content: content,
			},
		}},
	}
}

func ExtractTextFromSSE(data []byte) string {
	events := sseDataEvents(data)
	if len(events) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, event := range events {
		if event == "[DONE]" {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(event), &payload); err != nil {
			if builder.Len() == 0 {
				builder.WriteString(event)
			}
			continue
		}

		builder.WriteString(extractTextValue(payload))
	}

	return builder.String()
}

func sseDataEvents(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	events := make([]string, 0, 8)
	var builder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if builder.Len() > 0 {
				events = append(events, strings.TrimSuffix(builder.String(), "\n"))
				builder.Reset()
			}
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		builder.WriteString(payload)
		builder.WriteByte('\n')
	}
	if builder.Len() > 0 {
		events = append(events, strings.TrimSuffix(builder.String(), "\n"))
	}

	return events
}

func mergeContent(current any, delta any) any {
	switch typed := delta.(type) {
	case nil:
		return current
	case string:
		currentString, _ := current.(string)
		return currentString + typed
	case []any:
		currentSlice, _ := current.([]any)
		return append(currentSlice, typed...)
	default:
		if current == nil {
			return typed
		}
		return current
	}
}

func mergeToolCalls(current []openaiwire.ToolCall, delta []openaiwire.ToolCall) []openaiwire.ToolCall {
	if len(delta) == 0 {
		return current
	}

	if len(current) == 0 {
		return append([]openaiwire.ToolCall(nil), delta...)
	}

	out := append([]openaiwire.ToolCall(nil), current...)
	for _, next := range delta {
		matched := false
		for i := range out {
			if out[i].ID == next.ID && next.ID != "" {
				if next.Type != "" {
					out[i].Type = next.Type
				}
				if next.Function.Name != "" {
					out[i].Function.Name = next.Function.Name
				}
				if next.Function.Arguments != "" {
					out[i].Function.Arguments += next.Function.Arguments
				}
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, next)
		}
	}
	return out
}

func extractTextValue(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if text, _ := typed["text"].(string); text != "" {
			return text
		}
		if delta, ok := typed["delta"].(map[string]any); ok {
			return extractTextValue(delta)
		}
		if content, ok := typed["content"].([]any); ok {
			var builder strings.Builder
			for _, current := range content {
				builder.WriteString(extractTextValue(current))
			}
			return builder.String()
		}
		if choices, ok := typed["choices"].([]any); ok {
			var builder strings.Builder
			for _, current := range choices {
				builder.WriteString(extractTextValue(current))
			}
			return builder.String()
		}
		if output, ok := typed["output"].([]any); ok {
			var builder strings.Builder
			for _, current := range output {
				builder.WriteString(extractTextValue(current))
			}
			return builder.String()
		}
	case []any:
		var builder strings.Builder
		for _, current := range typed {
			builder.WriteString(extractTextValue(current))
		}
		return builder.String()
	case string:
		return typed
	}

	return ""
}
