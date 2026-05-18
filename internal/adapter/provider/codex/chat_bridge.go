package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

const (
	responsesEventOutputItemAdded            = "response.output_item.added"
	responsesEventFunctionArgsDelta          = "response.function_call_arguments.delta"
	responsesEventFunctionArgsDone           = "response.function_call_arguments.done"
	responsesInputTypeMessage                = "message"
	responsesInputTypeFunctionCall           = "function_call"
	responsesInputTypeFunctionCallOutput     = "function_call_output"
	responsesInputContentTypeInputText       = "input_text"
	responsesInputContentTypeInputImage      = "input_image"
	chatCompletionsStreamObject              = "chat.completion.chunk"
	chatCompletionsObject                    = "chat.completion"
	defaultAssistantToolCallIndex        int = 0
)

func translateChatCompletionsRequest(req openaiwire.ChatCompletionsRequest) (openaiwire.ResponsesRequest, error) {
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
			content, err := translateChatMessageContent(message.Content)
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
		tools, err := translateTools(req.Tools)
		if err != nil {
			return openaiwire.ResponsesRequest{}, err
		}
		payload["tools"] = tools
	}
	if req.ToolChoice != nil {
		toolChoice, err := translateToolChoice(req.ToolChoice)
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

func translateChatMessageContent(content openaiwire.ChatMessageContent) ([]openaiwire.ResponseInputContentPart, error) {
	if !content.IsParts() {
		if content.Text() == "" {
			return nil, nil
		}
		return []openaiwire.ResponseInputContentPart{{
			Type: responsesInputContentTypeInputText,
			Text: content.Text(),
		}}, nil
	}

	parts := content.Parts()
	out := make([]openaiwire.ResponseInputContentPart, 0, len(parts))
	for index, part := range parts {
		switch part.Type {
		case "text":
			out = append(out, openaiwire.ResponseInputContentPart{
				Type: responsesInputContentTypeInputText,
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

func translateTools(tools []openaiwire.Tool) ([]map[string]any, error) {
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

func translateToolChoice(choice any) (any, error) {
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

func translateResponsesResponseToChatCompletion(response openaiwire.ResponsesResponse) openaiwire.ChatCompletionsResponse {
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
		ID:      response.ID,
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
}

type toolCallState struct {
	ID           string
	Name         string
	Arguments    string
	Index        int
	MetadataSent bool
}

func translateResponsesStreamToChatCompletions(body io.ReadCloser) io.ReadCloser {
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

		flushEvent := func() error {
			if len(dataLines) == 0 {
				return nil
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]
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
		}
	}()

	return reader
}

func (s *chatStreamState) writeTranslatedEvent(w io.Writer, payload string) error {
	if strings.TrimSpace(payload) == "" {
		return nil
	}
	if payload == "[DONE]" {
		if !s.terminalSent {
			if err := s.writeTerminalChunk(w, nil); err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(w, "data: [DONE]\n\n")
		return err
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
	case responsesEventOutputItemAdded, string(openaiwire.ResponsesStreamEventTypeOutputItemDone):
		return s.writeOutputItemEvent(w, envelope)
	case string(openaiwire.ResponsesStreamEventTypeCompleted):
		return s.writeTerminalChunk(w, envelope["response"])
	default:
		return nil
	}
}

func (s *chatStreamState) writeOutputItemEvent(w io.Writer, envelope map[string]json.RawMessage) error {
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
		s.id = response.ID
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

func defaultCallID(callID string, fallback string) string {
	if strings.TrimSpace(callID) != "" {
		return callID
	}
	return strings.TrimSpace(fallback)
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
