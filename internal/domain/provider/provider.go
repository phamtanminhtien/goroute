package provider

type AuthType string

const (
	AuthTypeOAuth  AuthType = "oauth"
	AuthTypeAPIKey AuthType = "api_key"
)

type Provider struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	AuthType     AuthType `json:"auth_type"`
	Category     string   `json:"category"`
	DefaultModel string   `json:"default_model"`
	Models       []Model  `json:"models"`
}

type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
