package protocoltranslator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/anthropicwire"
	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

func AnthropicToResponses(req anthropicwire.MessagesRequest) (openaiwire.ResponsesRequest, error) {
	input := make([]openaiwire.ResponseInputItem, 0, len(req.Messages))
	for index, message := range req.Messages {
		items, err := anthropicMessageToResponsesItems(message)
		if err != nil {
			return openaiwire.ResponsesRequest{}, fmt.Errorf("messages[%d]: %w", index, err)
		}
		input = append(input, items...)
	}

	payload := map[string]any{
		"model":  req.Model,
		"input":  input,
		"stream": req.Stream,
	}
	if strings.TrimSpace(string(req.System)) != "" {
		payload["instructions"] = string(req.System)
	}
	if req.MaxTokens > 0 {
		payload["max_output_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		payload["temperature"] = *req.Temperature
	}
	if len(req.Tools) > 0 {
		payload["tools"] = anthropicToolsToResponses(req.Tools)
	}
	if req.ToolChoice != nil {
		payload["tool_choice"] = req.ToolChoice
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return openaiwire.ResponsesRequest{}, fmt.Errorf("marshal translated responses payload: %w", err)
	}

	return openaiwire.ResponsesRequest{
		Model:        req.Model,
		Instructions: string(req.System),
		Input:        input,
		Stream:       req.Stream,
		RawBody:      rawBody,
	}, nil
}

func AnthropicToChatCompletions(req anthropicwire.MessagesRequest) (openaiwire.ChatCompletionsRequest, error) {
	out := openaiwire.ChatCompletionsRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
	}
	if req.MaxTokens > 0 {
		out.MaxTokens = &req.MaxTokens
	}
	if strings.TrimSpace(string(req.System)) != "" {
		out.Messages = append(out.Messages, openaiwire.ChatMessage{
			Role:    openaiwire.ChatRoleSystem,
			Content: openaiwire.TextContent(string(req.System)),
		})
	}
	for index, message := range req.Messages {
		messages, err := anthropicMessageToChatMessages(message)
		if err != nil {
			return openaiwire.ChatCompletionsRequest{}, fmt.Errorf("messages[%d]: %w", index, err)
		}
		out.Messages = append(out.Messages, messages...)
	}
	if len(req.Tools) > 0 {
		out.Tools = anthropicToolsToChat(req.Tools)
	}
	out.ToolChoice = anthropicToolChoiceToChat(req.ToolChoice)
	return out, nil
}

func ResponsesToAnthropic(response openaiwire.ResponsesResponse) anthropicwire.MessagesResponse {
	content := make([]anthropicwire.ContentBlock, 0, len(response.Output))
	stopReason := anthropicwire.StopReasonEndTurn
	for _, item := range response.Output {
		switch item.Type {
		case openaiwire.OutputItemTypeMessage:
			for _, part := range item.Content {
				if part.Type == openaiwire.OutputContentTypeOutputText && part.Text != "" {
					content = append(content, anthropicwire.ContentBlock{
						Type: anthropicwire.ContentTypeText,
						Text: part.Text,
					})
				}
			}
		case openaiwire.OutputItemTypeFunctionCall:
			stopReason = anthropicwire.StopReasonToolUse
			content = append(content, anthropicwire.ContentBlock{
				Type:  anthropicwire.ContentTypeToolUse,
				ID:    defaultCallID(item.CallID, item.ID),
				Name:  item.Name,
				Input: json.RawMessage(defaultJSON(item.Arguments)),
			})
		}
	}
	if len(content) == 0 {
		content = []anthropicwire.ContentBlock{{Type: anthropicwire.ContentTypeText, Text: response.TextValue()}}
	}

	out := anthropicwire.MessagesResponse{
		ID:         anthropicMessageID(response.ID),
		Type:       anthropicwire.MessageTypeMessage,
		Role:       anthropicwire.RoleAssistant,
		Content:    content,
		Model:      response.Model,
		StopReason: stopReason,
	}
	if response.Usage != nil {
		out.Usage = &anthropicwire.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		}
		if response.Usage.InputTokensDetails != nil {
			out.Usage.CacheReadInputTokens = response.Usage.InputTokensDetails.CachedTokens
		}
	}
	if response.Error != nil {
		out.Error = &anthropicwire.Error{Type: response.Error.Type, Message: response.Error.Message}
	}
	return out
}

