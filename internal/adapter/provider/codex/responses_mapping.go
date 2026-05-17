package codex

import (
	"fmt"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
	responsesusecase "github.com/phamtanminhtien/goroute/internal/usecase/responses"
)

func parseResponsesSSE(data []byte) (openaiwire.ResponsesResponse, error) {
	return responsesusecase.ParseSSE(data)
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
