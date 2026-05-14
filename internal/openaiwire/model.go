package openaiwire

type Model struct {
	ID       string            `json:"id"`
	Object   string            `json:"object"`
	OwnedBy  string            `json:"owned_by"`
	Root     string            `json:"root,omitempty"`
	Parent   string            `json:"parent,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ListModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}