func ChatCompletionsToAnthropic(response openaiwire.ChatCompletionsResponse) anthropicwire.MessagesResponse {
	return ResponsesToAnthropic(ChatCompletionToResponses(response))
}

func ResponsesToAnthropicRequest(req openaiwire.ResponsesRequest) (anthropicwire.MessagesRequest, error) {
	if len(req.RawBody) > 0 {
		var decoded openaiwire.ResponsesRequest
		if err := json.Unmarshal(req.RawBody, &decoded); err != nil {
			return anthropicwire.MessagesRequest{}, err
		}
		decoded.RawBody = nil
		decoded.Model = req.Model
		if req.Stream {
			decoded.Stream = true
		}
		req = decoded
	}

	out := anthropicwire.MessagesRequest{
		Model:  req.Model,
		System: anthropicwire.SystemPrompt(req.Instructions),
		Stream: req.Stream,
	}
	if strings.TrimSpace(req.InputText) != "" {
		out.Messages = append(out.Messages, anthropicwire.MessageParam{
			Role:    anthropicwire.RoleUser,
			Content: anthropicwire.TextContent(req.InputText),
		})
	}
	for index, item := range req.Input {
		messages, err := responseInputItemToAnthropicMessages(item)
		if err != nil {
			return anthropicwire.MessagesRequest{}, fmt.Errorf("input[%d]: %w", index, err)
		}
		out.Messages = append(out.Messages, messages...)
	}
	return out, nil
}

func AnthropicToResponsesResponse(response anthropicwire.MessagesResponse) openaiwire.ResponsesResponse {
	output := make([]openaiwire.OutputItem, 0, len(response.Content))
	for _, block := range response.Content {
		switch block.Type {
		case anthropicwire.ContentTypeText:
			output = append(output, openaiwire.OutputItem{
				Type: openaiwire.OutputItemTypeMessage,
				Role: string(openaiwire.ChatRoleAssistant),
				Content: []openaiwire.OutputContent{{
					Type: openaiwire.OutputContentTypeOutputText,
					Text: block.Text,
				}},
			})
		case anthropicwire.ContentTypeToolUse:
			output = append(output, openaiwire.OutputItem{
				Type:      openaiwire.OutputItemTypeFunctionCall,
				ID:        block.ID,
				CallID:    block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}
	status := openaiwire.ResponsesStatusCompleted
	var responseErr *openaiwire.ResponseError
	if response.Error != nil {
		status = openaiwire.ResponsesStatusFailed
		responseErr = &openaiwire.ResponseError{Type: response.Error.Type, Message: response.Error.Message}
	}
	out := openaiwire.ResponsesResponse{
		ID:        response.ID,
		Object:    "response",
		CreatedAt: time.Now().Unix(),
		Status:    status,
		Model:     response.Model,
		Output:    output,
		Error:     responseErr,
	}
	if response.Usage != nil {
		out.Usage = &openaiwire.ResponseUsage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
		}
		if response.Usage.CacheReadInputTokens > 0 {
			out.Usage.InputTokensDetails = &openaiwire.ResponseInputTokenDetails{CachedTokens: response.Usage.CacheReadInputTokens}
		}
	}
	return out
}

func responseInputItemToAnthropicMessages(item openaiwire.ResponseInputItem) ([]anthropicwire.MessageParam, error) {
	switch item.Type {
	case "", responsesInputTypeMessage:
		role := item.Role
		if role == "" {
			role = anthropicwire.RoleUser
		}
		blocks := make([]anthropicwire.ContentBlock, 0, len(item.Content))
		for index, part := range item.Content {
			switch part.Type {
			case responsesInputContentTypeInputText, responsesInputContentTypeOutputText:
				blocks = append(blocks, anthropicwire.ContentBlock{Type: anthropicwire.ContentTypeText, Text: part.Text})
			case responsesInputContentTypeInputImage:
				blocks = append(blocks, anthropicwire.ContentBlock{Type: anthropicwire.ContentTypeImage, Source: imageURLToAnthropicSource(part.ImageURL)})
			default:
				return nil, fmt.Errorf("content[%d].type %q is not supported by anthropic translation", index, part.Type)
			}
		}
		return []anthropicwire.MessageParam{{Role: role, Content: anthropicwire.BlocksContent(blocks...)}}, nil
	case responsesInputTypeFunctionCall:
		return []anthropicwire.MessageParam{{Role: anthropicwire.RoleAssistant, Content: anthropicwire.BlocksContent(anthropicwire.ContentBlock{Type: anthropicwire.ContentTypeToolUse, ID: item.CallID, Name: item.Name, Input: json.RawMessage(defaultJSON(item.Arguments))})}}, nil
	case responsesInputTypeFunctionCallOutput:
		return []anthropicwire.MessageParam{{Role: anthropicwire.RoleUser, Content: anthropicwire.BlocksContent(anthropicwire.ContentBlock{Type: anthropicwire.ContentTypeToolResult, ToolUseID: item.CallID, Content: item.Output})}}, nil
	case "reasoning":
		return nil, nil
	default:
		return nil, fmt.Errorf("type %q is not supported by anthropic translation", item.Type)
	}
}

func imageURLToAnthropicSource(url string) *anthropicwire.ImageSource {
	if strings.HasPrefix(url, "data:") {
		mediaType := "image/jpeg"
		data := url
		if parts := strings.SplitN(strings.TrimPrefix(url, "data:"), ";base64,", 2); len(parts) == 2 {
			mediaType = parts[0]
			data = parts[1]
		}
		return &anthropicwire.ImageSource{Type: "base64", MediaType: mediaType, Data: data}
	}
	return &anthropicwire.ImageSource{Type: "url", URL: url}
}

func AnthropicToChatCompletionsResponse(response anthropicwire.MessagesResponse) openaiwire.ChatCompletionsResponse {
	return ResponsesToChatCompletion(AnthropicToResponsesResponse(response))
}

func ResponsesStreamToAnthropic(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer body.Close()
		err := translateResponsesStreamToAnthropic(body, writer)
		_ = writer.CloseWithError(err)
	}()
	return reader
}

func ChatCompletionsStreamToAnthropic(body io.ReadCloser) io.ReadCloser {
	return ResponsesStreamToAnthropic(ChatCompletionsStreamToResponses(body))
}

func AnthropicStreamToResponses(body io.ReadCloser) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer body.Close()
		err := translateAnthropicStreamToResponses(body, writer)
		_ = writer.CloseWithError(err)
	}()
	return reader
}

