package protocoltranslator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

const (
	responsesEventOutputItemAdded        = "response.output_item.added"
	responsesEventFunctionArgsDelta      = "response.function_call_arguments.delta"
	responsesEventFunctionArgsDone       = "response.function_call_arguments.done"
	responsesInputTypeMessage            = "message"
	responsesInputTypeFunctionCall       = "function_call"
	responsesInputTypeFunctionCallOutput = "function_call_output"
	responsesInputContentTypeInputText   = "input_text"
	responsesInputContentTypeOutputText  = "output_text"
	responsesInputContentTypeInputImage  = "input_image"
	chatCompletionsStreamObject          = "chat.completion.chunk"
	chatCompletionsObject                = "chat.completion"
	defaultAssistantToolCallIndex        = 0
)

func ChatCompletionsToResponses(req openaiwire.ChatCompletionsRequest) (openaiwire.ResponsesRequest, error) {
	instructions := make([]string, 0, len(req.Messages))
	input := make([]openaiwire.ResponseInputItem, 0, len(req.Messages))

	for index, message := range req.Messages {
		switch message.Role {
		case openaiwire.ChatRoleSystem:
			text, err := chatMessageText(message.Content, true)
			if err != nil {
				return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].content: %w", index, err)
			}
			if strings.TrimSpace(text) != "" {
				instructions = append(instructions, text)
			}
		case openaiwire.ChatRoleUser, openaiwire.ChatRoleAssistant:
			content, err := translateChatMessageContent(message.Role, message.Content)
			if err != nil {
				return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].content: %w", index, err)
			}
			if len(content) > 0 {
				input = append(input, openaiwire.ResponseInputItem{
					Type:    responsesInputTypeMessage,
					Role:    string(message.Role),
					Content: content,
				})
			}
			if message.Role == openaiwire.ChatRoleAssistant {
				for toolIndex, toolCall := range message.ToolCalls {
					if toolCall.Function.Name == "" {
						return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].tool_calls[%d].function.name is required", index, toolIndex)
					}
					input = append(input, openaiwire.ResponseInputItem{
						Type:      responsesInputTypeFunctionCall,
						CallID:    toolCall.ID,
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					})
				}
			}
		case openaiwire.ChatRoleTool:
			if strings.TrimSpace(message.ToolCallID) == "" {
				return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].tool_call_id is required for tool messages", index)
			}
			text, err := chatMessageText(message.Content, false)
			if err != nil {
				return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].content: %w", index, err)
			}
			input = append(input, openaiwire.ResponseInputItem{
				Type:   responsesInputTypeFunctionCallOutput,
				CallID: message.ToolCallID,
				Output: text,
			})
		default:
			return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d].role %q is not supported", index, message.Role)
		}
	}

	reasoning, err := translateReasoning(req.Reasoning, req.ReasoningEffort)
	if err != nil {
		return openaiwire.ResponsesRequest{}, err
	}

	payload := map[string]any{
		"model":  req.Model,
		"input":  input,
		"stream": req.Stream,
	}
	if joined := strings.Join(instructions, "\n"); strings.TrimSpace(joined) != "" {
		payload["instructions"] = joined
	}
	if reasoning != nil {
		payload["reasoning"] = reasoning
	}
	if len(req.Tools) > 0 {
		tools, err := translateToolsToResponses(req.Tools)
		if err != nil {
			return openaiwire.ResponsesRequest{}, err
		}
		payload["tools"] = tools
	}
	if req.ToolChoice != nil {
		toolChoice, err := translateToolChoiceToResponses(req.ToolChoice)
		if err != nil {
			return openaiwire.ResponsesRequest{}, err
		}
		payload["tool_choice"] = toolChoice
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return openaiwire.ResponsesRequest{}, fmt.Errorf("marshal translated responses payload: %w", err)
	}

	return openaiwire.ResponsesRequest{
		Model:        req.Model,
		Instructions: strings.Join(instructions, "\n"),
		Input:        input,
		Stream:       req.Stream,
		Reasoning:    reasoning,
		RawBody:      rawBody,
	}, nil
}

func ResponsesToChatCompletions(req openaiwire.ResponsesRequest) (openaiwire.ChatCompletionsRequest, error) {
	if len(req.RawBody) > 0 {
		return responsesRawBodyToChatCompletions(req.RawBody)
	}

	out := openaiwire.ChatCompletionsRequest{
		Model:  req.Model,
		Stream: req.Stream,
	}
	if strings.TrimSpace(req.Instructions) != "" {
		out.Messages = append(out.Messages, openaiwire.ChatMessage{
			Role:    openaiwire.ChatRoleSystem,
			Content: openaiwire.TextContent(req.Instructions),
		})
	}
	if strings.TrimSpace(req.InputText) != "" {
		out.Messages = append(out.Messages, openaiwire.ChatMessage{
			Role:    openaiwire.ChatRoleUser,
			Content: openaiwire.TextContent(req.InputText),
		})
	}
	for index, item := range req.Input {
		messages, err := responseInputItemToChatMessages(item)
		if err != nil {
			return openaiwire.ChatCompletionsRequest{}, fmt.Errorf("input[%d]: %w", index, err)
		}
		out.Messages = append(out.Messages, messages...)
	}
	return out, nil
}

