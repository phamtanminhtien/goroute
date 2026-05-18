package openaiwire

import (
	"encoding/json"
	"reflect"
)

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
	InputText    string              `json:"-"`
	Stream       bool                `json:"stream"`
	Store        bool                `json:"store"`
	Reasoning    any                 `json:"reasoning,omitempty"`
	Text         *ResponseText       `json:"text,omitempty"`
	Include      []string            `json:"include,omitempty"`
	RawBody      json.RawMessage     `json:"-"`
}

func (r *ResponsesRequest) UnmarshalJSON(data []byte) error {
	type alias struct {
		Model        string          `json:"model"`
		Instructions string          `json:"instructions"`
		Input        json.RawMessage `json:"input"`
		Stream       bool            `json:"stream"`
		Store        bool            `json:"store"`
		Reasoning    any             `json:"reasoning,omitempty"`
		Text         *ResponseText   `json:"text,omitempty"`
		Include      []string        `json:"include,omitempty"`
	}

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	r.Model = decoded.Model
	r.Instructions = decoded.Instructions
	r.Stream = decoded.Stream
	r.Store = decoded.Store
	r.Reasoning = decoded.Reasoning
	r.Text = decoded.Text
	r.Include = decoded.Include
	r.Input = nil
	r.InputText = ""

	if len(decoded.Input) == 0 || string(decoded.Input) == "null" {
		return nil
	}

	var inputText string
	if err := json.Unmarshal(decoded.Input, &inputText); err == nil {
		r.InputText = inputText
		return nil
	}

	var inputItems []ResponseInputItem
	if err := json.Unmarshal(decoded.Input, &inputItems); err == nil {
		r.Input = inputItems
		return nil
	}

	return &json.UnmarshalTypeError{
		Value: "input",
		Type:  reflect.TypeOf([]ResponseInputItem{}),
		Field: "input",
	}
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

func (p ResponseInputContentPart) MarshalJSON() ([]byte, error) {
	type payload struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		ImageURL string `json:"image_url,omitempty"`
		Detail   string `json:"detail,omitempty"`
	}
	type textPayload struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	switch p.Type {
	case "input_text", "output_text":
		return json.Marshal(textPayload{Type: p.Type, Text: p.Text})
	default:
		return json.Marshal(payload(p))
	}
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
	extraFields       map[string]json.RawMessage
}

func (r *ResponsesResponse) UnmarshalJSON(data []byte) error {
	type responseAlias ResponsesResponse

	var decoded responseAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var extras map[string]json.RawMessage
	if err := json.Unmarshal(data, &extras); err != nil {
		return err
	}

	*r = ResponsesResponse(decoded)
	r.extraFields = extras
	return nil
}

func (r ResponsesResponse) MarshalJSON() ([]byte, error) {
	type responseAlias ResponsesResponse

	encodedKnown, err := json.Marshal(responseAlias(r))
	if err != nil {
		return nil, err
	}

	var merged map[string]json.RawMessage
	if len(r.extraFields) > 0 {
		merged = cloneRawMap(r.extraFields)
	} else {
		merged = make(map[string]json.RawMessage)
	}
	if err := json.Unmarshal(encodedKnown, &merged); err != nil {
		return nil, err
	}

	return json.Marshal(merged)
}

type OutputItem struct {
	ID        string                `json:"id,omitempty"`
	Type      OutputItemType        `json:"type"`
	Status    string                `json:"status,omitempty"`
	Phase     string                `json:"phase,omitempty"`
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
	Annotations []any             `json:"annotations"`
	LogProbs    []any             `json:"logprobs"`
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
	extraFields         map[string]json.RawMessage
}

func (u *ResponseUsage) UnmarshalJSON(data []byte) error {
	type usageAlias ResponseUsage

	var decoded usageAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var extras map[string]json.RawMessage
	if err := json.Unmarshal(data, &extras); err != nil {
		return err
	}

	*u = ResponseUsage(decoded)
	u.extraFields = extras
	return nil
}

func (u ResponseUsage) MarshalJSON() ([]byte, error) {
	type usageAlias ResponseUsage

	normalized := usageAlias(u)
	if normalized.OutputTokensDetails == nil {
		normalized.OutputTokensDetails = &ResponseOutputTokenDetails{}
	}

	encodedKnown, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}

	var merged map[string]json.RawMessage
	if len(u.extraFields) > 0 {
		merged = cloneRawMap(u.extraFields)
	} else {
		merged = make(map[string]json.RawMessage)
	}
	if err := json.Unmarshal(encodedKnown, &merged); err != nil {
		return nil, err
	}

	return json.Marshal(merged)
}

type ResponseInputTokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type ResponseOutputTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type ResponseError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	dst := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		dst[key] = append(json.RawMessage(nil), value...)
	}
	return dst
}