func AnthropicStreamToChatCompletions(body io.ReadCloser) io.ReadCloser {
	return ResponsesStreamToChatCompletions(AnthropicStreamToResponses(body))
}

func RewriteAnthropicStreamModel(body io.ReadCloser, model string) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer body.Close()
		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload != "" && payload != "[DONE]" {
					var object map[string]json.RawMessage
					if err := json.Unmarshal([]byte(payload), &object); err == nil {
						if rawMessage, ok := object["message"]; ok {
							var message map[string]json.RawMessage
							if err := json.Unmarshal(rawMessage, &message); err == nil {
								encodedModel, _ := json.Marshal(model)
								message["model"] = encodedModel
								if encodedMessage, err := json.Marshal(message); err == nil {
									object["message"] = encodedMessage
								}
							}
						}
						if _, ok := object["model"]; ok {
							encodedModel, _ := json.Marshal(model)
							object["model"] = encodedModel
						}
						if encoded, err := json.Marshal(object); err == nil {
							line = "data: " + string(encoded)
						}
					}
				}
			}
			if _, err := io.WriteString(writer, line+"\n"); err != nil {
				_ = writer.CloseWithError(err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()
	return reader
}

func anthropicMessageToResponsesItems(message anthropicwire.MessageParam) ([]openaiwire.ResponseInputItem, error) {
	role := message.Role
	if role == "" {
		role = anthropicwire.RoleUser
	}
	switch role {
	case anthropicwire.RoleUser, anthropicwire.RoleAssistant:
	default:
		return nil, fmt.Errorf("role %q is not supported", role)
	}

	parts := message.Content.Blocks()
	if !message.Content.IsBlocks() {
		textType := responsesInputContentTypeInputText
		if role == anthropicwire.RoleAssistant {
			textType = responsesInputContentTypeOutputText
		}
		return []openaiwire.ResponseInputItem{{
			Type:    responsesInputTypeMessage,
			Role:    role,
			Content: []openaiwire.ResponseInputContentPart{{Type: textType, Text: message.Content.Text()}},
		}}, nil
	}

	items := make([]openaiwire.ResponseInputItem, 0, len(parts))
	messageParts := make([]openaiwire.ResponseInputContentPart, 0, len(parts))
	for index, part := range parts {
		switch part.Type {
		case anthropicwire.ContentTypeText:
			textType := responsesInputContentTypeInputText
			if role == anthropicwire.RoleAssistant {
				textType = responsesInputContentTypeOutputText
			}
			messageParts = append(messageParts, openaiwire.ResponseInputContentPart{Type: textType, Text: part.Text})
		case anthropicwire.ContentTypeImage:
			if part.Source == nil {
				return nil, fmt.Errorf("content[%d].source is required", index)
			}
			messageParts = append(messageParts, openaiwire.ResponseInputContentPart{Type: responsesInputContentTypeInputImage, ImageURL: anthropicImageURL(part.Source)})
		case anthropicwire.ContentTypeToolUse:
			if len(messageParts) > 0 {
				items = append(items, openaiwire.ResponseInputItem{Type: responsesInputTypeMessage, Role: role, Content: messageParts})
				messageParts = nil
			}
			items = append(items, openaiwire.ResponseInputItem{Type: responsesInputTypeFunctionCall, CallID: part.ID, Name: part.Name, Arguments: string(part.Input)})
		case anthropicwire.ContentTypeToolResult:
			if len(messageParts) > 0 {
				items = append(items, openaiwire.ResponseInputItem{Type: responsesInputTypeMessage, Role: role, Content: messageParts})
				messageParts = nil
			}
			items = append(items, openaiwire.ResponseInputItem{Type: responsesInputTypeFunctionCallOutput, CallID: part.ToolUseID, Output: anthropicToolResultText(part.Content)})
		default:
			return nil, fmt.Errorf("content[%d].type %q is not supported", index, part.Type)
		}
	}
	if len(messageParts) > 0 {
		items = append(items, openaiwire.ResponseInputItem{Type: responsesInputTypeMessage, Role: role, Content: messageParts})
	}
	return items, nil
}

func anthropicMessageToChatMessages(message anthropicwire.MessageParam) ([]openaiwire.ChatMessage, error) {
	role := openaiwire.ChatRole(message.Role)
	if role == "" {
		role = openaiwire.ChatRoleUser
	}
	if !message.Content.IsBlocks() {
		return []openaiwire.ChatMessage{{Role: role, Content: openaiwire.TextContent(message.Content.Text())}}, nil
	}

	out := make([]openaiwire.ChatMessage, 0, 2)
	chatParts := make([]openaiwire.ChatMessageContentPart, 0, len(message.Content.Blocks()))
	toolCalls := make([]openaiwire.ToolCall, 0, 1)
	for index, part := range message.Content.Blocks() {
		switch part.Type {
		case anthropicwire.ContentTypeText:
			chatParts = append(chatParts, openaiwire.ChatMessageContentPart{Type: "text", Text: part.Text})
		case anthropicwire.ContentTypeImage:
			if part.Source == nil {
				return nil, fmt.Errorf("content[%d].source is required", index)
			}
			chatParts = append(chatParts, openaiwire.ChatMessageContentPart{Type: "image_url", ImageURL: &openaiwire.ChatMessageImageURL{URL: anthropicImageURL(part.Source)}})
		case anthropicwire.ContentTypeToolUse:
			toolCalls = append(toolCalls, openaiwire.ToolCall{ID: part.ID, Type: openaiwire.ToolTypeFunction, Function: openaiwire.ToolCallFunction{Name: part.Name, Arguments: string(part.Input)}})
		case anthropicwire.ContentTypeToolResult:
			if len(chatParts) > 0 || len(toolCalls) > 0 {
				out = append(out, openaiwire.ChatMessage{Role: role, Content: chatContent(chatParts), ToolCalls: toolCalls})
				chatParts = nil
				toolCalls = nil
			}
			out = append(out, openaiwire.ChatMessage{Role: openaiwire.ChatRoleTool, ToolCallID: part.ToolUseID, Content: openaiwire.TextContent(anthropicToolResultText(part.Content))})
		default:
			return nil, fmt.Errorf("content[%d].type %q is not supported", index, part.Type)
		}
	}
	if len(chatParts) > 0 || len(toolCalls) > 0 {
		out = append(out, openaiwire.ChatMessage{Role: role, Content: chatContent(chatParts), ToolCalls: toolCalls})
	}
	return out, nil
}

func chatContent(parts []openaiwire.ChatMessageContentPart) openaiwire.ChatMessageContent {
	if len(parts) == 0 {
		return openaiwire.TextContent("")
	}
	if len(parts) == 1 && parts[0].Type == "text" {
		return openaiwire.TextContent(parts[0].Text)
	}
	return openaiwire.PartsContent(parts...)
}

func anthropicToolsToResponses(tools []anthropicwire.Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		item := map[string]any{"type": "function", "name": tool.Name}
		if tool.Description != "" {
			item["description"] = tool.Description
		}
		if len(tool.InputSchema) > 0 {
			var schema any
			if err := json.Unmarshal(tool.InputSchema, &schema); err == nil {
				item["parameters"] = schema
			}
		}
		out = append(out, item)
	}
	return out
}

