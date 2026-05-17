package openaiwire

import (
	"bytes"
	"encoding/json"
)

type ChatRole string

const (
	ChatRoleSystem    ChatRole = "system"
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleTool      ChatRole = "tool"
)

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolCalls FinishReason = "tool_calls"
)

type ChatCompletionsRequest struct {
	Model           string          `json:"model"`
	Messages        []ChatMessage   `json:"messages"`
	Stream          bool            `json:"stream,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
	MaxTokens       *int            `json:"max_tokens,omitempty"`
	Tools           []Tool          `json:"tools,omitempty"`
	ToolChoice      any             `json:"tool_choice,omitempty"`
	Reasoning       json.RawMessage `json:"reasoning,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
}

type ChatMessageContent struct {
	text  string
	parts []ChatMessageContentPart
}

func TextContent(text string) ChatMessageContent {
	return ChatMessageContent{text: text}
}

func PartsContent(parts ...ChatMessageContentPart) ChatMessageContent {
	cloned := append([]ChatMessageContentPart(nil), parts...)
	return ChatMessageContent{parts: cloned}
}

func (c ChatMessageContent) MarshalJSON() ([]byte, error) {
	if len(c.parts) > 0 {
		return json.Marshal(c.parts)
	}
	return json.Marshal(c.text)
}

func (c *ChatMessageContent) UnmarshalJSON(data []byte) error {
	if c == nil {
		return nil
	}

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
		*c = ChatMessageContent{}
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		*c = TextContent(text)
		return nil
	}

	var parts []ChatMessageContentPart
	if err := json.Unmarshal(trimmed, &parts); err == nil {
		*c = PartsContent(parts...)
		return nil
	}

	return &json.UnmarshalTypeError{Value: "content", Type: nil, Field: "content"}
}

func (c ChatMessageContent) Text() string {
	return c.text
}

func (c ChatMessageContent) Parts() []ChatMessageContentPart {
	return append([]ChatMessageContentPart(nil), c.parts...)
}

func (c ChatMessageContent) IsParts() bool {
	return len(c.parts) > 0
}

type ChatMessageContentPart struct {
	Type     string               `json:"type"`
	Text     string               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL `json:"image_url,omitempty"`
}

type ChatMessageImageURL struct {
	URL    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type ChatMessage struct {
	Role       ChatRole           `json:"role"`
	Content    ChatMessageContent `json:"content"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall         `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     ToolType         `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     ToolType     `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

type ChatCompletionsResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             *CompletionUsage       `json:"usage,omitempty"`
	Error             *OpenAIError           `json:"error,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
	LogProbs     any          `json:"logprobs,omitempty"`
}

type Message struct {
	Role      ChatRole   `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Refusal   string     `json:"refusal,omitempty"`
	Audio     any        `json:"audio,omitempty"`
}

type CompletionUsage struct {
	PromptTokens        int                  `json:"prompt_tokens"`
	CompletionTokens    int                  `json:"completion_tokens"`
	TotalTokens         int                  `json:"total_tokens"`
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type ChatChoice = ChatCompletionChoice

type Usage = CompletionUsage