func ResponsesToChatCompletion(response openaiwire.ResponsesResponse) openaiwire.ChatCompletionsResponse {
	message := openaiwire.Message{Role: openaiwire.ChatRoleAssistant}
	toolCalls := make([]openaiwire.ToolCall, 0, len(response.Output))

	for _, item := range response.Output {
		switch item.Type {
		case openaiwire.OutputItemTypeMessage:
			if item.Role != "" && item.Role != string(openaiwire.ChatRoleAssistant) {
				continue
			}
			for _, content := range item.Content {
				if content.Type == openaiwire.OutputContentTypeOutputText {
					message.Content += content.Text
				}
			}
		case openaiwire.OutputItemTypeFunctionCall:
			toolCalls = append(toolCalls, openaiwire.ToolCall{
				ID:   defaultCallID(item.CallID, item.ID),
				Type: openaiwire.ToolTypeFunction,
				Function: openaiwire.ToolCallFunction{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		}
	}

	finishReason := openaiwire.FinishReasonStop
	if len(toolCalls) > 0 && message.Content == "" {
		finishReason = openaiwire.FinishReasonToolCalls
		message.ToolCalls = toolCalls
	}

	out := openaiwire.ChatCompletionsResponse{
		ID:      chatCompletionID(response.ID),
		Object:  chatCompletionsObject,
		Created: response.CreatedAt,
		Model:   response.Model,
		Choices: []openaiwire.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: finishReason,
		}},
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
	if response.Error != nil {
		out.Error = &openaiwire.OpenAIError{
			Message: response.Error.Message,
			Type:    response.Error.Type,
			Code:    response.Error.Code,
		}
	}
	return out
}

func ChatCompletionToResponses(response openaiwire.ChatCompletionsResponse) openaiwire.ResponsesResponse {
	output := make([]openaiwire.OutputItem, 0, len(response.Choices))
	for _, choice := range response.Choices {
		message := choice.Message
		if message.Content != "" {
			output = append(output, openaiwire.OutputItem{
				Type: openaiwire.OutputItemTypeMessage,
				Role: string(openaiwire.ChatRoleAssistant),
				Content: []openaiwire.OutputContent{{
					Type: openaiwire.OutputContentTypeOutputText,
					Text: message.Content,
				}},
			})
		}
		for _, toolCall := range message.ToolCalls {
			output = append(output, openaiwire.OutputItem{
				Type:      openaiwire.OutputItemTypeFunctionCall,
				CallID:    toolCall.ID,
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			})
		}
	}

	out := openaiwire.ResponsesResponse{
		ID:        response.ID,
		Object:    "response",
		CreatedAt: response.Created,
		Status:    openaiwire.ResponsesStatusCompleted,
		Model:     response.Model,
		Output:    output,
	}
	if response.Usage != nil {
		out.Usage = &openaiwire.ResponseUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
		if response.Usage.PromptTokensDetails != nil {
			out.Usage.InputTokensDetails = &openaiwire.ResponseInputTokenDetails{
				CachedTokens: response.Usage.PromptTokensDetails.CachedTokens,
			}
		}
	}
	if response.Error != nil {
		out.Status = openaiwire.ResponsesStatusFailed
		out.Error = &openaiwire.ResponseError{
			Message: response.Error.Message,
			Type:    response.Error.Type,
		}
		if code, ok := response.Error.Code.(string); ok {
			out.Error.Code = code
		}
	}
	return out
}

func translateChatMessageContent(role openaiwire.ChatRole, content openaiwire.ChatMessageContent) ([]openaiwire.ResponseInputContentPart, error) {
	textContentType := responsesInputContentTypeInputText
	if role == openaiwire.ChatRoleAssistant {
		textContentType = responsesInputContentTypeOutputText
	}

	if !content.IsParts() {
		if content.Text() == "" {
			return nil, nil
		}
		return []openaiwire.ResponseInputContentPart{{
			Type: textContentType,
			Text: content.Text(),
		}}, nil
	}

	parts := content.Parts()
	out := make([]openaiwire.ResponseInputContentPart, 0, len(parts))
	for index, part := range parts {
		switch part.Type {
		case "text":
			out = append(out, openaiwire.ResponseInputContentPart{
				Type: textContentType,
				Text: part.Text,
			})
		case "image_url":
			if part.ImageURL == nil || part.ImageURL.URL == "" {
				return nil, fmt.Errorf("content[%d].image_url.url is required", index)
			}
			out = append(out, openaiwire.ResponseInputContentPart{
				Type:     responsesInputContentTypeInputImage,
				ImageURL: part.ImageURL.URL,
				Detail:   part.ImageURL.Detail,
			})
		default:
			return nil, fmt.Errorf("content[%d].type %q is not supported", index, part.Type)
		}
	}
	return out, nil
}

func chatMessageText(content openaiwire.ChatMessageContent, allowImages bool) (string, error) {
	if !content.IsParts() {
		return content.Text(), nil
	}

	parts := content.Parts()
	var builder strings.Builder
	for index, part := range parts {
		switch part.Type {
		case "text":
			builder.WriteString(part.Text)
		case "image_url":
			if allowImages {
				continue
			}
			return "", fmt.Errorf("content[%d].type %q is not supported here", index, part.Type)
		default:
			return "", fmt.Errorf("content[%d].type %q is not supported", index, part.Type)
		}
	}
	return builder.String(), nil
}

func translateReasoning(raw json.RawMessage, effort string) (any, error) {
	if len(raw) == 0 && strings.TrimSpace(effort) == "" {
		return nil, nil
	}
	if len(raw) == 0 {
		return map[string]any{"effort": effort}, nil
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid reasoning payload: %w", err)
	}
	if strings.TrimSpace(effort) == "" {
		return payload, nil
	}

	object, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("reasoning_effort requires reasoning to be a JSON object")
	}
	object["effort"] = effort
	return object, nil
}

func translateToolsToResponses(tools []openaiwire.Tool) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(tools))
	for index, tool := range tools {
		if tool.Type != openaiwire.ToolTypeFunction {
			return nil, fmt.Errorf("tools[%d].type %q is not supported", index, tool.Type)
		}
		if strings.TrimSpace(tool.Function.Name) == "" {
			return nil, fmt.Errorf("tools[%d].function.name is required", index)
		}
		item := map[string]any{
			"type": "function",
			"name": tool.Function.Name,
		}
		if tool.Function.Description != "" {
			item["description"] = tool.Function.Description
		}
		if len(tool.Function.Parameters) > 0 {
			var parameters any
			if err := json.Unmarshal(tool.Function.Parameters, &parameters); err != nil {
				return nil, fmt.Errorf("tools[%d].function.parameters: %w", index, err)
			}
			item["parameters"] = parameters
		}
		if tool.Function.Strict {
			item["strict"] = true
		}
		out = append(out, item)
	}
	return out, nil
}