func anthropicToolsToChat(tools []anthropicwire.Tool) []openaiwire.Tool {
	out := make([]openaiwire.Tool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, openaiwire.Tool{Type: openaiwire.ToolTypeFunction, Function: openaiwire.ToolFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema}})
	}
	return out
}

func anthropicToolChoiceToChat(choice any) any {
	object, ok := choice.(map[string]any)
	if !ok {
		return choice
	}
	if object["type"] == "tool" {
		if name, ok := object["name"].(string); ok {
			return map[string]any{"type": "function", "function": map[string]any{"name": name}}
		}
	}
	return choice
}

func anthropicImageURL(source *anthropicwire.ImageSource) string {
	if source == nil {
		return ""
	}
	if source.URL != "" {
		return source.URL
	}
	if source.Data != "" {
		mediaType := source.MediaType
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		return "data:" + mediaType + ";base64," + source.Data
	}
	return ""
}

func anthropicToolResultText(content any) string {
	switch typed := content.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []any:
		var builder strings.Builder
		for _, item := range typed {
			if block, ok := item.(map[string]any); ok && block["type"] == anthropicwire.ContentTypeText {
				if text, ok := block["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		if builder.Len() > 0 {
			return builder.String()
		}
	}
	encoded, _ := json.Marshal(content)
	return string(encoded)
}

func defaultJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return "{}"
	}
	return value
}

