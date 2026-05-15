package provider

type Provider struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	AuthType     string  `json:"auth_type"`
	DefaultModel string  `json:"default_model"`
	Models       []Model `json:"models"`
}

type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