func translateToolChoiceToResponses(choice any) (any, error) {
	switch typed := choice.(type) {
	case string:
		switch typed {
		case "auto", "none", "required":
			return typed, nil
		default:
			return nil, fmt.Errorf("tool_choice %q is not supported", typed)
		}
	case map[string]any:
		choiceType, _ := typed["type"].(string)
		if choiceType != "function" {
			return nil, fmt.Errorf("tool_choice.type %q is not supported", choiceType)
		}
		if name, ok := typed["name"].(string); ok && strings.TrimSpace(name) != "" {
			return map[string]any{"type": "function", "name": name}, nil
		}
		functionMap, _ := typed["function"].(map[string]any)
		if functionMap == nil {
			return nil, fmt.Errorf("tool_choice.function.name is required")
		}
		name, _ := functionMap["name"].(string)
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("tool_choice.function.name is required")
		}
		return map[string]any{"type": "function", "name": name}, nil
	default:
		return nil, fmt.Errorf("tool_choice type %T is not supported", choice)
	}
}

func responsesRawBodyToChatCompletions(raw json.RawMessage) (openaiwire.ChatCompletionsRequest, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return openaiwire.ChatCompletionsRequest{}, err
	}

	var req openaiwire.ResponsesRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return openaiwire.ChatCompletionsRequest{}, err
	}
	req.RawBody = nil
	out, err := ResponsesToChatCompletions(req)
	if err != nil {
		return openaiwire.ChatCompletionsRequest{}, err
	}

	if rawTools := payload["tools"]; len(rawTools) > 0 {
		tools, err := translateRawResponsesToolsToChat(rawTools)
		if err != nil {
			return openaiwire.ChatCompletionsRequest{}, err
		}
		out.Tools = tools
	}
	if rawChoice := payload["tool_choice"]; len(rawChoice) > 0 {
		choice, err := translateRawResponsesToolChoiceToChat(rawChoice)
		if err != nil {
			return openaiwire.ChatCompletionsRequest{}, err
		}
		out.ToolChoice = choice
	}
	return out, nil
}