func anthropicMessageID(id string) string {
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "msg_") {
		return id
	}
	return "msg_" + strings.TrimPrefix(strings.TrimPrefix(id, "resp_"), "resp-")
}

func translateResponsesStreamToAnthropic(r io.Reader, w io.Writer) error {
	state := responsesParseState{CreatedAt: time.Now().Unix(), Status: openaiwire.ResponsesStatusInProgress, Usage: &openaiwire.ResponseUsage{}, ItemsByIdx: make(map[int]openaiwire.OutputItem)}
	streamState := anthropicStreamState{toolBlocksByOutputIndex: make(map[int]anthropicToolBlock)}
	started := false
	startAnthropicMessage := func() error {
		if started {
			return nil
		}
		started = true
		if err := writeAnthropicEvent(w, "message_start", anthropicMessageStartPayload(state)); err != nil {
			return err
		}
		if err := writeAnthropicEvent(w, "ping", map[string]any{"type": "ping"}); err != nil {
			return err
		}
		return nil
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event openaiwire.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return err
		}
		processResponsesEvent(event, &state)
		if event.Type == openaiwire.ResponsesStreamEventTypeCreated {
			if err := startAnthropicMessage(); err != nil {
				return err
			}
		}
		if event.Type == openaiwire.ResponsesStreamEventTypeOutputTextDelta {
			delta := event.TextValue()
			if delta == "" {
				continue
			}
			if err := startAnthropicMessage(); err != nil {
				return err
			}
			if err := streamState.startTextBlock(w); err != nil {
				return err
			}
			if err := writeAnthropicEvent(w, "content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": streamState.activeBlockIndex,
				"delta": map[string]any{"type": "text_delta", "text": delta},
			}); err != nil {
				return err
			}
		}
		if event.Type == responsesEventOutputItemAdded && event.Item != nil && event.Item.Type == openaiwire.OutputItemTypeFunctionCall {
			if err := startAnthropicMessage(); err != nil {
				return err
			}
			if err := streamState.startToolBlock(w, event.OutputIndex, *event.Item); err != nil {
				return err
			}
		}
		if event.Type == responsesEventFunctionArgsDelta {
			if err := startAnthropicMessage(); err != nil {
				return err
			}
			if err := streamState.writeToolDelta(w, event); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := startAnthropicMessage(); err != nil {
		return err
	}
	if !streamState.startedAnyBlock {
		if err := streamState.startTextBlock(w); err != nil {
			return err
		}
	}
	response := ResponsesToAnthropic(finalizeResponses(state))
	if err := streamState.stopActiveBlock(w); err != nil {
		return err
	}
	if err := writeAnthropicEvent(w, "message_delta", anthropicMessageDeltaPayload(response)); err != nil {
		return err
	}
	return writeAnthropicEvent(w, "message_stop", anthropicwire.StreamEvent{Type: "message_stop"})
}

