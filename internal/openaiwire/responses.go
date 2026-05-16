package openaiwire

import "encoding/json"

type ResponsesStatus string

const (
	ResponsesStatusInProgress ResponsesStatus = "in_progress"
	ResponsesStatusCompleted  ResponsesStatus = "completed"
	ResponsesStatusFailed     ResponsesStatus = "failed"
)

type OutputItemType string

const (
	OutputItemTypeMessage      OutputItemType = "message"
	OutputItemTypeFunctionCall OutputItemType = "function_call"
	OutputItemTypeReasoning    OutputItemType = "reasoning"
)

type OutputContentType string

const (
	OutputContentTypeOutputText OutputContentType = "output_text"
)

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

type ResponsesResponse struct {
	ID                string          `json:"id"`
	Object            string          `json:"object"`
	CreatedAt         int64           `json:"created_at"`
	Status            ResponsesStatus `json:"status"`
	Model             string          `json:"model"`
	Output            []OutputItem    `json:"output"`
	Usage             *ResponseUsage  `json:"usage,omitempty"`
	Error             *ResponseError  `json:"error,omitempty"`
	IncompleteDetails any             `json:"incomplete_details,omitempty"`
}

type OutputItem struct {
	ID        string                `json:"id,omitempty"`
	Type      OutputItemType        `json:"type"`
	Role      string                `json:"role,omitempty"`
	Content   []OutputContent       `json:"content,omitempty"`
	CallID    string                `json:"call_id,omitempty"`
	Name      string                `json:"name,omitempty"`
	Arguments string                `json:"arguments,omitempty"`
	Summary   []ResponseSummaryPart `json:"summary,omitempty"`
}

type OutputContent struct {
	Type        OutputContentType `json:"type"`
	Text        string            `json:"text,omitempty"`
	Annotations []any             `json:"annotations,omitempty"`
}

type ResponseSummaryPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseUsage struct {
	InputTokens         int                         `json:"input_tokens"`
	OutputTokens        int                         `json:"output_tokens"`
	TotalTokens         int                         `json:"total_tokens"`
	InputTokensDetails  *ResponseInputTokenDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponseOutputTokenDetails `json:"output_tokens_details,omitempty"`
}

type ResponseInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type ResponseOutputTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type ResponseError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