func responseInputItemToChatMessages(item openaiwire.ResponseInputItem) ([]openaiwire.ChatMessage, error) {
	switch item.Type {
	case "", responsesInputTypeMessage:
		role := openaiwire.ChatRole(item.Role)
		if role == "" {
			role = openaiwire.ChatRoleUser
		}
		content, err := responseInputContentToChat(role, item.Content)
		if err != nil {
			return nil, err
		}
		return []openaiwire.ChatMessage{{Role: role, Content: content}}, nil
	case responsesInputTypeFunctionCall:
		return []openaiwire.ChatMessage{{
			Role:    openaiwire.ChatRoleAssistant,
			Content: openaiwire.TextContent(""),
			ToolCalls: []openaiwire.ToolCall{{
				ID:   item.CallID,
				Type: openaiwire.ToolTypeFunction,
				Function: openaiwire.ToolCallFunction{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			}},
		}}, nil
	case responsesInputTypeFunctionCallOutput:
		return []openaiwire.ChatMessage{{
			Role:       openaiwire.ChatRoleTool,
			ToolCallID: item.CallID,
			Content:    openaiwire.TextContent(item.Output),
		}}, nil
	case "reasoning":
		return nil, nil
	default:
		return nil, fmt.Errorf("type %q is not supported by chat completions translation", item.Type)
	}
}

func responseInputContentToChat(role openaiwire.ChatRole, parts []openaiwire.ResponseInputContentPart) (openaiwire.ChatMessageContent, error) {
	if len(parts) == 0 {
		return openaiwire.TextContent(""), nil
	}
	if len(parts) == 1 && isResponsesTextContent(role, parts[0].Type) {
		return openaiwire.TextContent(parts[0].Text), nil
	}

	out := make([]openaiwire.ChatMessageContentPart, 0, len(parts))
	for index, part := range parts {
		switch part.Type {
		case responsesInputContentTypeInputText, responsesInputContentTypeOutputText:
			if !isResponsesTextContent(role, part.Type) {
				return openaiwire.ChatMessageContent{}, fmt.Errorf("content[%d].type %q is not supported for role %q by chat completions translation", index, part.Type, role)
			}
			out = append(out, openaiwire.ChatMessageContentPart{Type: "text", Text: part.Text})
		case responsesInputContentTypeInputImage:
			if part.ImageURL == "" {
				return openaiwire.ChatMessageContent{}, fmt.Errorf("content[%d].image_url is required", index)
			}
			out = append(out, openaiwire.ChatMessageContentPart{
				Type: "image_url",
				ImageURL: &openaiwire.ChatMessageImageURL{
					URL:    part.ImageURL,
					Detail: part.Detail,
				},
			})
		default:
			return openaiwire.ChatMessageContent{}, fmt.Errorf("content[%d].type %q is not supported by chat completions translation", index, part.Type)
		}
	}
	return openaiwire.PartsContent(out...), nil
}

func isResponsesTextContent(role openaiwire.ChatRole, contentType string) bool {
	switch role {
	case openaiwire.ChatRoleAssistant:
		return contentType == responsesInputContentTypeOutputText
	default:
		return contentType == responsesInputContentTypeInputText
	}
}

func translateRawResponsesToolsToChat(raw json.RawMessage) ([]openaiwire.Tool, error) {
	var tools []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil, err
	}
	out := make([]openaiwire.Tool, 0, len(tools))
	for index, tool := range tools {
		var toolType string
		_ = json.Unmarshal(tool["type"], &toolType)
		if toolType != "function" {
			return nil, fmt.Errorf("tools[%d].type %q is not supported", index, toolType)
		}
		var fn openaiwire.ToolFunction
		if err := json.Unmarshal(tool["name"], &fn.Name); err != nil || strings.TrimSpace(fn.Name) == "" {
			return nil, fmt.Errorf("tools[%d].name is required", index)
		}
		_ = json.Unmarshal(tool["description"], &fn.Description)
		_ = json.Unmarshal(tool["parameters"], &fn.Parameters)
		_ = json.Unmarshal(tool["strict"], &fn.Strict)
		out = append(out, openaiwire.Tool{Type: openaiwire.ToolTypeFunction, Function: fn})
	}
	return out, nil
}

func translateRawResponsesToolChoiceToChat(raw json.RawMessage) (any, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}
	var choice map[string]any
	if err := json.Unmarshal(raw, &choice); err != nil {
		return nil, err
	}
	if choice["type"] != "function" {
		return nil, fmt.Errorf("tool_choice.type %q is not supported", choice["type"])
	}
	name, _ := choice["name"].(string)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("tool_choice.name is required")
	}
	return map[string]any{"type": "function", "function": map[string]any{"name": name}}, nil
}

type responsesParseState struct {
	ID         string
	CreatedAt  int64
	Status     openaiwire.ResponsesStatus
	Model      string
	Usage      *openaiwire.ResponseUsage
	Error      *openaiwire.ResponseError
	ItemsByIdx map[int]openaiwire.OutputItem
}

func ParseResponsesSSE(data []byte) (openaiwire.ResponsesResponse, error) {
	current := responsesParseState{
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
		processResponsesEvent(event, &current)
	}

	return finalizeResponses(current), nil
}