type anthropicStreamState struct {
	startedAnyBlock         bool
	activeBlockIndex        int
	activeBlockType         string
	nextBlockIndex          int
	toolBlocksByOutputIndex map[int]anthropicToolBlock
}

type anthropicToolBlock struct {
	ContentIndex int
	ID           string
	Name         string
}

func (s *anthropicStreamState) startTextBlock(w io.Writer) error {
	if s.activeBlockType == anthropicwire.ContentTypeText {
		return nil
	}
	if err := s.stopActiveBlock(w); err != nil {
		return err
	}
	index := s.nextContentIndex()
	s.activeBlockIndex = index
	s.activeBlockType = anthropicwire.ContentTypeText
	return writeAnthropicEvent(w, "content_block_start", map[string]any{
		"type":          "content_block_start",
		"index":         index,
		"content_block": map[string]any{"type": anthropicwire.ContentTypeText, "text": ""},
	})
}

func (s *anthropicStreamState) startToolBlock(w io.Writer, outputIndex int, item openaiwire.OutputItem) error {
	if block, ok := s.toolBlocksByOutputIndex[outputIndex]; ok {
		if s.activeBlockType == anthropicwire.ContentTypeToolUse && s.activeBlockIndex == block.ContentIndex {
			return nil
		}
		return nil
	}
	if err := s.stopActiveBlock(w); err != nil {
		return err
	}
	id := defaultCallID(item.CallID, item.ID)
	if id == "" {
		id = "toolu_" + strconv.Itoa(outputIndex)
	}
	block := anthropicToolBlock{ContentIndex: s.nextContentIndex(), ID: id, Name: item.Name}
	s.toolBlocksByOutputIndex[outputIndex] = block
	s.activeBlockIndex = block.ContentIndex
	s.activeBlockType = anthropicwire.ContentTypeToolUse
	return writeAnthropicEvent(w, "content_block_start", map[string]any{
		"type":  "content_block_start",
		"index": block.ContentIndex,
		"content_block": map[string]any{
			"type":  anthropicwire.ContentTypeToolUse,
			"id":    block.ID,
			"name":  block.Name,
			"input": map[string]any{},
		},
	})
}

func (s *anthropicStreamState) writeToolDelta(w io.Writer, event openaiwire.ResponsesStreamEvent) error {
	delta := event.Delta
	if delta == "" {
		return nil
	}
	block, ok := s.toolBlocksByOutputIndex[event.OutputIndex]
	if !ok {
		item := openaiwire.OutputItem{Type: openaiwire.OutputItemTypeFunctionCall, ID: event.ItemID, CallID: event.CallID}
		if event.Item != nil {
			item = *event.Item
		}
		if err := s.startToolBlock(w, event.OutputIndex, item); err != nil {
			return err
		}
		block = s.toolBlocksByOutputIndex[event.OutputIndex]
	}
	return writeAnthropicEvent(w, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": block.ContentIndex,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": delta},
	})
}

func (s *anthropicStreamState) stopActiveBlock(w io.Writer) error {
	if s.activeBlockType == "" {
		return nil
	}
	index := s.activeBlockIndex
	s.activeBlockType = ""
	return writeAnthropicEvent(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
}

func (s *anthropicStreamState) nextContentIndex() int {
	index := s.nextBlockIndex
	s.nextBlockIndex++
	s.startedAnyBlock = true
	return index
}

func anthropicMessageStartPayload(state responsesParseState) map[string]any {
	id := anthropicMessageID(state.ID)
	if id == "" {
		id = "msg_stream"
	}
	return map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            id,
			"type":          anthropicwire.MessageTypeMessage,
			"role":          anthropicwire.RoleAssistant,
			"content":       []any{},
			"model":         state.Model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         anthropicUsageFromResponses(state.Usage),
		},
	}
}

func anthropicMessageDeltaPayload(response anthropicwire.MessagesResponse) map[string]any {
	return map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   response.StopReason,
			"stop_sequence": nil,
		},
		"usage": response.Usage,
	}
}

