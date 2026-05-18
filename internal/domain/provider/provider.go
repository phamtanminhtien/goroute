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
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	InputPricePerMillionUSD  float64 `json:"input_price_per_million_usd"`
	OutputPricePerMillionUSD float64 `json:"output_price_per_million_usd"`
	Source                   string  `json:"source"`
}

type ModelRecord struct {
	ID                       string  `json:"id" gorm:"primaryKey"`
	ProviderID               string  `json:"provider_id" gorm:"index;not null"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	InputPricePerMillionUSD  float64 `json:"input_price_per_million_usd"`
	OutputPricePerMillionUSD float64 `json:"output_price_per_million_usd"`
}

func (ModelRecord) TableName() string {
	return "provider_models"
}