func processResponsesEvent(event openaiwire.ResponsesStreamEvent, current *responsesParseState) {
	if current == nil {
		return
	}
	if event.Response != nil {
		mergeResponsesSnapshot(current, *event.Response)
	}

	switch event.Type {
	case openaiwire.ResponsesStreamEventTypeCreated:
		if event.Response != nil {
			mergeResponsesSnapshot(current, *event.Response)
		}
	case openaiwire.ResponsesStreamEventTypeOutputItemDone:
		if event.Item != nil {
			current.ItemsByIdx[event.OutputIndex] = *event.Item
		}
	case openaiwire.ResponsesStreamEventTypeCompleted:
		current.Status = openaiwire.ResponsesStatusCompleted
		if event.Response != nil {
			mergeResponsesSnapshot(current, *event.Response)
			if event.Response.Usage != nil {
				copied := *event.Response.Usage
				current.Usage = &copied
			}
		}
	case openaiwire.ResponsesStreamEventTypeFailed:
		current.Status = openaiwire.ResponsesStatusFailed
		if event.Response != nil && event.Response.Error != nil {
			copied := *event.Response.Error
			current.Error = &copied
		}
	}
}

func mergeResponsesSnapshot(current *responsesParseState, response openaiwire.ResponsesResponse) {
	if response.ID != "" {
		current.ID = response.ID
	}
	if response.CreatedAt != 0 {
		current.CreatedAt = response.CreatedAt
	}
	if response.Status != "" {
		current.Status = response.Status
	}
	if response.Model != "" {
		current.Model = response.Model
	}
	if response.Usage != nil {
		copied := *response.Usage
		current.Usage = &copied
	}
	if response.Error != nil {
		copied := *response.Error
		current.Error = &copied
	}
	if len(response.Output) > 0 {
		for index, item := range response.Output {
			current.ItemsByIdx[index] = item
		}
	}
}