func anthropicUsageFromResponses(usage *openaiwire.ResponseUsage) map[string]int {
	out := map[string]int{"input_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0, "output_tokens": 0}
	if usage == nil {
		return out
	}
	out["input_tokens"] = usage.InputTokens
	out["output_tokens"] = usage.OutputTokens
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens > 0 {
		out["cache_read_input_tokens"] = usage.InputTokensDetails.CachedTokens
	}
	return out
}

func translateAnthropicStreamToResponses(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	var id, model string
	var content strings.Builder
	var usage *anthropicwire.Usage
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event anthropicwire.StreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return err
		}
		switch event.Type {
		case "message_start":
			if event.Message != nil {
				id = event.Message.ID
				model = event.Message.Model
				usage = event.Message.Usage
			}
			if err := writeResponsesEvent(w, string(openaiwire.ResponsesStreamEventTypeCreated), openaiwire.ResponsesStreamEvent{Type: openaiwire.ResponsesStreamEventTypeCreated, Response: &openaiwire.ResponsesResponse{ID: id, Object: "response", CreatedAt: time.Now().Unix(), Status: openaiwire.ResponsesStatusInProgress, Model: model, Output: []openaiwire.OutputItem{}}}); err != nil {
				return err
			}
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				content.WriteString(event.Delta.Text)
				if err := writeResponsesEvent(w, string(openaiwire.ResponsesStreamEventTypeOutputTextDelta), openaiwire.ResponsesStreamEvent{Type: openaiwire.ResponsesStreamEventTypeOutputTextDelta, Delta: event.Delta.Text}); err != nil {
					return err
				}
			}
		case "message_delta":
			if event.Usage != nil {
				usage = event.Usage
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	response := AnthropicToResponsesResponse(anthropicwire.MessagesResponse{
		ID: id, Type: "message", Role: "assistant", Model: model, StopReason: anthropicwire.StopReasonEndTurn,
		Content: []anthropicwire.ContentBlock{{Type: anthropicwire.ContentTypeText, Text: content.String()}},
		Usage:   usage,
	})
	if err := writeResponsesEvent(w, string(openaiwire.ResponsesStreamEventTypeOutputItemDone), openaiwire.ResponsesStreamEvent{Type: openaiwire.ResponsesStreamEventTypeOutputItemDone, OutputIndex: 0, Item: &response.Output[0]}); err != nil {
		return err
	}
	if err := writeResponsesEvent(w, string(openaiwire.ResponsesStreamEventTypeCompleted), openaiwire.ResponsesStreamEvent{Type: openaiwire.ResponsesStreamEventTypeCompleted, Response: &response}); err != nil {
		return err
	}
	_, err := io.WriteString(w, "data: [DONE]\n\n")
	return err
}

func writeAnthropicEvent(w io.Writer, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n\n", event, encoded)
	return err
}

func writeResponsesEvent(w io.Writer, event string, payload openaiwire.ResponsesStreamEvent) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, encoded)
	return err
}

func ParseAnthropicSSE(data []byte) (anthropicwire.MessagesResponse, error) {
	var id, model string
	var content strings.Builder
	var usage *anthropicwire.Usage
	for _, raw := range anthropicSSEDataEvents(data) {
		var event anthropicwire.StreamEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return anthropicwire.MessagesResponse{}, err
		}
		if event.Message != nil {
			id = event.Message.ID
			model = event.Message.Model
			if event.Message.Usage != nil {
				usage = event.Message.Usage
			}
		}
		if event.Delta != nil && event.Delta.Text != "" {
			content.WriteString(event.Delta.Text)
		}
		if event.Usage != nil {
			usage = event.Usage
		}
	}
	return anthropicwire.MessagesResponse{
		ID: id, Type: "message", Role: "assistant", Model: model, StopReason: anthropicwire.StopReasonEndTurn,
		Content: []anthropicwire.ContentBlock{{Type: anthropicwire.ContentTypeText, Text: content.String()}},
		Usage:   usage,
	}, nil
}

func anthropicSSEDataEvents(data []byte) []string {
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
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload != "" && payload != "[DONE]" {
				builder.WriteString(payload)
				builder.WriteByte('\n')
			}
		}
	}
	if builder.Len() > 0 {
		events = append(events, strings.TrimSuffix(builder.String(), "\n"))
	}
	return events
}
