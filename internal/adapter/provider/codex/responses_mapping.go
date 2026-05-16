package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type responsesState struct {
	ID         string
	CreatedAt  int64
	Status     openaiwire.ResponsesStatus
	Model      string
	Usage      *openaiwire.ResponseUsage
	Error      *openaiwire.ResponseError
	ItemsByIdx map[int]openaiwire.OutputItem
}

func parseResponsesSSE(data []byte) (openaiwire.ResponsesResponse, error) {
	state := responsesState{
		CreatedAt:  time.Now().Unix(),
		Status:     openaiwire.ResponsesStatusInProgress,
		Usage:      &openaiwire.ResponseUsage{},
		ItemsByIdx: make(map[int]openaiwire.OutputItem),
	}

	for _, raw := range sseDataEvents(data) {
		if raw == "" || raw == "[DONE]" {
			continue
		}

		var event openaiwire.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return openaiwire.ResponsesResponse{}, fmt.Errorf("decode responses SSE event: %w", err)
		}
		processResponsesEvent(event, &state)
	}

	return finalizeResponsesState(state), nil
}

func processResponsesEvent(event openaiwire.ResponsesStreamEvent, state *responsesState) {
	if state == nil {
		return
	}

	if event.Response != nil {
		mergeResponseSnapshot(state, *event.Response)
	}

	switch event.Type {
	case openaiwire.ResponsesStreamEventTypeCreated:
		if event.Response != nil {
			mergeResponseSnapshot(state, *event.Response)
		}
	case openaiwire.ResponsesStreamEventTypeOutputItemDone:
		if event.Item != nil {
			state.ItemsByIdx[event.OutputIndex] = *event.Item
		}
	case openaiwire.ResponsesStreamEventTypeCompleted:
		state.Status = openaiwire.ResponsesStatusCompleted
		if event.Response != nil {
			mergeResponseSnapshot(state, *event.Response)
			if event.Response.Usage != nil {
				copied := *event.Response.Usage
				state.Usage = &copied
			}
		}
	case openaiwire.ResponsesStreamEventTypeFailed:
		state.Status = openaiwire.ResponsesStatusFailed
		if event.Response != nil && event.Response.Error != nil {
			copied := *event.Response.Error
			state.Error = &copied
		}
	}
}

func mergeResponseSnapshot(state *responsesState, response openaiwire.ResponsesResponse) {
	if response.ID != "" {
		state.ID = response.ID
	}
	if response.CreatedAt != 0 {
		state.CreatedAt = response.CreatedAt
	}
	if response.Status != "" {
		state.Status = response.Status
	}
	if response.Model != "" {
		state.Model = response.Model
	}
	if response.Usage != nil {
		copied := *response.Usage
		state.Usage = &copied
	}
	if response.Error != nil {
		copied := *response.Error
		state.Error = &copied
	}
	if len(response.Output) > 0 {
		for index, item := range response.Output {
			state.ItemsByIdx[index] = item
		}
	}
}

func finalizeResponsesState(state responsesState) openaiwire.ResponsesResponse {
	indexes := make([]int, 0, len(state.ItemsByIdx))
	for index := range state.ItemsByIdx {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	output := make([]openaiwire.OutputItem, 0, len(indexes))
	for _, index := range indexes {
		output = append(output, state.ItemsByIdx[index])
	}

	return openaiwire.ResponsesResponse{
		ID:        state.ID,
		Object:    "response",
		CreatedAt: state.CreatedAt,
		Status:    state.Status,
		Model:     state.Model,
		Output:    output,
		Usage:     state.Usage,
		Error:     state.Error,
	}
}

func responseToChatCompletion(response openaiwire.ResponsesResponse, model string) openaiwire.ChatCompletionsResponse {
	content := finalAssistantText(response.Output)
	toolCalls := responseToolCalls(response.Output)
	choice := openaiwire.ChatCompletionChoice{
		Index: 0,
		Message: openaiwire.Message{
			Role:    openaiwire.ChatRoleAssistant,
			Content: content,
		},
		FinishReason: openaiwire.FinishReasonStop,
	}
	if len(toolCalls) > 0 {
		choice.Message.ToolCalls = toolCalls
		choice.FinishReason = openaiwire.FinishReasonToolCalls
		choice.Message.Content = ""
	}

	out := openaiwire.ChatCompletionsResponse{
		ID:      chatCompletionID(response.ID),
		Object:  "chat.completion",
		Created: response.CreatedAt,
		Model:   model,
		Choices: []openaiwire.ChatCompletionChoice{choice},
	}
	if response.Usage != nil {
		out.Usage = &openaiwire.CompletionUsage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}
		if response.Usage.InputTokensDetails != nil {
			out.Usage.PromptTokensDetails = &openaiwire.PromptTokensDetails{
				CachedTokens: response.Usage.InputTokensDetails.CachedTokens,
			}
		}
	}
	return out
}

func finalAssistantText(output []openaiwire.OutputItem) string {
	for i := len(output) - 1; i >= 0; i-- {
		item := output[i]
		if item.Type != openaiwire.OutputItemTypeMessage || item.Role != string(openaiwire.ChatRoleAssistant) {
			continue
		}
		if text := outputText(item.Content); text != "" {
			return text
		}
	}
	return ""
}

func outputText(content []openaiwire.OutputContent) string {
	var builder strings.Builder
	for _, part := range content {
		if part.Type == openaiwire.OutputContentTypeOutputText || part.Type == "" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

func responseToolCalls(output []openaiwire.OutputItem) []openaiwire.ToolCall {
	toolCalls := make([]openaiwire.ToolCall, 0)
	for index, item := range output {
		if item.Type != openaiwire.OutputItemTypeFunctionCall {
			continue
		}
		toolCalls = append(toolCalls, openaiwire.ToolCall{
			ID:   defaultString(item.CallID, fmt.Sprintf("call_%d", index)),
			Type: openaiwire.ToolTypeFunction,
			Function: openaiwire.ToolCallFunction{
				Name:      item.Name,
				Arguments: defaultString(item.Arguments, "{}"),
			},
		})
	}
	return toolCalls
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

func chatCompletionID(responseID string) string {
	if responseID == "" {
		return ""
	}
	if strings.HasPrefix(responseID, "chatcmpl-") {
		return responseID
	}
	if strings.HasPrefix(responseID, "resp_") {
		return "chatcmpl-" + strings.TrimPrefix(responseID, "resp_")
	}
	return "chatcmpl-" + responseID
}