func finalizeResponses(current responsesParseState) openaiwire.ResponsesResponse {
	indexes := make([]int, 0, len(current.ItemsByIdx))
	for index := range current.ItemsByIdx {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	output := make([]openaiwire.OutputItem, 0, len(indexes))
	for _, index := range indexes {
		output = append(output, current.ItemsByIdx[index])
	}

	return openaiwire.ResponsesResponse{
		ID:        current.ID,
		Object:    "response",
		CreatedAt: current.CreatedAt,
		Status:    current.Status,
		Model:     current.Model,
		Output:    output,
		Usage:     current.Usage,
		Error:     current.Error,
	}
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

type chatStreamState struct {
	id                   string
	created              int64
	model                string
	roleEmitted          bool
	textEmitted          bool
	toolCallEmitted      bool
	terminalSent         bool
	textOutputIndexes    map[int]bool
	toolCallsByKey       map[string]toolCallState
	toolCallOrder        []string
	outputIndexToToolKey map[int]string
	doneSent             bool
}

type toolCallState struct {
	ID           string
	Name         string
	Arguments    string
	Index        int
	MetadataSent bool
}

func ResponsesStreamToChatCompletions(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer body.Close()
		defer writer.Close()

		state := chatStreamState{
			textOutputIndexes:    make(map[int]bool),
			toolCallsByKey:       make(map[string]toolCallState),
			outputIndexToToolKey: make(map[int]string),
		}

		buffered := bufio.NewScanner(body)
		buffered.Buffer(make([]byte, 0, 1024), 1024*1024)
		dataLines := make([]string, 0, 4)
		sawPayload := false

		flushEvent := func() error {
			if len(dataLines) == 0 {
				return nil
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]
			if strings.TrimSpace(payload) != "" {
				sawPayload = true
			}
			return state.writeTranslatedEvent(writer, payload)
		}

		for buffered.Scan() {
			line := buffered.Text()
			if line == "" {
				if err := flushEvent(); err != nil {
					_ = writer.CloseWithError(err)
					return
				}
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if err := buffered.Err(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		if err := flushEvent(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		if sawPayload {
			if err := state.writeDone(writer); err != nil {
				_ = writer.CloseWithError(err)
			}
		}
	}()

	return reader
}

func (s *chatStreamState) writeTranslatedEvent(w io.Writer, payload string) error {
	if strings.TrimSpace(payload) == "" {
		return nil
	}
	if payload == "[DONE]" {
		return s.writeDone(w)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return fmt.Errorf("decode responses stream event: %w", err)
	}

	s.captureResponseEnvelope(envelope)

	var eventType string
	if rawType := envelope["type"]; len(rawType) > 0 {
		if err := json.Unmarshal(rawType, &eventType); err != nil {
			return fmt.Errorf("decode responses stream event type: %w", err)
		}
	}

	switch eventType {
	case string(openaiwire.ResponsesStreamEventTypeOutputTextDelta):
		text := decodeStringField(envelope, "delta")
		if text == "" {
			text = decodeStringField(envelope, "text")
		}
		if text == "" {
			return nil
		}
		s.textEmitted = true
		return s.writeChunk(w, openaiwire.ChatCompletionsStreamChunk{
			ID:      s.id,
			Object:  chatCompletionsStreamObject,
			Created: s.created,
			Model:   s.model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index: 0,
				Delta: openaiwire.Message{
					Role:    s.consumeRole(),
					Content: text,
				},
			}},
		})
	case responsesEventFunctionArgsDelta:
		return s.writeFunctionArgumentDelta(w, envelope)
	case responsesEventFunctionArgsDone:
		return nil
	case responsesEventOutputItemAdded:
		return s.writeOutputItemEvent(w, envelope, false)
	case string(openaiwire.ResponsesStreamEventTypeOutputItemDone):
		return s.writeOutputItemEvent(w, envelope, true)
	case string(openaiwire.ResponsesStreamEventTypeCompleted):
		return s.writeTerminalChunk(w, envelope["response"])
	default:
		return nil
	}
}

func (s *chatStreamState) writeDone(w io.Writer) error {
	if s.doneSent {
		return nil
	}
	s.doneSent = true
	if !s.terminalSent {
		if err := s.writeTerminalChunk(w, nil); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "data: [DONE]\n\n")
	return err
}

func (s *chatStreamState) writeOutputItemEvent(w io.Writer, envelope map[string]json.RawMessage, itemDone bool) error {
	rawItem := envelope["item"]
	if len(rawItem) == 0 {
		return nil
	}

	var item openaiwire.OutputItem
	if err := json.Unmarshal(rawItem, &item); err != nil {
		return fmt.Errorf("decode responses output item: %w", err)
	}

	outputIndex := decodeIntField(envelope, "output_index")
	switch item.Type {
	case openaiwire.OutputItemTypeMessage:
		if item.Role != "" && item.Role != string(openaiwire.ChatRoleAssistant) {
			return nil
		}
		if s.textOutputIndexes[outputIndex] || s.textEmitted {
			return nil
		}
		text := item.TextValue()
		if text == "" {
			return nil
		}
		s.textOutputIndexes[outputIndex] = true
		s.textEmitted = true
		return s.writeChunk(w, openaiwire.ChatCompletionsStreamChunk{
			ID:      s.id,
			Object:  chatCompletionsStreamObject,
			Created: s.created,
			Model:   s.model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index: 0,
				Delta: openaiwire.Message{
					Role:    s.consumeRole(),
					Content: text,
				},
			}},
		})
	case openaiwire.OutputItemTypeFunctionCall:
		key := s.registerToolCall(item, outputIndex)
		state := s.toolCallsByKey[key]
		if itemDone && (state.MetadataSent || state.Arguments != "") {
			return nil
		}
		if state.MetadataSent && item.Arguments == "" {
			return nil
		}
		state.MetadataSent = true
		state.Arguments = item.Arguments
		s.toolCallsByKey[key] = state
		s.toolCallEmitted = true
		return s.writeChunk(w, openaiwire.ChatCompletionsStreamChunk{
			ID:      s.id,
			Object:  chatCompletionsStreamObject,
			Created: s.created,
			Model:   s.model,
			Choices: []openaiwire.ChatCompletionsStreamChoice{{
				Index: 0,
				Delta: openaiwire.Message{
					Role: s.consumeRole(),
					ToolCalls: []openaiwire.ToolCall{{
						ID:   state.ID,
						Type: openaiwire.ToolTypeFunction,
						Function: openaiwire.ToolCallFunction{
							Name:      state.Name,
							Arguments: item.Arguments,
						},
					}},
				},
			}},
		})
	default:
		return nil
	}
}

func (s *chatStreamState) writeFunctionArgumentDelta(w io.Writer, envelope map[string]json.RawMessage) error {
	delta := decodeStringField(envelope, "delta")
	if delta == "" {
		return nil
	}

	outputIndex := decodeIntField(envelope, "output_index")
	key := decodeStringField(envelope, "call_id")
	if key == "" {
		key = decodeStringField(envelope, "item_id")
	}
	if key == "" && outputIndex >= 0 {
		key = s.outputIndexToToolKey[outputIndex]
	}

	if rawItem := envelope["item"]; len(rawItem) > 0 {
		var item openaiwire.OutputItem
		if err := json.Unmarshal(rawItem, &item); err == nil {
			key = s.registerToolCall(item, outputIndex)
		}
	}

	if key == "" {
		key = "tool_" + strconv.Itoa(defaultAssistantToolCallIndex)
	}

	state, ok := s.toolCallsByKey[key]
	if !ok {
		state = toolCallState{
			ID:    key,
			Index: len(s.toolCallOrder),
		}
		s.toolCallsByKey[key] = state
		s.toolCallOrder = append(s.toolCallOrder, key)
	}
	state.Arguments += delta
	s.toolCallsByKey[key] = state
	s.toolCallEmitted = true

	return s.writeChunk(w, openaiwire.ChatCompletionsStreamChunk{
		ID:      s.id,
		Object:  chatCompletionsStreamObject,
		Created: s.created,
		Model:   s.model,
		Choices: []openaiwire.ChatCompletionsStreamChoice{{
			Index: 0,
			Delta: openaiwire.Message{
				Role: s.consumeRole(),
				ToolCalls: []openaiwire.ToolCall{{
					ID:   state.ID,
					Type: openaiwire.ToolTypeFunction,
					Function: openaiwire.ToolCallFunction{
						Name:      state.Name,
						Arguments: delta,
					},
				}},
			},
		}},
	})
}

func (s *chatStreamState) writeTerminalChunk(w io.Writer, rawResponse json.RawMessage) error {
	if s.terminalSent {
		return nil
	}
	s.terminalSent = true

	var usage *openaiwire.CompletionUsage
	if len(rawResponse) > 0 {
		var response openaiwire.ResponsesResponse
		if err := json.Unmarshal(rawResponse, &response); err == nil {
			s.captureResponse(response)
			if response.Usage != nil {
				usage = &openaiwire.CompletionUsage{
					PromptTokens:     response.Usage.InputTokens,
					CompletionTokens: response.Usage.OutputTokens,
					TotalTokens:      response.Usage.TotalTokens,
				}
				if response.Usage.InputTokensDetails != nil {
					usage.PromptTokensDetails = &openaiwire.PromptTokensDetails{
						CachedTokens: response.Usage.InputTokensDetails.CachedTokens,
					}
				}
			}
		}
	}

	finishReason := openaiwire.FinishReasonStop
	if s.toolCallEmitted && !s.textEmitted {
		finishReason = openaiwire.FinishReasonToolCalls
	}

	return s.writeChunk(w, openaiwire.ChatCompletionsStreamChunk{
		ID:      s.id,
		Object:  chatCompletionsStreamObject,
		Created: s.created,
		Model:   s.model,
		Choices: []openaiwire.ChatCompletionsStreamChoice{{
			Index:        0,
			Delta:        openaiwire.Message{},
			FinishReason: finishReason,
		}},
		Usage: usage,
	})
}

func (s *chatStreamState) writeChunk(w io.Writer, chunk openaiwire.ChatCompletionsStreamChunk) error {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("marshal translated chat stream chunk: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

func (s *chatStreamState) captureResponseEnvelope(envelope map[string]json.RawMessage) {
	rawResponse := envelope["response"]
	if len(rawResponse) == 0 {
		return
	}
	var response openaiwire.ResponsesResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return
	}
	s.captureResponse(response)
}

func (s *chatStreamState) captureResponse(response openaiwire.ResponsesResponse) {
	if response.ID != "" {
		s.id = chatCompletionID(response.ID)
	}
	if response.CreatedAt != 0 {
		s.created = response.CreatedAt
	}
	if response.Model != "" {
		s.model = response.Model
	}
}

func (s *chatStreamState) consumeRole() openaiwire.ChatRole {
	if s.roleEmitted {
		return ""
	}
	s.roleEmitted = true
	return openaiwire.ChatRoleAssistant
}

func (s *chatStreamState) registerToolCall(item openaiwire.OutputItem, outputIndex int) string {
	key := defaultCallID(item.CallID, item.ID)
	if key == "" {
		key = "tool_" + strconv.Itoa(outputIndex)
	}

	state, exists := s.toolCallsByKey[key]
	if !exists {
		state = toolCallState{
			ID:    key,
			Index: len(s.toolCallOrder),
		}
		s.toolCallOrder = append(s.toolCallOrder, key)
	}
	if item.Name != "" {
		state.Name = item.Name
	}
	if item.Arguments != "" {
		state.Arguments = item.Arguments
	}
	s.toolCallsByKey[key] = state
	if outputIndex >= 0 {
		s.outputIndexToToolKey[outputIndex] = key
	}
	return key
}

func ChatCompletionsStreamToResponses(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer body.Close()
		defer writer.Close()

		buffered := bufio.NewScanner(body)
		buffered.Buffer(make([]byte, 0, 1024), 1024*1024)
		var responseID string
		var created int64
		var model string
		var content strings.Builder
		outputIndex := 0
		createdSent := false
		dataLines := make([]string, 0, 4)

		writeEvent := func(eventType string, payload any) error {
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", eventType, data)
			return err
		}

		flushEvent := func() error {
			if len(dataLines) == 0 {
				return nil
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]
			if payload == "[DONE]" || strings.TrimSpace(payload) == "" {
				return nil
			}

			var chunk openaiwire.ChatCompletionsStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				return fmt.Errorf("decode chat completions stream chunk: %w", err)
			}
			if chunk.ID != "" {
				responseID = chunk.ID
			}
			if chunk.Created != 0 {
				created = chunk.Created
			}
			if chunk.Model != "" {
				model = chunk.Model
			}
			if !createdSent {
				createdSent = true
				if created == 0 {
					created = time.Now().Unix()
				}
				if err := writeEvent(string(openaiwire.ResponsesStreamEventTypeCreated), openaiwire.ResponsesStreamEvent{
					Type: openaiwire.ResponsesStreamEventTypeCreated,
					Response: &openaiwire.ResponsesResponse{
						ID:        responseID,
						Object:    "response",
						CreatedAt: created,
						Status:    openaiwire.ResponsesStatusInProgress,
						Model:     model,
						Output:    []openaiwire.OutputItem{},
					},
				}); err != nil {
					return err
				}
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					content.WriteString(choice.Delta.Content)
					if err := writeEvent(string(openaiwire.ResponsesStreamEventTypeOutputTextDelta), openaiwire.ResponsesStreamEvent{
						Type:        openaiwire.ResponsesStreamEventTypeOutputTextDelta,
						Delta:       choice.Delta.Content,
						OutputIndex: outputIndex,
					}); err != nil {
						return err
					}
				}
				if choice.FinishReason != "" {
					item := openaiwire.OutputItem{
						Type: openaiwire.OutputItemTypeMessage,
						Role: string(openaiwire.ChatRoleAssistant),
						Content: []openaiwire.OutputContent{{
							Type: openaiwire.OutputContentTypeOutputText,
							Text: content.String(),
						}},
					}
					if err := writeEvent(string(openaiwire.ResponsesStreamEventTypeOutputItemDone), openaiwire.ResponsesStreamEvent{
						Type:        openaiwire.ResponsesStreamEventTypeOutputItemDone,
						OutputIndex: outputIndex,
						Item:        &item,
					}); err != nil {
						return err
					}
					response := ChatCompletionToResponses(openaiwire.ChatCompletionsResponse{
						ID:      responseID,
						Object:  chatCompletionsObject,
						Created: created,
						Model:   model,
						Choices: []openaiwire.ChatChoice{{
							Index: 0,
							Message: openaiwire.Message{
								Role:    openaiwire.ChatRoleAssistant,
								Content: content.String(),
							},
							FinishReason: openaiwire.FinishReasonStop,
						}},
						Usage: chunk.Usage,
					})
					if err := writeEvent(string(openaiwire.ResponsesStreamEventTypeCompleted), openaiwire.ResponsesStreamEvent{
						Type:     openaiwire.ResponsesStreamEventTypeCompleted,
						Response: &response,
					}); err != nil {
						return err
					}
				}
			}
			return nil
		}

		for buffered.Scan() {
			line := buffered.Text()
			if line == "" {
				if err := flushEvent(); err != nil {
					_ = writer.CloseWithError(err)
					return
				}
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if err := buffered.Err(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		if err := flushEvent(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}()

	return reader
}

func RewriteChatCompletionsStreamModel(body io.ReadCloser, model string) io.ReadCloser {
	return rewriteSSEData(body, func(payload string) (string, bool) {
		if payload == "[DONE]" {
			return payload, true
		}

		var chunk openaiwire.ChatCompletionsStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return payload, false
		}
		chunk.Model = model

		encoded, err := json.Marshal(chunk)
		if err != nil {
			return payload, false
		}
		return string(encoded), true
	})
}

func RewriteResponsesStreamModel(body io.ReadCloser, model string) io.ReadCloser {
	return rewriteSSEData(body, func(payload string) (string, bool) {
		if payload == "[DONE]" {
			return payload, true
		}

		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return payload, false
		}

		rawResponse, ok := event["response"]
		if !ok {
			return payload, true
		}

		var response map[string]json.RawMessage
		if err := json.Unmarshal(rawResponse, &response); err != nil {
			return payload, false
		}

		encodedModel, err := json.Marshal(model)
		if err != nil {
			return payload, false
		}
		response["model"] = encodedModel

		encodedResponse, err := json.Marshal(response)
		if err != nil {
			return payload, false
		}
		event["response"] = encodedResponse

		encoded, err := json.Marshal(event)
		if err != nil {
			return payload, false
		}
		return string(encoded), true
	})
}

func rewriteSSEData(body io.ReadCloser, rewrite func(string) (string, bool)) io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer body.Close()
		defer writer.Close()

		buffered := bufio.NewReader(body)
		for {
			line, err := buffered.ReadString('\n')
			if err != nil && err != io.EOF {
				_ = writer.CloseWithError(err)
				return
			}
			if line == "" && err == io.EOF {
				return
			}

			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if nextPayload, ok := rewrite(payload); ok {
					line = "data: " + nextPayload
				}
			}
			if _, err := fmt.Fprintln(writer, line); err != nil {
				_ = writer.CloseWithError(err)
				return
			}

			if err == io.EOF {
				return
			}
		}
	}()

	return reader
}

func defaultCallID(callID string, fallback string) string {
	if strings.TrimSpace(callID) != "" {
		return callID
	}
	return strings.TrimSpace(fallback)
}

func chatCompletionID(responseID string) string {
	trimmed := strings.TrimSpace(responseID)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "chatcmpl-") {
		return trimmed
	}
	for _, prefix := range []string{"resp_", "resp-"} {
		if strings.HasPrefix(trimmed, prefix) {
			return "chatcmpl-" + strings.TrimPrefix(trimmed, prefix)
		}
	}
	return "chatcmpl-" + trimmed
}

func decodeStringField(fields map[string]json.RawMessage, key string) string {
	raw := fields[key]
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func decodeIntField(fields map[string]json.RawMessage, key string) int {
	raw := fields[key]
	if len(raw) == 0 {
		return -1
	}
	var value int
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return -1
}
