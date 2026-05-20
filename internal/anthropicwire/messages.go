package anthropicwire

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

const (
	MessageTypeMessage = "message"
	RoleUser           = "user"
	RoleAssistant      = "assistant"

	ContentTypeText       = "text"
	ContentTypeImage      = "image"
	ContentTypeToolUse    = "tool_use"
	ContentTypeToolResult = "tool_result"

	StopReasonEndTurn  = "end_turn"
	StopReasonToolUse  = "tool_use"
	StopReasonMaxToken = "max_tokens"
)

type MessagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Messages    []MessageParam  `json:"messages"`
	System      SystemPrompt    `json:"system,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	RawBody     json.RawMessage `json:"-"`
}

type MessageParam struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

type Content struct {
	text   string
	blocks []ContentBlock
}

type SystemPrompt string

func (s SystemPrompt) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *SystemPrompt) UnmarshalJSON(data []byte) error {
	if s == nil {
		return nil
	}
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
		*s = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		*s = SystemPrompt(text)
		return nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(trimmed, &blocks); err == nil {
		parts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			if strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		}
		*s = SystemPrompt(strings.Join(parts, "\n"))
		return nil
	}
	return &json.UnmarshalTypeError{Value: "system", Type: reflect.TypeOf([]ContentBlock{}), Field: "system"}
}

func TextContent(text string) Content {
	return Content{text: text}
}

func BlocksContent(blocks ...ContentBlock) Content {
	cloned := append([]ContentBlock(nil), blocks...)
	return Content{blocks: cloned}
}

func (c Content) MarshalJSON() ([]byte, error) {
	if len(c.blocks) > 0 {
		return json.Marshal(c.blocks)
	}
	return json.Marshal(c.text)
}

func (c *Content) UnmarshalJSON(data []byte) error {
	if c == nil {
		return nil
	}
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
		*c = Content{}
		return nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		*c = TextContent(text)
		return nil
	}
	var blocks []ContentBlock
	if err := json.Unmarshal(trimmed, &blocks); err == nil {
		*c = BlocksContent(blocks...)
		return nil
	}
	return &json.UnmarshalTypeError{Value: "content", Type: reflect.TypeOf([]ContentBlock{}), Field: "content"}
}

func (c Content) Text() string {
	return c.text
}

func (c Content) Blocks() []ContentBlock {
	return append([]ContentBlock(nil), c.blocks...)
}

func (c Content) IsBlocks() bool {
	return len(c.blocks) > 0
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Source    *ImageSource    `json:"source,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   any             `json:"content,omitempty"`
}

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason,omitempty"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        *Usage         `json:"usage,omitempty"`
	Error        *Error         `json:"error,omitempty"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type StreamEvent struct {
	Type         string            `json:"type"`
	Message      *MessagesResponse `json:"message,omitempty"`
	Index        int               `json:"index,omitempty"`
	ContentBlock *ContentBlock     `json:"content_block,omitempty"`
	Delta        *StreamDelta      `json:"delta,omitempty"`
	Usage        *Usage            `json:"usage,omitempty"`
}

type StreamDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}
