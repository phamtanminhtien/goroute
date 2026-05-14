package openaiwire

import "encoding/json"

type ChatCompletionsRequest struct {
	Model           string          `json:"model"`
	Messages        []ChatMessage   `json:"messages"`
	Stream          bool            `json:"stream,omitempty"`
	Reasoning       json.RawMessage `json:"reasoning,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatCompletionsResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
}

type ChatChoice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}
