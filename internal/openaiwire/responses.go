package openaiwire

import "encoding/json"

type ResponsesRequest struct {
	Model        string              `json:"model"`
	Instructions string              `json:"instructions"`
	Input        []ResponseInputItem `json:"input"`
	Stream       bool                `json:"stream"`
	Store        bool                `json:"store"`
	Reasoning    any                 `json:"reasoning,omitempty"`
	Text         *ResponseText       `json:"text,omitempty"`
	Include      []string            `json:"include,omitempty"`
}

type ResponseInputItem struct {
	Type      string                     `json:"type"`
	Role      string                     `json:"role,omitempty"`
	Content   []ResponseInputContentPart `json:"content,omitempty"`
	CallID    string                     `json:"call_id,omitempty"`
	Name      string                     `json:"name,omitempty"`
	Arguments string                     `json:"arguments,omitempty"`
	Output    string                     `json:"output,omitempty"`
}

type ResponseInputContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type ResponseText struct {
	Format *ResponseFormat `json:"format,omitempty"`
}

type ResponseFormat struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

type ResponseReasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

type Response struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	CreatedAt         int64          `json:"created_at"`
	Status            string         `json:"status"`
	Model             string         `json:"model"`
	Output            []OutputItem   `json:"output"`
	Usage             *ResponseUsage `json:"usage,omitempty"`
	Error             *ResponseError `json:"error,omitempty"`
	IncompleteDetails any            `json:"incomplete_details,omitempty"`
}

type OutputItem struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content []OutputContent `json:"content,omitempty"`
}

type OutputContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type ResponseError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
