package openaiwire

import "strings"

type ChatCompletionsStreamChunk struct {
	ID      string                        `json:"id"`
	Object  string                        `json:"object"`
	Created int64                         `json:"created"`
	Model   string                        `json:"model"`
	Choices []ChatCompletionsStreamChoice `json:"choices"`
	Usage   *CompletionUsage              `json:"usage,omitempty"`
}

type ChatCompletionsStreamChoice struct {
	Index        int     `json:"index"`
	Delta        Message `json:"delta"`
	FinishReason string  `json:"finish_reason"`
}

type ResponsesStreamEvent struct {
	Type     string          `json:"type,omitempty"`
	ID       string          `json:"id,omitempty"`
	Text     string          `json:"text,omitempty"`
	Delta    string          `json:"delta,omitempty"`
	Content  []OutputContent `json:"content,omitempty"`
	Output   []OutputItem    `json:"output,omitempty"`
	Item     *OutputItem     `json:"item,omitempty"`
	Response *Response       `json:"response,omitempty"`
}

func (e ResponsesStreamEvent) TextValue() string {
	var builder strings.Builder
	builder.WriteString(e.Text)
	builder.WriteString(e.Delta)
	for _, part := range e.Content {
		builder.WriteString(part.TextValue())
	}
	for _, item := range e.Output {
		builder.WriteString(item.TextValue())
	}
	if e.Item != nil {
		builder.WriteString(e.Item.TextValue())
	}
	if e.Response != nil {
		builder.WriteString(e.Response.TextValue())
	}
	return builder.String()
}

func (r Response) TextValue() string {
	var builder strings.Builder
	for _, item := range r.Output {
		builder.WriteString(item.TextValue())
	}
	return builder.String()
}

func (i OutputItem) TextValue() string {
	var builder strings.Builder
	for _, part := range i.Content {
		builder.WriteString(part.TextValue())
	}
	return builder.String()
}

func (p OutputContent) TextValue() string {
	return p.Text
}
